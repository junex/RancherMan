package main

import (
	"RancherMan/rancher"
	workload2 "RancherMan/rancher/types/workload"
	"RancherMan/ui"
	"RancherMan/ui/component"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v3"
)

// 数据
var gNamespaces []rancher.Namespace
var gFilteredNamespaces []rancher.Namespace
var gSelectedNamespace rancher.Namespace
var gWorkloads []rancher.Workload
var gFilteredWorkloads []rancher.Workload
var gSelectedWorkloads []rancher.Workload

// UI组件
var gNamespaceList *widget.List
var gNamespaceSearch *widget.Entry
var gWorkloadList *component.MultiSelectList
var gWorkloadSearch *widget.Entry
var gInfoArea *widget.Entry
var gApp fyne.App

// 数据库和配置
var gDb *rancher.DatabaseManager
var gConfig map[string]interface{}
var gEnvironment *rancher.Environment
var gJumpHostConfig *rancher.JumpHostConfig

func main() {
	//// 创建数据库管理器实例
	database, err := rancher.NewDatabaseManager("")
	if err != nil {
		log.Fatal(err)
	}
	gDb = database
	defer gDb.Close()
	window := initView()
	loadConfig(false)
	initData()
	window.ShowAndRun()
}
func initView() fyne.Window {
	//// 初始化界面
	gApp = app.New()
	myWindow := gApp.NewWindow("Rancher助手")

	// 创建主菜单
	mainMenu := fyne.NewMainMenu(
		fyne.NewMenu("配置",
			fyne.NewMenuItem("保存配置", func() {
				var content = gInfoArea.Text
				rancher.SaveConfigToDb(gDb, content)
				loadConfig(true)
				initData()
			}),
			fyne.NewMenuItem("显示配置", func() {
				configContent, _ := gDb.GetConfigContent(1)
				gInfoArea.SetText(configContent)
			}),
		),
		fyne.NewMenu("数据",
			fyne.NewMenuItem("更新跳板机", func() {
				if gJumpHostConfig == nil {
					gInfoArea.SetText("错误：未配置跳板机信息")
					return
				}

				// 创建进度监听器
				listener := &jumpHostProgressListener{
					infoArea: gInfoArea,
				}

				// 清空信息区域并显示初始信息
				gInfoArea.SetText("开始扫描跳板机配置...\n")

				// 在新的 goroutine 中执行耗时操作
				go func() {
					gDb.DeleteAllUploadConfigs()
					rancher.ListUploadConfig(gJumpHostConfig, 50, listener)
				}()
			}),
			fyne.NewMenuItem("更新数据", func() {
				var info strings.Builder
				if gEnvironment != nil {
					// 只更新当前选中的环境
					info.WriteString(fmt.Sprintf("更新数据: %s ", gEnvironment.Name))
					gInfoArea.SetText(info.String())
					rancher.UpdateEnvironment(gDb, gEnvironment.ID, gEnvironment, true)
					info.WriteString("完成!\n")
					gInfoArea.SetText(info.String())
				} else {
					// 如果没有选中环境，则更新所有环境
					for envName, _ := range gConfig["environment"].(map[interface{}]interface{}) {
						environment, _ := rancher.GetEnvironmentFromConfig(gConfig, envName.(string))
						info.WriteString(fmt.Sprintf("更新数据: %s ", environment.Name))
						gInfoArea.SetText(info.String())
						rancher.UpdateEnvironment(gDb, environment.ID, environment, true)
						info.WriteString("完成!\n")
						gInfoArea.SetText(info.String())
					}
				}
				initData()
			}),
			fyne.NewMenuItem("清空数据", func() {
				err := gDb.ClearAllData()
				if err != nil {
					gInfoArea.SetText(fmt.Sprintf("清空数据失败: %v", err))
				} else {
					gInfoArea.SetText("数据已清空")
				}

				initData()
			}),
		),
		fyne.NewMenu("克隆和导出",
			fyne.NewMenuItem("导出configMap", func() {
				ui.ShowSelectNamespaceDialog(myWindow, gDb, func(destNamespace rancher.Namespace) {
					cloneOrExportConfigMap(false, destNamespace)
				})
			}),
			fyne.NewMenuItem("克隆configMap", func() {
				ui.ShowSelectNamespaceDialog(myWindow, gDb, func(destNamespace rancher.Namespace) {
					cloneOrExportConfigMap(false, destNamespace)
				})
			}),
			fyne.NewMenuItem("导出workload", func() {
				ui.ShowSelectNamespaceDialog(myWindow, gDb, func(destNamespace rancher.Namespace) {
					cloneOrExportWorkload(false, destNamespace)
				})
			}),
			fyne.NewMenuItem("克隆workload", func() {
				ui.ShowSelectNamespaceDialog(myWindow, gDb, func(destNamespace rancher.Namespace) {
					cloneOrExportWorkload(true, destNamespace)
				})
			}),
		),
		fyne.NewMenu("帮助",
			fyne.NewMenuItem("关于", func() {
				dialog.ShowInformation("关于",
					"Rancher助手 v1.0\n\n"+
						"一个用于管理Rancher工作负载的工具\n"+
						"作者: 六月盒饭\n"+
						"版权所有 2024",
					myWindow)
			}),
		),
	)
	myWindow.SetMainMenu(mainMenu)

	// 创建命名空间搜索框
	gNamespaceSearch = widget.NewEntry()
	gNamespaceSearch.SetPlaceHolder("搜索命名空间...")

	// 创建过滤后的命名空间列表
	gFilteredNamespaces = gNamespaces

	// 创建左侧的命名空间列表
	gNamespaceList = widget.NewList(
		func() int { return len(gFilteredNamespaces) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Item")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(gFilteredNamespaces[id].Name)
		},
	)
	gNamespaceList.OnSelected = func(id widget.ListItemID) {
		selectNamespace(gFilteredNamespaces[id])
		updateInfoArea()
	}

	// 添加命名空间搜索功能
	gNamespaceSearch.OnChanged = func(s string) {
		gFilteredNamespaces = filterNamespaces(gNamespaces, s)
		gNamespaceList.UnselectAll()
		gNamespaceList.ScrollToTop()
		gNamespaceList.Refresh()
		if len(gFilteredNamespaces) >= 1 {
			gNamespaceList.Select(0)
		}
		updateInfoArea()
	}

	namespaceScroll := container.NewScroll(gNamespaceList)
	namespaceScroll.SetMinSize(fyne.NewSize(200, 350))

	// 创建服务workload索框
	gWorkloadSearch = widget.NewEntry()
	gWorkloadSearch.SetPlaceHolder("搜索服务...")

	// 创建过滤后的服务列表
	gFilteredWorkloads = gWorkloads

	// 建中间的服务列表
	gWorkloadList = component.NewList(
		func() int { return len(gFilteredWorkloads) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", func(bool) {}),
				widget.NewLabel("Template Service"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			workload := gFilteredWorkloads[id]
			check := item.(*fyne.Container).Objects[0].(*widget.Check)
			label := item.(*fyne.Container).Objects[1].(*widget.Label)
			label.SetText(workload.Name)
			check.OnChanged = func(checked bool) {
				if checked {
					gWorkloadList.MultiSelectedOne(id)
				} else {
					gWorkloadList.UnMultiSelectedOne(id)
				}
			}

			isSelected := false
			for _, w := range gSelectedWorkloads {
				if w.Name == workload.Name {
					isSelected = true
					break
				}
			}
			check.SetChecked(isSelected)
		},
	)
	gWorkloadList.OnMultiSelected(func(ids []int) {
		// 清空之前选择的workloads
		gSelectedWorkloads = []rancher.Workload{}

		// 根据选中的ID添加workload到选中列表
		for _, id := range ids {
			if id < len(gFilteredWorkloads) {
				gSelectedWorkloads = append(gSelectedWorkloads, gFilteredWorkloads[id])
			}
		}
		// 更新信息区域显示
		updateInfoArea()
	})

	// 添加服务搜索功能
	gWorkloadSearch.OnChanged = func(s string) {
		gFilteredWorkloads = filterWorkloads(gWorkloads, s)
		gWorkloadList.UnselectMulti()
		gWorkloadList.RefreshList()
		if len(gFilteredWorkloads) == 1 {
			gWorkloadList.Select(0)
		}
		updateInfoArea()
	}

	workloadScroll := container.NewScroll(gWorkloadList)
	workloadScroll.SetMinSize(fyne.NewSize(200, 350))

	// 创建右侧信息区域
	gInfoArea = widget.NewMultiLineEntry()
	gInfoArea.SetText("")
	gInfoArea.SetMinRowsVisible(15)
	// 将 InfoArea 放固定大小的容器中
	infoContainer := container.NewScroll(gInfoArea)
	infoContainer.SetMinSize(fyne.NewSize(600, 380))

	// 添加更pod按钮
	buttonUpdatePod := widget.NewButton("更新Pod", func() {
		var info strings.Builder
		if gEnvironment != nil {
			// 只更新当前选中的环境
			info.WriteString(fmt.Sprintf("更新Pod: %s ", gEnvironment.Name))
			gInfoArea.SetText(info.String())
			rancher.UpdatePod(gDb, gEnvironment.ID, gEnvironment)
			info.WriteString("完成!\n")
		} else {
			// 如果没有选中环境，则更新所有环境
			for envName, _ := range gConfig["environment"].(map[interface{}]interface{}) {
				environment, _ := rancher.GetEnvironmentFromConfig(gConfig, envName.(string))
				info.WriteString(fmt.Sprintf("更新Pod: %s ", environment.Name))
				gInfoArea.SetText(info.String())
				rancher.UpdateEnvironment(gDb, environment.ID, environment, false)
				info.WriteString("完成!\n")
			}
		}
		gInfoArea.SetText(info.String())
		updateInfoArea()
	})

	buttonOpen := widget.NewButton("打开", func() {
		var info strings.Builder
		if len(gSelectedWorkloads) > 0 {
			// 处理多选的情况
			for _, workload := range gSelectedWorkloads {
				info.WriteString(fmt.Sprintf("打开: %s", workload.Name))
				success := rancher.Scale(*gEnvironment, workload.Namespace, workload.Name, 1)
				if success {
					info.WriteString("成功!\n")
				} else {
					info.WriteString("失败!\n")
				}
			}
		} else if len(gFilteredWorkloads) > 0 {
			// 处理未选择的情况，使用过滤列表中的所有数据
			for _, workload := range gFilteredWorkloads {
				info.WriteString(fmt.Sprintf("打开: %s    ", workload.Name))
				success := rancher.Scale(*gEnvironment, workload.Namespace, workload.Name, 1)
				if success {
					info.WriteString("成功!\n")
				} else {
					info.WriteString("失败!\n")
				}
			}
		}
		gInfoArea.SetText(info.String())
	})
	buttonClose := widget.NewButton("关闭", func() {
		var info strings.Builder
		if len(gSelectedWorkloads) > 0 {
			// 处理多选的情况
			for _, workload := range gSelectedWorkloads {
				info.WriteString(fmt.Sprintf("关闭: %s    ", workload.Name))
				success := rancher.Scale(*gEnvironment, workload.Namespace, workload.Name, 0)
				if success {
					info.WriteString("成功!\n")
				} else {
					info.WriteString("失败!\n")
				}
			}
		} else if len(gFilteredWorkloads) > 0 {
			// 处理未选择的情况，使用过滤列表中的所有数据
			for _, workload := range gFilteredWorkloads {
				info.WriteString(fmt.Sprintf("关闭: %s    ", workload.Name))
				success := rancher.Scale(*gEnvironment, workload.Namespace, workload.Name, 0)
				if success {
					info.WriteString("成功!\n")
				} else {
					info.WriteString("失败!\n")
				}
			}
		}
		gInfoArea.SetText(info.String())
	})
	buttonRedeploy := widget.NewButton("重新部署", func() {
		var info strings.Builder
		if len(gSelectedWorkloads) > 0 {
			// 处理多选的情况
			for _, workload := range gSelectedWorkloads {
				info.WriteString(fmt.Sprintf("重新部署: %s    ", workload.Name))
				success := rancher.Redeploy(*gEnvironment, workload.Namespace, workload.Name)
				if success {
					info.WriteString("成功!\n")
				} else {
					info.WriteString("失败!\n")
				}
			}
		} else if len(gFilteredWorkloads) > 0 {
			// 处理未选择的情况，使用过滤列表中的所有数据
			for _, workload := range gFilteredWorkloads {
				info.WriteString(fmt.Sprintf("重新部署: %s    ", workload.Name))
				success := rancher.Redeploy(*gEnvironment, workload.Namespace, workload.Name)
				if success {
					info.WriteString("成功!\n")
				} else {
					info.WriteString("失败!\n")
				}
			}
		}
		gInfoArea.SetText(info.String())
	})

	// 更新布局（移除了buttonUpdateData）
	content := container.NewHBox(
		container.NewVBox(
			widget.NewLabel("命名空间"),
			gNamespaceSearch,
			namespaceScroll,
		),
		container.NewVBox(
			widget.NewLabel("服务"),
			gWorkloadSearch,
			workloadScroll,
		),
		container.NewVBox(
			container.NewHBox(buttonUpdatePod, buttonOpen, buttonClose, buttonRedeploy),
			infoContainer,
		),
	)
	myWindow.SetContent(content)
	return myWindow
}
func loadConfig(showSuccessTip bool) {
	var err error
	gConfig, err = rancher.LoadConfigFromDb(gDb)
	// 解析跳板机配置
	if jumpHost, exists := gConfig["jump_host"].(map[interface{}]interface{}); exists {
		gJumpHostConfig = &rancher.JumpHostConfig{
			Ip:       jumpHost["ip"].(string),
			Port:     strconv.Itoa(jumpHost["port"].(int)),
			Username: jumpHost["username"].(string),
			Password: jumpHost["password"].(string),
			RootPath: jumpHost["root_path"].(string),
		}
	}
	if err != nil {
		gInfoArea.SetText(fmt.Sprintf("从数据库读取配置时出错: %v", err))
	} else {
		if showSuccessTip {
			gInfoArea.SetText("配置已成功加载")
		}
	}
}
func initData() {
	namespaces, _ := gDb.GetAllNamespacesDetail()
	gNamespaces = append(namespaces)
	gFilteredNamespaces = append(gNamespaces)
	gSelectedNamespace = rancher.Namespace{}
	gNamespaceList.UnselectAll()
	gNamespaceList.ScrollToTop()
	gNamespaceList.Refresh()
	gNamespaceSearch.SetText("")

	gWorkloads = []rancher.Workload{}
	gFilteredWorkloads = []rancher.Workload{}
	gWorkloadList.RefreshList()
	gWorkloadSearch.SetText("")
}

func selectNamespace(namespace rancher.Namespace) {
	gSelectedNamespace = namespace
	gEnvironment, _ = rancher.GetEnvironmentFromConfig(gConfig, gSelectedNamespace.Environment)

	workloads, _ := gDb.GetWorkloadsByNamespace(namespace.Name)
	gWorkloads = workloads
	gWorkloadSearch.SetText("")
	gFilteredWorkloads = gWorkloads
	gSelectedWorkloads = []rancher.Workload{}
	gWorkloadList.UnselectMulti()
	gWorkloadList.RefreshList()
}

func updateInfoArea() {
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

func updateInfoAreaForSelectNamespace() {
	podList, _ := gDb.GetPodsByEnvNamespace(gSelectedNamespace.Environment, gSelectedNamespace.Name)

	var info strings.Builder
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

func updateInfoAreaForSingleWorkload() {
	workload := gSelectedWorkloads[0]
	podList, _ := gDb.GetPodsByEnvNamespaceWorkload(workload.Environment, workload.Namespace, workload.Name)

	// 构建信息字符串
	var info strings.Builder

	info.WriteString(fmt.Sprintf("环境: %s\n", workload.Environment))
	info.WriteString(fmt.Sprintf("命名空间: %s\n", workload.Namespace))
	info.WriteString(fmt.Sprintf("名称: %s\n", workload.Name))
	info.WriteString(fmt.Sprintf("镜像: %s\n", workload.Image))
	info.WriteString(fmt.Sprintf("pod数量: %d\n", len(podList)))
	if len(podList) > 0 {
		var states []string
		for _, pod := range podList {
			states = append(states, pod.State)
		}
		info.WriteString(fmt.Sprintf("Pod状态: %s\n", strings.Join(states, ",")))
	}
	// 如果工作负载名称包含mysql，尝试获取MySQL root密码
	if strings.Contains(strings.ToLower(workload.Name), "mysql") && workload.ContainerEnvironment != "" {
		var envVars map[string]string
		if err := json.Unmarshal([]byte(workload.ContainerEnvironment), &envVars); err == nil {
			if rootPassword, exists := envVars["MYSQL_ROOT_PASSWORD"]; exists {
				info.WriteString(fmt.Sprintf("MySQL Root密码: %s\n", rootPassword))
			}
		}
	}
	// 只有当 NodePort 不为空时才显示，并按逗号分隔成多行
	if workload.NodePort != "" {
		info.WriteString("端口访问:\n")
		ports := strings.Split(workload.NodePort, ",")
		ip := gEnvironment.Ip
		for _, port := range ports {
			info.WriteString(fmt.Sprintf("  %s:%s\n", ip, strings.TrimSpace(port)))
		}
	}
	if workload.ImagePullPolicy != "" {
		info.WriteString(fmt.Sprintf("镜像拉取策略: %s\n", workload.ImagePullPolicy))
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
				if strings.Count(config.Image, "$") == 1 {
					script = script + " " + tag
				} else if strings.Count(config.Image, "$") == 2 {
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

// 添加新的函数来更新信息区域显示多选内容
func updateInfoAreaForSelectMultiWorkload() {
	var info strings.Builder
	info.WriteString(fmt.Sprintf("已选择 %d 个服务:\n", len(gSelectedWorkloads)))

	for _, workload := range gSelectedWorkloads {
		info.WriteString(fmt.Sprintf("\n服务名称: %s\n", workload.Name))
		info.WriteString(fmt.Sprintf("镜像: %s\n", workload.Image))
	}

	gInfoArea.SetText(info.String())
}

// 添加过过滤函数
func filterNamespaces(items []rancher.Namespace, filter string) []rancher.Namespace {
	if filter == "" {
		return items
	}
	var filtered []rancher.Namespace
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(filter)) || strings.Contains(strings.ToLower(item.Description), strings.ToLower(filter)) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterWorkloads(items []rancher.Workload, filter string) []rancher.Workload {
	if filter == "" {
		return items
	}
	var filtered []rancher.Workload
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.Name), strings.ToLower(filter)) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func cloneOrExportWorkload(isClone bool, destNamespace rancher.Namespace) {
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

func cloneOrExportConfigMap(isClone bool, destNamespace rancher.Namespace) {
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

// 在 main.go 中添加以下结构体和方法
type jumpHostProgressListener struct {
	infoArea *widget.Entry
}

func (l *jumpHostProgressListener) OnProgress(currentFolder string, current, total int) {
	// 直接更新 UI
	l.infoArea.SetText(fmt.Sprintf("正在扫描... %d/%d\n当前目录:%s", current, total, currentFolder))
}

func (l *jumpHostProgressListener) OnComplete() {
	// 直接更新 UI
	l.infoArea.SetText("更新跳板机完成")
}

func (l *jumpHostProgressListener) OnBatchResult(configs []rancher.SSHUploadConfig) {
	// 将 SSHUploadConfig 转换为 UploadConfig
	var uploadConfigs []rancher.UploadConfig
	for _, config := range configs {
		uploadConfig := rancher.UploadConfig{
			Dir:    config.Dir,
			Script: config.Script,
			Jar:    config.Jar,
			Image:  config.Image,
		}
		uploadConfigs = append(uploadConfigs, uploadConfig)
	}

	// 插入数据库
	gDb.InsertUploadConfigs(uploadConfigs)
}
