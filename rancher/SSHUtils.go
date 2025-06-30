package rancher

import (
	"fmt"
	"path/filepath"
	"regexp"
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

// 提取镜像名称的函数，支持多种脚本格式
func extractImageName(shContent string, dirName string) string {
	lines := strings.Split(shContent, "\n")

	// 方法2: 解析使用变量的脚本格式
	var imageNameVar string

	// 查找 image_name 变量定义
	imageNameRegex := regexp.MustCompile(`image_name\s*=\s*\x60basename\s+\\\x60pwd\\\x60\x60`)
	for _, line := range lines {
		if imageNameRegex.MatchString(line) {
			imageNameVar = dirName // 使用目录名作为镜像名
			break
		}
	}

	// 如果没找到 image_name 变量，尝试其他可能的定义方式
	if imageNameVar == "" {
		imageNameRegex2 := regexp.MustCompile(`image_name\s*=.*`)
		for _, line := range lines {
			if imageNameRegex2.MatchString(line) {
				imageNameVar = dirName
				break
			}
		}
	}

	// 解析包含harbor地址的docker命令行，保留原始参数变量
	if imageNameVar != "" {
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// 查找包含 harbor 地址的 docker build 或 docker push 行
			if strings.Contains(line, "harbor.yunjingtech.cn:30002") &&
				(strings.Contains(line, "docker build") || strings.Contains(line, "docker push")) {

				result := strings.ReplaceAll(line, "$image_name", imageNameVar)

				// 从处理后的行中提取镜像地址部分
				// 查找 -t 参数后面的镜像地址
				if strings.Contains(result, " -t ") {
					parts := strings.Fields(result)
					for i, part := range parts {
						if part == "-t" && i+1 < len(parts) {
							return parts[i+1]
						}
					}
				}

				// 如果没有 -t 参数，查找 docker push 后面的镜像地址
				if strings.Contains(result, "docker push") {
					parts := strings.Fields(result)
					for i, part := range parts {
						if part == "push" && i+1 < len(parts) {
							return parts[i+1]
						}
					}
				}
			}
		}
	}
	// 方法1: 查找直接的 docker push 命令
	for _, line := range lines {
		if strings.Contains(line, "docker push") {
			parts := strings.Fields(line)
			if len(parts) > 2 {
				return parts[2]
			}
		}
	}

	return ""
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

				// 获取目录名称，用于解析镜像名称
				dirName := filepath.Base(strings.TrimPrefix(dir, "./"))
				if dirName == "." {
					// 如果是根目录，尝试从完整路径获取最后一个目录名
					pathParts := strings.Split(strings.TrimPrefix(dir, "./"), "/")
					if len(pathParts) > 0 && pathParts[len(pathParts)-1] != "" {
						dirName = pathParts[len(pathParts)-1]
					}
				}

				// 使用增强的镜像名称提取函数
				imageName := extractImageName(string(shContent), dirName)

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
