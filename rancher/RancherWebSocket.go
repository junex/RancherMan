package rancher

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 生成唯一ID的函数
func generateSockId() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

type PodEventHandler struct {
	OnPodsChanged func()
}

type ResourceWebSocket struct {
	environment          Environment
	db                   *DatabaseManager
	podEventHandler      PodEventHandler
	conn                 *websocket.Conn
	connMutex            sync.RWMutex
	reconnectAttempts    int
	reconnectMutex       sync.Mutex // 保护 reconnectAttempts
	maxReconnectAttempts int
	reconnectDelay       time.Duration
	ctx                  context.Context
	cancel               context.CancelFunc
	wg                   sync.WaitGroup
	disconnectOnce       *sync.Once
}

type Message struct {
	Name string `json:"name"`
}

type PodMessage struct {
	Message
	Pod PodResp `json:"data"`
}

func NewRancherWebSocket(environment Environment, db *DatabaseManager, podEventHandler PodEventHandler) *ResourceWebSocket {
	ctx, cancel := context.WithCancel(context.Background())
	return &ResourceWebSocket{
		environment:          environment,
		db:                   db,
		podEventHandler:      podEventHandler,
		maxReconnectAttempts: 3,
		reconnectDelay:       time.Second,
		ctx:                  ctx,
		cancel:               cancel,
		disconnectOnce:       &sync.Once{},
	}
}

func buildWSURL(base string, project string) (string, error) {
	// 解析解码后的 URL（这样能正确拿到 Host 和 Path）
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	basePath := strings.TrimPrefix(parsed.Path, "/") // "v3"

	// 拼接最终路径：/v3/projects/<project>/subscribe
	finalPath := "/" + path.Join(basePath, "projects", project, "subscribe")

	u := url.URL{
		Scheme: "wss",
		Host:   parsed.Host,
		Path:   finalPath,
		RawQuery: url.Values{
			"sockId": {generateSockId()},
		}.Encode(),
	}

	return u.String(), nil
}

func (ws *ResourceWebSocket) Connect() error {
	wsurl, err := buildWSURL(ws.environment.BaseURL, ws.environment.Project)
	if err != nil {
		return fmt.Errorf("WebSocket连接失败: %v", err)
	}

	log.Printf("正在连接WebSocket: %s", wsurl)

	ctx, cancel := context.WithTimeout(ws.ctx, 5*time.Second)
	defer cancel()

	// 创建带 Cookie 的 HTTP Header
	header := http.Header{}
	if ws.environment.username != "" && ws.environment.password != "" {
		auth := ws.environment.username + ":" + ws.environment.password
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
		header.Set("Authorization", basicAuth)
	}

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true}}
	c, resp, err := dialer.DialContext(ctx, wsurl, header)
	if err != nil {
		if resp != nil {
			fmt.Printf("websocket dial failed, status=%d, statusText=%s, err=%v\n", resp.StatusCode, resp.Status, err)
		} else {
			fmt.Printf("websocket dial failed, no response, err=%v\n", err)
		}
		return err
	}

	// 加锁设置连接
	ws.connMutex.Lock()
	ws.conn = c
	ws.connMutex.Unlock()

	// 重置重连计数
	ws.reconnectMutex.Lock()
	ws.reconnectAttempts = 0
	ws.reconnectMutex.Unlock()

	// 重置 disconnectOnce，允许新连接的断开处理
	ws.disconnectOnce = &sync.Once{}

	// 启动消息接收协程
	ws.wg.Add(1)
	go ws.readMessages()

	return nil
}

