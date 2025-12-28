package operations

import (
	"fmt"
	"log"
	"os"
	"strings"

	"RancherMan/rancher"
	workload2 "RancherMan/rancher/types/workload"
	"gopkg.in/yaml.v3"
)

// CloneOrExportWorkload 克隆或导出工作负载
func CloneOrExportWorkload(isClone bool, destNamespace rancher.Namespace, tag string) {
	if isClone && destNamespace.Name == "" {
		gInfoArea.SetText("未选择目标命名空间")
		return
	}

	var info strings.Builder
	var allYaml strings.Builder // 用于存储所有workload的YAML

	processWorkloads := func(workloads []rancher.Workload) {
		for _, workload := range workloads {
			info.WriteString(fmt.Sprintf("获取deployment: %s    ", workload.Name))
			deployment, err := rancher.GetDeploymentYaml(*gEnvironment, workload.Namespace, workload.Name)
			if err == nil {
				info.WriteString("成功!\n")
				// 替换deployment名称中的namespace
				if destNamespace.Name != "" {
					deployment = strings.ReplaceAll(deployment, fmt.Sprintf(":\"%s:", workload.Namespace), fmt.Sprintf(":\"%s:", destNamespace.Name))
					deployment = strings.ReplaceAll(deployment, fmt.Sprintf("deployment-%s-", workload.Namespace), fmt.Sprintf("deployment-%s-", destNamespace.Name))
				}
				if tag != "" {
					// 检查workload是否在忽略列表中
					shouldUpdateTag := true
					for _, ignoreName := range gCloneIgnoreTagWorkload {
						if strings.Contains(workload.Name, ignoreName) {
							shouldUpdateTag = false
							break
						}
					}
					if shouldUpdateTag {
						// 从原始镜像名称中分离基础名称和标签
						baseImage := workload.Image
						if colonIndex := strings.LastIndex(workload.Image, ":"); colonIndex > 0 {
							baseImage = workload.Image[:colonIndex]
						}
						// 使用新标签替换
						deployment = strings.ReplaceAll(deployment, fmt.Sprintf("image: %s", workload.Image), fmt.Sprintf("image: %s:%s", baseImage, tag))
					}
				}
				// 解析yaml
				var deploymentStruct workload2.Deployment
				if err := yaml.Unmarshal([]byte(deployment), &deploymentStruct); err != nil {
					info.WriteString(fmt.Sprintf("解析deployment失败: %v\n", err))
					continue
				}
				// 如果nodeSelectorTerms为空,添加默认的node selector
				if len(deploymentStruct.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
					deploymentStruct.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = []workload2.NodeSelectorTerm{
						{
							MatchExpressions: []workload2.MatchExpression{
								{
									Key:      "role",
									Operator: "In",
									Values:   []string{"node"},
								},
							},
						},
					}
				}
				if destNamespace.Name != "" {
					deploymentStruct.Metadata.Namespace = destNamespace.Name
				}
				// 编码yaml
				yamlData, err := yaml.Marshal(deploymentStruct)
				if err != nil {
					info.WriteString(fmt.Sprintf("写入YAML失败: %v\n", err))
					continue
				}

				if isClone {
					// 克隆模式：导入到Rancher
					destEnvironment, _ := rancher.GetEnvironmentFromConfig(gConfig, destNamespace.Environment)
					err := rancher.ImportYaml(*destEnvironment, "big-data", yamlData)
					if err != nil {
						info.WriteString("克隆失败!\n")
					} else {
						info.WriteString("克隆成功!\n")
					}
				} else {
					allYaml.WriteString(fmt.Sprintf("# workload %s\n", workload.Name))
					// 导出模式：添加到YAML字符串
					allYaml.WriteString("---\n") // YAML文档分隔符
					allYaml.Write(yamlData)
					allYaml.WriteString("\n")
					info.WriteString("已添加到导出文件\n")
				}
			} else {
				info.WriteString("失败!\n")
			}
			gInfoArea.SetText(info.String())
		}
	}

	if len(gSelectedWorkloads) > 0 {
		processWorkloads(gSelectedWorkloads)
	} else if len(gFilteredWorkloads) > 0 {
		processWorkloads(gFilteredWorkloads)
	}

	// 如果是导出模式，将所有YAML写入文件
	if !isClone && allYaml.Len() > 0 {
		err := os.WriteFile("workloads.yaml", []byte(allYaml.String()), 0644)
		if err != nil {
			info.WriteString(fmt.Sprintf("\n导出到文件失败: %v", err))
		} else {
			info.WriteString("\n已成功导出到 workloads.yaml")
		}
	}

	gInfoArea.SetText(info.String())
}

