package ui

import (
	"fmt"
	"log"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"RancherMan/operations"
	"RancherMan/rancher"
	"RancherMan/ui/component"
)

// InitView 初始化应用程序界面
func InitView() fyne.Window {
	// 初始化界面
	gApp = app.New()
	myWindow := gApp.NewWindow("Rancher助手")

	// 创建主菜单
	mainMenu := fyne.NewMainMenu(
		fyne.NewMenu("配置",
			fyne.NewMenuItem("保存配置", func() {
				var content = operations.GetInfoArea().Text
				rancher.SaveConfigToDb(operations.GetDb(), content)
				operations.LoadConfig(true)
				operations.InitData()
			}),
			fyne.NewMenuItem("显示配置", func() {
				configContent, _ := operations.GetDb().GetConfigContent(1)
				operations.GetInfoArea().SetText(configContent)
			}),
		),
		fyne.NewMenu("数据",
			fyne.NewMenuItem("更新数据", func() {
				var info strings.Builder
				env := operations.GetEnvironment()
				config := operations.GetConfig()
				db := operations.GetDb()
				infoArea := operations.GetInfoArea()

				if env != nil {
					// 只更新当前选中的环境
					info.WriteString(fmt.Sprintf("更新数据: %s ", env.Name))
					infoArea.SetText(info.String())
					rancher.UpdateEnvironment(db, env.ID, env, true)
					info.WriteString("完成!\n")
					infoArea.SetText(info.String())
				} else {
					// 如果没有选中环境，则更新所有环境
					for envName, _ := range config["environment"].(map[interface{}]interface{}) {
						environment, _ := rancher.GetEnvironmentFromConfig(config, envName.(string))
						info.WriteString(fmt.Sprintf("更新数据: %s ", environment.Name))
						infoArea.SetText(info.String())
						rancher.UpdateEnvironment(db, environment.ID, environment, true)
						info.WriteString("完成!\n")
						infoArea.SetText(info.String())
					}
				}
				operations.InitData()
			}),
			fyne.NewMenuItem("更新端口映射", func() {
				var info strings.Builder
				env := operations.GetEnvironment()
				config := operations.GetConfig()
				db := operations.GetDb()
				infoArea := operations.GetInfoArea()

				if env != nil {
					// 只更新当前选中的环境
					info.WriteString(fmt.Sprintf("更新端口映射: %s ", env.Name))
					infoArea.SetText(info.String())
					rancher.UpdateService(db, env.ID, env)
					info.WriteString("完成!\n")
					infoArea.SetText(info.String())
				} else {
					// 如果没有选中环境，则更新所有环境
					for envName, _ := range config["environment"].(map[interface{}]interface{}) {
						environment, _ := rancher.GetEnvironmentFromConfig(config, envName.(string))
						info.WriteString(fmt.Sprintf("更新端口映射: %s ", environment.Name))
						infoArea.SetText(info.String())
						rancher.UpdateService(db, environment.ID, environment)
						info.WriteString("完成!\n")
						infoArea.SetText(info.String())
					}
				}
			}),
			fyne.NewMenuItem("更新跳板机", func() {
				jumpHostConfig := operations.GetJumpHostConfig()
				db := operations.GetDb()
				infoArea := operations.GetInfoArea()

				if jumpHostConfig == nil {
					infoArea.SetText("错误：未配置跳板机信息")
					return
				}

				// 创建进度监听器
				listener := operations.NewJumpHostProgressListener(infoArea)

				// 清空信息区域并显示初始信息
				infoArea.SetText("开始扫描跳板机配置...\n")

				// 在新的 goroutine 中执行耗时操作
				go func() {
					db.DeleteAllUploadConfigs()
					rancher.ListUploadConfig(jumpHostConfig, 50, listener)
				}()
			}),
			fyne.NewMenuItem("清空数据", func() {
				db := operations.GetDb()
				infoArea := operations.GetInfoArea()
				err := db.ClearAllData()
				if err != nil {
					infoArea.SetText(fmt.Sprintf("清空数据失败: %v", err))
				} else {
					infoArea.SetText("数据已清空")
				}

				operations.InitData()
			}),
		),
		fyne.NewMenu("克隆和导出",
			fyne.NewMenuItem("导出configMap", func() {
				ShowSelectNamespaceDialog(myWindow, operations.GetDb(), false, func(destNamespace rancher.Namespace, tag string) {
					operations.CloneOrExportConfigMap(false, destNamespace)
				})
			}),
			fyne.NewMenuItem("克隆configMap", func() {
				ShowSelectNamespaceDialog(myWindow, operations.GetDb(), true, func(destNamespace rancher.Namespace, tag string) {
					operations.CloneOrExportConfigMap(true, destNamespace)
				})
			}),
			fyne.NewMenuItem("导出workload", func() {
				ShowSelectNamespaceDialog(myWindow, operations.GetDb(), true, func(destNamespace rancher.Namespace, tag string) {
					operations.CloneOrExportWorkload(false, destNamespace, tag)
				})
			}),
			fyne.NewMenuItem("克隆workload", func() {
				ShowSelectNamespaceDialog(myWindow, operations.GetDb(), true, func(destNamespace rancher.Namespace, tag string) {
					operations.CloneOrExportWorkload(true, destNamespace, tag)
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

	// 创建左侧的命名空间列表
	gNamespaceList = widget.NewList(
		func() int { return len(operations.GetFilteredNamespaces()) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Item")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(operations.GetFilteredNamespaces()[id].Name)
		},
	)
	gNamespaceList.OnSelected = func(id widget.ListItemID) {
		selectNamespace(operations.GetFilteredNamespaces()[id])
		operations.UpdateInfoArea()
	}

	// 添加命名空间搜索功能
	gNamespaceSearch.OnChanged = func(s string) {
		operations.SetFilteredNamespaces(operations.FilterNamespaces(operations.GetNamespaces(), s))
		gNamespaceList.UnselectAll()
		gNamespaceList.ScrollToTop()
		gNamespaceList.Refresh()
		if len(operations.GetFilteredNamespaces()) >= 1 {
			gNamespaceList.Select(0)
		}
		operations.UpdateInfoArea()
	}

	namespaceScroll := container.NewScroll(gNamespaceList)
	namespaceScroll.SetMinSize(fyne.NewSize(250, 0))

	// 创建服务workload索框
	gWorkloadSearch = widget.NewEntry()
	gWorkloadSearch.SetPlaceHolder("搜索服务...")

	// 建中间的服务列表
	gWorkloadList = component.NewList(
		func() int { return len(operations.GetFilteredWorkloads()) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", func(bool) {}),
				widget.NewLabel("Template Service"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			workload := operations.GetFilteredWorkloads()[id]
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
			for _, w := range operations.GetSelectedWorkloads() {
				if w.Name == workload.Name {
					isSelected = true
					break
				}
			}
			check.SetChecked(isSelected)
		},
	)
	gWorkloadList.OnMultiSelected(func(ids []int) {
		log.Printf("[OnMultiSelected] Selected IDs: %v, len(gFilteredWorkloads)=%d", ids, len(operations.GetFilteredWorkloads()))

		// 清空之前选择的workloads
		operations.SetSelectedWorkloads([]rancher.Workload{})

		// 根据选中的ID添加workload到选中列表
		var selectedWorkloads []rancher.Workload
		for _, id := range ids {
			if id < len(operations.GetFilteredWorkloads()) {
				workloads := operations.GetFilteredWorkloads()
				selectedWorkloads = append(selectedWorkloads, workloads[id])
				log.Printf("[OnMultiSelected] Added workload: %s (id=%d)", workloads[id].Name, id)
			}
		}
		operations.SetSelectedWorkloads(selectedWorkloads)
		log.Printf("[OnMultiSelected] Total selected workloads: %d", len(selectedWorkloads))
		// 更新信息区域显示
		operations.UpdateInfoArea()
	})

	// 添加服务搜索功能
	gWorkloadSearch.OnChanged = func(s string) {
		operations.SetFilteredWorkloads(operations.FilterWorkloads(operations.GetWorkloads(), s))
		gWorkloadList.UnselectMulti()
		gWorkloadList.RefreshList()
		if len(operations.GetFilteredWorkloads()) == 1 {
			gWorkloadList.Select(0)
		}
		operations.UpdateInfoArea()
	}

	workloadScroll := container.NewScroll(gWorkloadList)
	workloadScroll.SetMinSize(fyne.NewSize(200, 0))

	// 创建右侧信息区域
	gInfoArea = widget.NewMultiLineEntry()
	gInfoArea.SetText("")
	infoContainer := container.NewScroll(gInfoArea)
	infoContainer.SetMinSize(fyne.NewSize(400, 0))

	// 将InfoArea设置到operations包
	operations.SetInfoArea(gInfoArea)

	// 添加更pod按钮
	buttonUpdatePod := widget.NewButton("更新Pod", func() {
		log.Printf("[buttonUpdatePod] Clicked, gEnvironment=%v", operations.GetEnvironment() != nil)
		env := operations.GetEnvironment()
		db := operations.GetDb()
		taskQueue := operations.GetTaskQueue()

		if env != nil {
			// 只更新当前选中的环境
			newTask := &rancher.Task{
				Type:        rancher.TaskTypeUpdatePod,
				Description: fmt.Sprintf("更新Pod: %s", env.Name),
				Environment: *env,
				DB:          db,
			}
			taskID := taskQueue.AddTask(newTask)
			log.Printf("[buttonUpdatePod] Task added: ID=%d, Description=%s", taskID, newTask.Description)
			taskQueue.Submit(newTask)
		} else {
			log.Printf("[buttonUpdatePod] gEnvironment is nil, skipping")
		}
	})

	buttonOpen := widget.NewButton("打开", func() {
		log.Printf("[buttonOpen] Clicked, gEnvironment=%v", operations.GetEnvironment() != nil)
		log.Printf("[buttonOpen] len(gSelectedWorkloads)=%d, len(gFilteredWorkloads)=%d", len(operations.GetSelectedWorkloads()), len(operations.GetFilteredWorkloads()))

		env := operations.GetEnvironment()
		infoArea := operations.GetInfoArea()
		taskQueue := operations.GetTaskQueue()

		if env == nil {
			log.Printf("[buttonOpen] ERROR: gEnvironment is nil!")
			infoArea.SetText("错误: 请先选择命名空间")
			return
		}

		workloadsToProcess := operations.GetSelectedWorkloads()
		if len(workloadsToProcess) == 0 {
			log.Printf("[buttonOpen] No selected workloads, using filtered workloads")
			workloadsToProcess = operations.GetFilteredWorkloads()
		}

		log.Printf("[buttonOpen] Processing %d workloads", len(workloadsToProcess))

		for _, workload := range workloadsToProcess {
			log.Printf("[buttonOpen] Creating task for workload: %s", workload.Name)
			newTask := &rancher.Task{
				Type:        rancher.TaskTypeScaleOpen,
				Description: fmt.Sprintf("打开 %s", workload.Name),
				Environment: *env,
				Namespace:   workload.Namespace,
				Workload:    workload.Name,
				Replicas:    1,
			}
			log.Printf("[buttonOpen] About to call AddTask...")
			taskID := taskQueue.AddTask(newTask)
			log.Printf("[buttonOpen] Task added: ID=%d, Description=%s", taskID, newTask.Description)
			taskQueue.Submit(newTask)
		}
	})

	buttonClose := widget.NewButton("关闭", func() {
		log.Printf("[buttonClose] Clicked, gEnvironment=%v", operations.GetEnvironment() != nil)

		env := operations.GetEnvironment()
		infoArea := operations.GetInfoArea()
		taskQueue := operations.GetTaskQueue()

		if env == nil {
			log.Printf("[buttonClose] ERROR: gEnvironment is nil!")
			infoArea.SetText("错误: 请先选择命名空间")
			return
		}

		workloadsToProcess := operations.GetSelectedWorkloads()
		if len(workloadsToProcess) == 0 {
			workloadsToProcess = operations.GetFilteredWorkloads()
		}

		log.Printf("[buttonClose] Processing %d workloads", len(workloadsToProcess))

		for _, workload := range workloadsToProcess {
			newTask := &rancher.Task{
				Type:        rancher.TaskTypeScaleClose,
				Description: fmt.Sprintf("关闭 %s", workload.Name),
				Environment: *env,
				Namespace:   workload.Namespace,
				Workload:    workload.Name,
				Replicas:    0,
			}
			taskID := taskQueue.AddTask(newTask)
			log.Printf("[buttonClose] Task added: ID=%d, Description=%s", taskID, newTask.Description)
			taskQueue.Submit(newTask)
		}
	})

	buttonRedeploy := widget.NewButton("重新部署", func() {
		log.Printf("[buttonRedeploy] Clicked, gEnvironment=%v", operations.GetEnvironment() != nil)

		env := operations.GetEnvironment()
		infoArea := operations.GetInfoArea()
		taskQueue := operations.GetTaskQueue()

		if env == nil {
			log.Printf("[buttonRedeploy] ERROR: gEnvironment is nil!")
			infoArea.SetText("错误: 请先选择命名空间")
			return
		}

		workloadsToProcess := operations.GetSelectedWorkloads()
		if len(workloadsToProcess) == 0 {
			workloadsToProcess = operations.GetFilteredWorkloads()
		}

		log.Printf("[buttonRedeploy] Processing %d workloads", len(workloadsToProcess))

		for _, workload := range workloadsToProcess {
			newTask := &rancher.Task{
				Type:        rancher.TaskTypeRedeploy,
				Description: fmt.Sprintf("重新部署 %s", workload.Name),
				Environment: *env,
				Namespace:   workload.Namespace,
				Workload:    workload.Name,
			}
			taskID := taskQueue.AddTask(newTask)
			log.Printf("[buttonRedeploy] Task added: ID=%d, Description=%s", taskID, newTask.Description)
			taskQueue.Submit(newTask)
		}
	})

	// 创建任务状态栏
	gTaskStatusBar = component.NewTaskStatusBar(func() {
		taskQueue := operations.GetTaskQueue()
		if taskQueue != nil {
			taskQueue.CancelAll()
		}
	})

	// 将TaskStatusBar设置到operations包
	operations.SetTaskStatusBar(gTaskStatusBar)

	// 收集所有操作按钮用于禁用/启用
	gOperationButtons = []*widget.Button{buttonUpdatePod, buttonOpen, buttonClose, buttonRedeploy}
	operations.SetOperationButtons(gOperationButtons)

	// 使用 Border 布局让高度自适应
	// 每一列内部用 Border：顶部是标签和搜索框，中间是滚动内容（自动扩展），底部为空
	leftCol := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("命名空间"),
			gNamespaceSearch,
		),
		nil,
		nil,
		nil,
		namespaceScroll,
	)

	middleCol := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("服务"),
			gWorkloadSearch,
		),
		nil,
		nil,
		nil,
		workloadScroll,
	)

	// 右侧区域：顶部是按钮，中间是信息区域
	rightCol := container.NewBorder(
		container.NewHBox(buttonUpdatePod, buttonOpen, buttonClose, buttonRedeploy), // top
		nil,           // bottom
		nil,           // left
		nil,           // right
		infoContainer, // center
	)

	// 将左右两列合并放在一起
	leftPanel := container.NewHBox(leftCol, middleCol)

	// 上半部分：左边是两列，右边是 InfoArea
	topContent := container.NewBorder(nil, nil, leftPanel, nil, rightCol)

	// 整体布局：上部是内容区，底部是任务栏（横跨整个窗口）
	content := container.NewBorder(nil, gTaskStatusBar.GetContainer(), nil, nil, topContent)
	myWindow.SetContent(content)
	// 设置窗口初始大小
	myWindow.Resize(fyne.NewSize(1050, 600))

	// 将其他UI组件设置到operations包
	operations.SetNamespaceList(gNamespaceList)
	operations.SetNamespaceSearch(gNamespaceSearch)
	operations.SetWorkloadList(gWorkloadList)
	operations.SetWorkloadSearch(gWorkloadSearch)

	return myWindow
}

// selectNamespace 选择命名空间
func selectNamespace(namespace rancher.Namespace) {
	operations.SetSelectedNamespace(namespace)

	db := operations.GetDb()
	config := operations.GetConfig()

	environment, _ := rancher.GetEnvironmentFromConfig(config, namespace.Environment)
	operations.SetEnvironment(environment)

	log.Printf("[selectNamespace] Selected namespace: %s, gEnvironment=%v", namespace.Name, environment != nil)
	if environment != nil {
		log.Printf("[selectNamespace] Environment: Name=%s, ID=%s", environment.Name, environment.ID)
	}

	workloads, _ := db.GetWorkloadsByNamespace(namespace.Name)
	operations.SetWorkloads(workloads)
	log.Printf("[selectNamespace] Loaded %d workloads", len(workloads))

	operations.SetWorkloadSearchText("")
	operations.SetFilteredWorkloads(workloads)
	operations.SetSelectedWorkloads([]rancher.Workload{})

	gWorkloadList.UnselectMulti()
	gWorkloadList.RefreshList()
}
