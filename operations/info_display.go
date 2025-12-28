package operations

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"RancherMan/rancher"
)

// UpdateInfoArea 根据选择的状态更新信息区域
func UpdateInfoArea() {
	if len(gSelectedWorkloads) == 0 && gSelectedNamespace.Name == "" {
		gInfoArea.SetText("")
	} else if len(gSelectedWorkloads) == 0 {
		updateInfoAreaForSelectNamespace()
	} else if len(gSelectedWorkloads) == 1 {
		updateInfoAreaForSingleWorkload()
	} else {
		updateInfoAreaForSelectMultiWorkload()
	}
}

// updateInfoAreaForSelectNamespace 更新选择命名空间时的信息显示
func updateInfoAreaForSelectNamespace() {
	podList, _ := gDb.GetPodsByEnvNamespace(gSelectedNamespace.Environment, gSelectedNamespace.Name)

	var info strings.Builder
	// todo 修复gEnvironment偶尔为null导致闪退
	info.WriteString(fmt.Sprintf("环境: %s\n", gEnvironment.Name))
	info.WriteString(fmt.Sprintf("命名空间: %s\n", gSelectedNamespace.Name))
	info.WriteString(fmt.Sprintf("项目: %s\n", gSelectedNamespace.Project))
	info.WriteString(fmt.Sprintf("描述: %s\n", gSelectedNamespace.Description))
	info.WriteString(fmt.Sprintf("pod数量: %d\n", len(podList)))
	// 创建一个map来存储相同workloadId的pod状态
	podStates := make(map[string][]string)
	for _, pod := range podList {
		// 获取workloadId的最后一部分
		parts := strings.Split(pod.WorkloadId, ":")
		workloadName := parts[len(parts)-1]
		podStates[workloadName] = append(podStates[workloadName], pod.State)
	}

	// 打印每个workload的pod状态
	for workloadName, states := range podStates {
		info.WriteString(fmt.Sprintf("%s: %s\n", workloadName, strings.Join(states, ",")))
	}
	gInfoArea.SetText(info.String())
}

