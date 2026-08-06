package rancher

import (
	"regexp"
	"strconv"
	"strings"
)

// ConfigEntry 表示解析后的配置条目
type ConfigEntry struct {
	BaseURL      string
	LocationPath string
	ServerName   string
	Domain       string
	Port         int
}

// LookupDict 用于服务查找的字典类型
type LookupDict map[ServiceKey]string

// ServiceKey 用于标识服务的键
type ServiceKey struct {
	Service   string
	Namespace string
}

// ParseNginxConfig 解析Nginx配置文件
func ParseNginxConfig(baseURL string, configText string) ([]ConfigEntry, error) {
	// 移除注释和空行
	lines := make([]string, 0)
	for _, line := range strings.Split(configText, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}

	// 将配置文件合并成一个字符串
	configStr := strings.Join(lines, " ")

	// 编译正则表达式
	locRegex := regexp.MustCompile(`location\s+([^{]+?)\s*\{`)
	proxyRegex := regexp.MustCompile(`proxy_pass\s+http://([^:/]+)[.]([^:/]+):(\d+)`)

	results := make([]ConfigEntry, 0)
	searchStart := 0

	for {
		// 找到下一个 location 指令
		locMatch := locRegex.FindStringSubmatchIndex(configStr[searchStart:])
		if locMatch == nil {
			break
		}

		locationPath := strings.TrimSpace(configStr[searchStart+locMatch[2] : searchStart+locMatch[3]])

		// locMatch[1] 是完整匹配结束位置，最后一个字符就是 {
		bracePos := searchStart + locMatch[1] - 1

		// 括号计数，找到匹配的 }
		depth := 0
		contentStart := bracePos + 1
		i := bracePos
	loop:
		for i < len(configStr) {
			switch configStr[i] {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					break loop
				}
			}
			i++
		}

		if depth != 0 {
			// 没找到匹配的 }，跳过这个异常的 location
			searchStart = bracePos + 1
			continue
		}

		blockContent := strings.TrimSpace(configStr[contentStart:i])

		// 在块内容中查找proxy_pass
		proxyMatch := proxyRegex.FindStringSubmatch(blockContent)
		if proxyMatch != nil {
			serverName := proxyMatch[1]
			domain := proxyMatch[2]
			port, err := strconv.Atoi(proxyMatch[3])
			if err != nil {
				searchStart = i + 1
				continue // 跳过无效的端口号
			}

			entry := ConfigEntry{
				BaseURL:      baseURL,
				LocationPath: locationPath,
				ServerName:   serverName,
				Domain:       domain,
				Port:         port,
			}
			results = append(results, entry)
		}

		// 从 { 后继续搜索，确保嵌套 location 也能被找到
		searchStart = bracePos + 1
	}

	return results, nil
}

// CreateLookupDict 创建查找字典
func CreateLookupDict(configList []ConfigEntry) LookupDict {
	lookup := make(LookupDict)

	for _, entry := range configList {
		key := ServiceKey{
			Service:   entry.ServerName,
			Namespace: entry.Domain,
		}

		value := entry.BaseURL + entry.LocationPath

		// 如果键已存在，追加值
		if existingValue, exists := lookup[key]; exists {
			lookup[key] = existingValue + "," + value
		} else {
			lookup[key] = value
		}
	}

	return lookup
}

// LookupService 查找服务对应的URL
func LookupService(lookupDict LookupDict, service, namespace string) string {
	key := ServiceKey{
		Service:   service,
		Namespace: namespace,
	}
	return lookupDict[key]
}