// CloneOrExportConfigMap 克隆或导出ConfigMap
func CloneOrExportConfigMap(isClone bool, destNamespace rancher.Namespace) {
	if isClone && destNamespace.Name == "" {
		gInfoArea.SetText("未选择目标命名空间")
		return
	}

	var info strings.Builder
	var allYaml strings.Builder // 用于存储所有workload的YAML
	list, err := rancher.GetConfigMapList(*gEnvironment, gSelectedNamespace.Name)
	if err != nil {
		gInfoArea.SetText("获取配置时出错")
		return
	}

	for _, configMap := range list {
		info.WriteString(fmt.Sprintf("获取configMap: %s    ", configMap.Name))
		configMap.ApiVersion = "v1"
		configMap.Kind = "ConfigMap"
		configMap.Metadata.Name = configMap.Name
		if destNamespace.Name != "" {
			configMap.Metadata.Namespace = destNamespace.Name
		}

		// 编码yaml
		yamlData, err := yaml.Marshal(configMap)
		if err != nil {
			info.WriteString(fmt.Sprintf("写入YAML失败: %v\n", err))
			continue
		}

		if isClone {
			// 克隆模式：导入到Rancher
			destEnvironment, _ := rancher.GetEnvironmentFromConfig(gConfig, destNamespace.Environment)
			err := rancher.ImportYaml(*destEnvironment, "big-data", yamlData)
			if err != nil {
				info.WriteString("克隆失败!\n")
			} else {
				info.WriteString("克隆成功!\n")
			}
		} else {
			allYaml.WriteString(fmt.Sprintf("# configMap %s\n", configMap.Name))
			// 导出模式：添加到YAML字符串
			allYaml.WriteString("---\n") // YAML文档分隔符
			allYaml.Write(yamlData)
			allYaml.WriteString("\n")
			info.WriteString("已添加到导出文件\n")
		}
		gInfoArea.SetText(info.String())
	}

	// 如果是导出模式，将所有YAML写入文件
	if !isClone && allYaml.Len() > 0 {
		err := os.WriteFile("configMaps.yaml", []byte(allYaml.String()), 0644)
		if err != nil {
			info.WriteString(fmt.Sprintf("\n导出到文件失败: %v", err))
		} else {
			info.WriteString("\n已成功导出到 configMaps.yaml")
		}
	}

	gInfoArea.SetText(info.String())
}

// UpdateDataTask 创建更新数据任务
func UpdateDataTask(env *rancher.Environment, db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	if env == nil {
		log.Printf("[UpdateDataTask] ERROR: Environment is nil!")
		return
	}

	newTask := &rancher.Task{
		Type:        rancher.TaskTypeUpdateData,
		Description: fmt.Sprintf("更新数据: %s", env.Name),
		Environment: *env,
		DB:          db,
	}
	taskID := taskQueue.AddTask(newTask)
	log.Printf("[UpdateDataTask] Task added: ID=%d, Description=%s", taskID, newTask.Description)
	taskQueue.Submit(newTask)
}

// UpdatePortMapTask 创建更新端口映射任务
func UpdatePortMapTask(env *rancher.Environment, db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	if env == nil {
		log.Printf("[UpdatePortMapTask] ERROR: Environment is nil!")
		return
	}

	newTask := &rancher.Task{
		Type:        rancher.TaskTypeUpdatePortMap,
		Description: fmt.Sprintf("更新端口映射: %s", env.Name),
		Environment: *env,
		DB:          db,
	}
	taskID := taskQueue.AddTask(newTask)
	log.Printf("[UpdatePortMapTask] Task added: ID=%d, Description=%s", taskID, newTask.Description)
	taskQueue.Submit(newTask)
}

// ClearDataTask 创建清空数据任务
func ClearDataTask(db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	newTask := &rancher.Task{
		Type:        rancher.TaskTypeClearData,
		Description: "清空数据",
		DB:          db,
	}
	taskID := taskQueue.AddTask(newTask)
	log.Printf("[ClearDataTask] Task added: ID=%d, Description=%s", taskID, newTask.Description)
	taskQueue.Submit(newTask)
}