// updateInfoAreaForSingleWorkload 更新选择单个工作负载时的信息显示
func updateInfoAreaForSingleWorkload() {
	workload := gSelectedWorkloads[0]
	podList, _ := gDb.GetPodsByEnvNamespaceWorkload(workload.Environment, workload.Namespace, workload.Name)

	// 构建信息字符串
	var info strings.Builder

	info.WriteString(fmt.Sprintf("环境: %s\n", workload.Environment))
	info.WriteString(fmt.Sprintf("命名空间: %s\n", workload.Namespace))
	info.WriteString(fmt.Sprintf("名称: %s\n", workload.Name))
	info.WriteString(fmt.Sprintf("镜像: %s\n", workload.Image))
	info.WriteString(fmt.Sprintf("镜像拉取策略: %s\n", workload.ImagePullPolicy))
	info.WriteString(fmt.Sprintf("pod数量: %d\n", len(podList)))
	if len(podList) > 0 {
		var states []string
		for _, pod := range podList {
			states = append(states, pod.State)
		}
		info.WriteString(fmt.Sprintf("Pod状态: %s\n", strings.Join(states, ",")))
	}
	// 检查数据库相关的环境变量
	if (strings.Contains(strings.ToLower(workload.Name), "mysql") ||
		strings.Contains(strings.ToLower(workload.Name), "mongo")) &&
		workload.ContainerEnvironment != "" {
		var envVars map[string]string
		if err := json.Unmarshal([]byte(workload.ContainerEnvironment), &envVars); err == nil {
			// 检查并输出每个可能存在的环境变量
			if password, exists := envVars["MYSQL_ROOT_PASSWORD"]; exists {
				info.WriteString(fmt.Sprintf("MySQL Root密码: %s\n", password))
			}
			if username, exists := envVars["MONGO_INITDB_ROOT_USERNAME"]; exists {
				info.WriteString(fmt.Sprintf("MongoDB初始化Root用户名: %s\n", username))
			}
			if password, exists := envVars["MONGO_INITDB_ROOT_PASSWORD"]; exists {
				info.WriteString(fmt.Sprintf("MongoDB初始化Root密码: %s\n", password))
			}
		}
	}
	// 检查是否有备注，有就展示
	if strings.TrimSpace(workload.Remark) != "" {
		info.WriteString(fmt.Sprintf("备注: \n%s\n", workload.Remark))
	}
	services, err := gDb.GetServicesByWorkload(workload.Environment, workload.ProjectId, workload.Namespace, workload.Name)
	if err == nil && len(services) > 0 {
		info.WriteString("端口访问:\n")
		ip := gEnvironment.Ip
		for _, port := range services {
			if port.Kind == "NodePort" {
				info.WriteString(fmt.Sprintf("  %s    %s    %d->%s:%d\n", port.PortName, port.PortProtocol, port.Port, ip, port.NodePort))
			} else {
				info.WriteString(fmt.Sprintf("  %s    %s    %d\n", port.PortName, port.PortProtocol, port.Port))
			}
		}
	}
	// 只有当 AccessPath 不为空时才显示，并按逗号分隔成多行
	if workload.AccessPath != "" {
		info.WriteString("访问路径:\n")
		paths := strings.Split(workload.AccessPath, ",")
		for _, path := range paths {
			info.WriteString(fmt.Sprintf("  %s\n", strings.TrimSpace(path)))
		}
	}
	var uploadConfigList []rancher.UploadConfig
	// 获取完整镜像名称的配置
	configs, _ := gDb.GetUploadConfigsByImage(workload.Image)
	uploadConfigList = append(uploadConfigList, configs...)

	// 获取不带标签的镜像名称的配置
	image := workload.Image
	tag := ""
	if colonIndex := strings.LastIndex(workload.Image, ":"); colonIndex > 0 {
		image = workload.Image[:colonIndex]
		tag = workload.Image[colonIndex+1:]
	}
	configs1, _ := gDb.GetUploadConfigsByImageLikeSpecial1(image)
	uploadConfigList = append(uploadConfigList, configs1...)
	// 获取最后两个/之间的部分
	imageDir := ""
	if strings.Count(image, "/") >= 2 {
		lastSlashIndex := strings.LastIndex(image, "/")
		lastTwoSlashIndex := strings.LastIndex(image[:lastSlashIndex], "/")
		if lastTwoSlashIndex > 0 {
			imageDir = image[lastTwoSlashIndex+1 : lastSlashIndex]
		}
	}
	if lastSlashIndex := strings.LastIndex(image, "/"); lastSlashIndex >= 0 {
		image = image[lastSlashIndex+1:]
	}
	configs2, _ := gDb.GetUploadConfigsByImageLikeSpecial2(image)
	uploadConfigList = append(uploadConfigList, configs2...)
	// 对uploadConfigList进行排序
	sort.Slice(uploadConfigList, func(i, j int) bool {
		// 优先条件：Image包含"$image_name"排前面
		containsImageNameI := strings.Contains(uploadConfigList[i].Image, "$image_name")
		containsImageNameJ := strings.Contains(uploadConfigList[j].Image, "$image_name")

		if containsImageNameI != containsImageNameJ {
			// 谁包含谁在前
			return containsImageNameI
		}

		// 如果两边都包含或都不包含，继续后续逻辑
		// 获取$符号数量
		dollarCountI := strings.Count(uploadConfigList[i].Image, "$")
		dollarCountJ := strings.Count(uploadConfigList[j].Image, "$")

		// 如果$数量不同,按数量升序排序
		if dollarCountI != dollarCountJ {
			return dollarCountI < dollarCountJ
		}

		// 如果$数量相同,检查namespace中的部分是否包含在Dir中
		parts := strings.Split(workload.Namespace, "-")
		if len(parts) >= 3 {
			// 取两个-号之间的部分
			middlePart := parts[1]
			containsI := strings.Contains(uploadConfigList[i].Dir, middlePart)
			containsJ := strings.Contains(uploadConfigList[j].Dir, middlePart)

			// 包含middlePart的排在前面
			if containsI != containsJ {
				return containsI
			}
		}

		// 其他情况保持原有顺序
		return i < j
	})
	// 如果有上传配置，则显示
	if len(uploadConfigList) > 0 {
		info.WriteString("\n上传配置:\n")
		for _, config := range uploadConfigList {
			info.WriteString(fmt.Sprintf("  目录: %s\n", strings.ReplaceAll(config.Dir, "\\", "/")))
			if config.Script != "" {
				var script = config.Script
				dollarCount := strings.Count(config.Image, "$")
				if dollarCount == 1 {
					script = script + " " + tag
				} else if dollarCount == 2 {
					script = script + " " + tag + " " + imageDir
				} else if dollarCount == 3 && strings.Contains(config.Image, "$image_name") {
					script = script + " " + tag + " " + imageDir
				}
				info.WriteString(fmt.Sprintf("  脚本: ./%s\n", script))
			}
			if config.Jar != "" {
				info.WriteString(fmt.Sprintf("  Jar包: %s\n", config.Jar))
			}
			if config.Image != "" {
				info.WriteString(fmt.Sprintf("  镜像: %s\n", config.Image))
			}
			info.WriteString("\n")
		}
	}
	gInfoArea.SetText(info.String())
}

// updateInfoAreaForSelectMultiWorkload 更新选择多个工作负载时的信息显示
func updateInfoAreaForSelectMultiWorkload() {
	var info strings.Builder
	info.WriteString(fmt.Sprintf("已选择 %d 个服务:\n", len(gSelectedWorkloads)))

	for _, workload := range gSelectedWorkloads {
		info.WriteString(fmt.Sprintf("\n服务名称: %s\n", workload.Name))
		info.WriteString(fmt.Sprintf("镜像: %s\n", workload.Image))
	}

	gInfoArea.SetText(info.String())
}
