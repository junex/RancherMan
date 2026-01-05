package operations

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
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
			deployment, err := rancher.GetDeploymentYaml(context.Background(), *gEnvironment, workload.Namespace, workload.Name)
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
					err := rancher.ImportYaml(context.Background(), *destEnvironment, "big-data", yamlData)
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
	list, err := rancher.GetConfigMapList(context.Background(), *gEnvironment, gSelectedNamespace.Name)
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
			err := rancher.ImportYaml(context.Background(), *destEnvironment, "big-data", yamlData)
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
	taskQueue.AddTask(newTask)
	UpdatePortMapTask(env, db, taskQueue)
	UpdatePodsTask(env, db, taskQueue)
}

func UpdateDataCompleteTask(taskQueue *rancher.TaskQueue) {
	newTask := &rancher.Task{
		Type:         rancher.TaskTypeUpdateDataComplete,
		Description:  fmt.Sprintf("更新数据完成"),
		UnCancelable: true,
	}
	taskQueue.AddTask(newTask)
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
	taskQueue.AddTask(newTask)
}

// ClearDataTask 创建清空数据任务
func ClearDataTask(db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	newTask := &rancher.Task{
		Type:        rancher.TaskTypeClearData,
		Description: "清空数据",
		DB:          db,
	}
	taskQueue.AddTask(newTask)
}

func UpdatePodsTask(env *rancher.Environment, db *rancher.DatabaseManager, taskQueue *rancher.TaskQueue) {
	if env == nil {
		log.Printf("[UpdatePodsTask] ERROR: Environment is nil!")
		return
	}
	newTask := &rancher.Task{
		Type:        rancher.TaskTypeUpdatePod,
		Description: fmt.Sprintf("更新Pod: %s", env.Name),
		Environment: *env,
		DB:          db,
	}
	taskQueue.AddTask(newTask)
}

func OpenWorkloadTask(env *rancher.Environment, db *rancher.DatabaseManager, workloads []rancher.Workload, taskQueue *rancher.TaskQueue) {
	hasDependDatabaseWorkload := false
	for _, w := range workloads {
		priority, _ := workloadPriority(w.Name)
		if priority == 4 {
			hasDependDatabaseWorkload = true
		}
	}

	SortWorkloadsByPriority(workloads, SortAsc)
	for i, workload := range workloads {
		newTask := &rancher.Task{
			Type:        rancher.TaskTypeScaleOpen,
			Description: fmt.Sprintf("打开 %s", workload.Name),
			Environment: *env,
			Namespace:   workload.Namespace,
			Workload:    workload.Name,
			Replicas:    1,
		}
		taskQueue.AddTask(newTask)
		if hasDependDatabaseWorkload {
			_, delayMs := workloadPriority(workload.Name)
			if i < len(workloads)-1 {
				DelayTask(newTask.Description+"后", delayMs, taskQueue)
			} else {
				DelayTask(newTask.Description+"后", 500, taskQueue)
			}
		}
	}
}

func CloseWorkloadTask(env *rancher.Environment, db *rancher.DatabaseManager, workloads []rancher.Workload, taskQueue *rancher.TaskQueue) {
	SortWorkloadsByPriority(workloads, SortDesc)
	for _, workload := range workloads {
		newTask := &rancher.Task{
			Type:        rancher.TaskTypeScaleClose,
			Description: fmt.Sprintf("关闭 %s", workload.Name),
			Environment: *env,
			Namespace:   workload.Namespace,
			Workload:    workload.Name,
			Replicas:    0,
		}
		taskQueue.AddTask(newTask)
	}
}

func RedeployWorkloadTask(env *rancher.Environment, db *rancher.DatabaseManager, workloads []rancher.Workload, taskQueue *rancher.TaskQueue) {
	hasDependDatabaseWorkload := false
	for i := range workloads {
		workload := workloads[i]
		priority, _ := workloadPriority(workload.Name)
		if priority == 4 {
			hasDependDatabaseWorkload = true
		}
	}
	SortWorkloadsByPriority(workloads, SortAsc)
	for i, workload := range workloads {
		newTask := &rancher.Task{
			Type:        rancher.TaskTypeRedeploy,
			Description: fmt.Sprintf("重新部署 %s", workload.Name),
			Environment: *env,
			Namespace:   workload.Namespace,
			Workload:    workload.Name,
		}
		taskQueue.AddTask(newTask)
		if hasDependDatabaseWorkload {
			_, delayMs := workloadPriority(workload.Name)
			if i < len(workloads)-1 {
				DelayTask(newTask.Description+"后", delayMs, taskQueue)
			}
		} else {
			DelayTask(newTask.Description+"后", 500, taskQueue)
		}
	}
}

func DelayTask(reason string, delayMs int, taskQueue *rancher.TaskQueue) {
	if delayMs > 0 {
		newTask := &rancher.Task{
			Type:        rancher.TaskTypeDelay,
			DelayMs:     delayMs,
			Description: fmt.Sprintf("%s 等待 %.1f 秒", reason, float32(delayMs)/float32(1000)),
		}
		taskQueue.AddTask(newTask)
	}
}

type SortOrder int

const (
	SortAsc  SortOrder = iota // 升序
	SortDesc                  // 降序
)

func SortWorkloadsByPriority(
	workloads []rancher.Workload,
	order SortOrder,
) {
	sort.SliceStable(workloads, func(i, j int) bool {
		pi, _ := workloadPriority(workloads[i].Name)
		pj, _ := workloadPriority(workloads[j].Name)

		// 先按优先级
		if pi != pj {
			if order == SortAsc {
				return pi < pj
			}
			return pi > pj
		}

		// 同优先级按名称
		if order == SortAsc {
			return workloads[i].Name < workloads[j].Name
		}
		return workloads[i].Name > workloads[j].Name
	})
	for i := range workloads {
		fmt.Printf("已处理 %s\n", workloads[i].Name)
	}
}

var workloadRules = []struct {
	priority int
	keywords []string
	delayMs  int
}{
	// 这个顺序是为了最后匹配数据库，防止错误匹配
	{priority: 2, keywords: []string{"redis", "mongo", "elasticsearch", "rabbitmq", "kafka", "minio"}, delayMs: 1000},
	{priority: 3, keywords: []string{"web-"}, delayMs: 1000},
	{priority: 4, keywords: []string{"-portal", "-api", "xxl-job", "one-travel", "sot-"}, delayMs: 3000},
	{priority: 1, keywords: []string{"mysql", "dm", "kingbase"}, delayMs: 10000},
}

func workloadPriority(name string) (int, int) {
	name = strings.ToLower(name)

	for _, rule := range workloadRules {
		for _, kw := range rule.keywords {
			if strings.Contains(name, kw) {
				return rule.priority, rule.delayMs
			}
		}
	}
	return 100, 0
}