func (ws *ResourceWebSocket) readMessages() {
	defer ws.wg.Done()

	for {
		select {
		case <-ws.ctx.Done():
			// 主动关闭，直接返回
			log.Println("WebSocket读取循环因主动关闭而退出")
			return
		default:
			ws.connMutex.RLock()
			conn := ws.conn
			ws.connMutex.RUnlock()

			if conn == nil {
				log.Println("连接为空，退出读取循环")
				return
			}
			_, message, err := conn.ReadMessage()
			if err != nil {
				// 再次检查是否是主动关闭
				select {
				case <-ws.ctx.Done():
					// 主动关闭导致的错误
					log.Printf("读取消息时检测到主动关闭: %v", err)
					return
				default:
					// 真正的连接错误
					log.Printf("读取消息出错: %v", err)
					ws.handleDisconnect(err)
					return
				}
			}

			ws.handleMessage(message)
		}
	}
}

func (ws *ResourceWebSocket) handleMessage(message []byte) {
	//msgStr := string(message)
	//log.Printf("WebSocket消息: %s", msgStr)

	// 尝试 JSON 解析
	var payload PodMessage
	if err := json.Unmarshal(message, &payload); err != nil {
		log.Printf("JSON 解析失败: %v", err)
		return
	}

	if !strings.HasPrefix(payload.Name, "resource.") {
		return
	}
	if payload.Pod.BaseType != "pod" || payload.Pod.Type != "pod" {
		return
	}

	if payload.Name == "resource.change" {
		pod := Pod{
			ID:          payload.Pod.ID,
			Environment: ws.environment.ID,
			ProjectId:   payload.Pod.ProjectID,
			NamespaceId: payload.Pod.NamespaceID,
			WorkloadId:  payload.Pod.WorkloadID,
			State:       payload.Pod.State,
		}
		ws.db.SaveOrUpdatePodByID(&pod)
	} else if payload.Name == "resource.remove" {
		ws.db.DeletePodByID(payload.Pod.ID)
	}

	// 调用事件处理器
	if ws.podEventHandler.OnPodsChanged != nil {
		ws.podEventHandler.OnPodsChanged()
	}
}

func (ws *ResourceWebSocket) handleDisconnect(err error) {
	// sync.Once 确保只执行一次
	ws.disconnectOnce.Do(func() {
		log.Printf("处理断开连接: %v", err)

		ws.connMutex.Lock()
		if ws.conn != nil {
			ws.conn.Close()
			ws.conn = nil
		}
		ws.connMutex.Unlock()

		// 检查是否需要重连（只有真正的错误才重连）
		ws.reconnectMutex.Lock()
		shouldReconnect := err != nil &&
			ws.reconnectAttempts < ws.maxReconnectAttempts &&
			ws.ctx.Err() == nil
		ws.reconnectMutex.Unlock()

		if shouldReconnect {
			go ws.reconnect()
		}
	})
}

func (ws *ResourceWebSocket) reconnect() {
	ws.reconnectMutex.Lock()
	ws.reconnectAttempts++
	attempt := ws.reconnectAttempts
	ws.reconnectMutex.Unlock()

	// 指数退避
	delay := ws.reconnectDelay * time.Duration(1<<uint(attempt-1))
	if delay > 30*time.Second {
		delay = 30 * time.Second
	}

	log.Printf("尝试重连 %d/%d，等待 %v...", attempt, ws.maxReconnectAttempts, delay)

	// 使用可中断的 sleep
	select {
	case <-time.After(delay):
	case <-ws.ctx.Done():
		return
	}

	if err := ws.Connect(); err != nil {
		log.Printf("重连失败: %v", err)
		// 重连失败后，handleDisconnect 会再次被调用，继续尝试
		ws.handleDisconnect(err)
	} else {
		log.Println("重连成功")
	}
}

func (ws *ResourceWebSocket) Close() {
	log.Println("正在关闭WebSocket连接...")

	// 1. 取消 context，停止所有操作
	ws.cancel()

	// 2. 关闭 WebSocket 连接（会导致 ReadMessage 返回错误）
	ws.connMutex.Lock()
	if ws.conn != nil {
		ws.conn.Close()
		ws.conn = nil
	}
	ws.connMutex.Unlock()

	// 3. 等待所有 goroutine 退出
	ws.wg.Wait()

	log.Println("WebSocket连接已关闭")
}
