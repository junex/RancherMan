package rancher

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

type JumpHostConfig struct {
	Ip       string
	Port     string
	Username string
	Password string
	RootPath string
}

type SSHUploadConfig struct {
	Dir    string
	Script string
	Jar    string
	Image  string
}

// 添加进度监听器接口
type ProgressListener interface {
	OnProgress(currentFolder string, current, total int)
	OnBatchResult(configs []SSHUploadConfig)
	OnComplete()
}

func connectToJumpHost(config *JumpHostConfig) (*ssh.Client, error) {
	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%s", config.Ip, config.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("无法连接到跳板机: %v", err)
	}

	return client, nil
}

func ListUploadConfig(jumpHostConfig *JumpHostConfig, batchSize int, listener ProgressListener) {
	// 连接到跳板机
	client, err := connectToJumpHost(jumpHostConfig)
	if err != nil {
		fmt.Printf("连接跳板机失败: %v\n", err)
		return
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		fmt.Printf("创建SSH会话失败: %v\n", err)
		return
	}
	defer session.Close()

	// 方法1：使用完整的shell命令
	cmd := fmt.Sprintf("/bin/bash -c 'cd %s && find . -type f -name \"Dockerfile\" -o -name \"*.sh\" 2>/dev/null'",
		jumpHostConfig.RootPath)

	// 设置伪终端，模拟交互式shell
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	// 请求伪终端
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		fmt.Printf("请求PTY失败: %v\n", err)
		return
	}

	output, err := session.Output(cmd)
	if err != nil && len(output) == 0 {
		fmt.Printf("执行find命令失败: %v\n", err)
		fmt.Printf("错误输出: %s\n", string(output))
		return
	}

	// 如果有输出，继续处理，不管是否有错误
	if len(output) == 0 {
		fmt.Println("命令执行成功但没有输出")
		return
	}

	// 打印原始输出以便调试
	//fmt.Printf("命令原始输出:\n%s\n", string(output))

	// 解析输出,按目录分组文件
	files := make(map[string][]string)
	for _, file := range strings.Split(string(output), "\n") {
		if file == "" {
			continue
		}
		// 去掉可能存在的\r结尾
		file = strings.TrimSuffix(file, "\r")
		dir := filepath.ToSlash(filepath.Dir(file))
		filename := filepath.Base(file)

		files[dir] = append(files[dir], filename)
	}

	var configs []SSHUploadConfig
	totalDirs := len(files)
	processedDirs := 0

	// 遍历每个目录
	for dir, fileList := range files {
		processedDirs++
		if listener != nil {
			listener.OnProgress(dir, processedDirs, totalDirs)
		}

		hasDockerfile := false
		var shFiles []string

		for _, file := range fileList {
			if file == "Dockerfile" {
				hasDockerfile = true
			} else if strings.HasSuffix(file, ".sh") {
				shFiles = append(shFiles, file)
			}
		}

		// 如果目录同时包含Dockerfile和sh文件
		if hasDockerfile && len(shFiles) > 0 {
			// 读取Dockerfile内容
			session, err = client.NewSession()
			if err != nil {
				continue
			}
			dockerfileCmd := fmt.Sprintf("cat %s/%s/Dockerfile",
				jumpHostConfig.RootPath,
				strings.TrimPrefix(dir, "./"))
			dockerfileContent, err := session.Output(dockerfileCmd)
			if err != nil {
			} else {
			}
			session.Close()

			// 提取jar包名称
			var jarName string
			for _, line := range strings.Split(string(dockerfileContent), "\n") {
				if strings.Contains(line, "COPY") && strings.Contains(line, ".jar") {
					parts := strings.Fields(line)
					for _, part := range parts {
						if strings.HasSuffix(part, ".jar") {
							jarName = part
							break
						}
					}
				}
			}

			// 处理每个sh文件
			for _, shFile := range shFiles {
				session, err = client.NewSession()
				if err != nil {
					continue
				}
				shCmd := fmt.Sprintf("cat %s/%s/%s",
					jumpHostConfig.RootPath,
					strings.TrimPrefix(dir, "./"),
					shFile)
				shContent, _ := session.Output(shCmd)
				session.Close()

				// 提取镜像名称
				var imageName string
				for _, line := range strings.Split(string(shContent), "\n") {
					if strings.Contains(line, "docker push") {
						parts := strings.Fields(line)
						// 获取docker push后面的第一个参数作为镜像名称
						if len(parts) > 2 {
							imageName = parts[2]
							break
						}
					}
				}

				if jarName != "" && imageName != "" {
					configs = append(configs, SSHUploadConfig{
						Dir:    filepath.Join(jumpHostConfig.RootPath, dir),
						Script: shFile,
						Jar:    jarName,
						Image:  imageName,
					})
				}
			}
		}

		// 当收集到足够的配置时，触发批量结果回调
		if len(configs) >= batchSize {
			if listener != nil {
				listener.OnBatchResult(configs)
			}
			configs = []SSHUploadConfig{} // 清空已处理的配置
		}
	}

	// 处理最后剩余的配置
	if len(configs) > 0 && listener != nil {
		listener.OnBatchResult(configs)
	}

	// 添加完成回调
	if listener != nil {
		listener.OnComplete()
	}
}
