package main

import (
	"RancherMan/rancher"
	"fmt"
	"log"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/flopp/go-findfont"
)

// 数据
var gNamespaces []rancher.Namespace
var gFilteredNamespaces []rancher.Namespace
var gSelectedNamespace rancher.Namespace
var gWorkloads []rancher.Workload
var gFilteredWorkloads []rancher.Workload
var gSelectedWorkloads []rancher.Workload
var gSelectedWorkload rancher.Workload

// UI组件
var gNamespaceList *widget.List
var gWorkloadList *widget.List
var gInfoArea *widget.Entry
var gWorkloadSearch *widget.Entry

// 数据库和配置
var gDb *rancher.DatabaseManager
var gConfig map[string]interface{}
var gEnvironment *rancher.Environment

// 绑定数据
var gWorkloadData binding.StringList

func main() {
	//// 创建数据库管理器实例
	database, err := rancher.NewDatabaseManager("")
	if err != nil {
		log.Fatal(err)
	}
	gDb = database
	defer gDb.Close()
	initFont()
	window := initView()
	loadConfig(false)
	initData()
	window.ShowAndRun()
}
func initFont() {
	// 设置中文字体
	fontPaths := findfont.List()
	for _, path := range fontPaths {
		if strings.Contains(path, "msyh.ttf") || strings.Contains(path, "simhei.ttf") || strings.Contains(path, "simsun.ttc") || strings.Contains(path, "simkai.ttf") {
			os.Setenv("FYNE_FONT", path)
			break
		}
	}
}
func initView() fyne.Window {
	//// 初始化界面
	myApp := app.New()
	myWindow := myApp.NewWindow("Rancher助手")

	// 创建主菜单
	mainMenu := fyne.NewMainMenu(
		fyne.NewMenu("文件",
			fyne.NewMenuItem("加载配置", func() {
				var content = gInfoArea.Text
				rancher.SaveConfigToDb(gDb, content)
				loadConfig(true)
			}),
			fyne.NewMenuItem("显示配置", func() {
				configContent, _ := gDb.GetConfigContent(1)
				gInfoArea.SetText(configContent)
			}),
			fyne.NewMenuItem("更新数据", func() {
				var info strings.Builder
				if gEnvironment != nil {
					// 只更新当前选中的环境
					info.WriteString(fmt.Sprintf("更新数据: %s ", gEnvironment.Name))
					gInfoArea.SetText(info.String())
					rancher.UpdateEnvironment(gDb, gEnvironment.ID, gEnvironment, true)
					info.WriteString("完成!\n")
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
					// 刷新界面
				}
				initData()
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
	namespaceSearch := widget.NewEntry()
	namespaceSearch.SetPlaceHolder("搜索命名空间...")

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
		selectNameSpace(gFilteredNamespaces[id])
	}

	// 添加命名空间搜索功能
	namespaceSearch.OnChanged = func(s string) {
		gFilteredNamespaces = filterNamespaces(gNamespaces, s)
		gNamespaceList.Refresh()
	}

	namespaceScroll := container.NewScroll(gNamespaceList)
	namespaceScroll.SetMinSize(fyne.NewSize(200, 300))

	// 创建服务workload索框
	gWorkloadSearch = widget.NewEntry()
	gWorkloadSearch.SetPlaceHolder("搜索服务...")

	// 创建过滤后的服务列表
	gFilteredWorkloads = gWorkloads

	// 建中间的服务列表
	gWorkloadData = binding.NewStringList()
	gWorkloadList = widget.NewListWithData(
		gWorkloadData,
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewCheck("", func(bool) {}),
				widget.NewLabel("Template Service"),
			)
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			text, _ := item.(binding.String).Get()
			index := -1
			for i, w := range gFilteredWorkloads {
				if w.Name == text {
					index = i
					break
				}
			}
			if index == -1 {
				return
			}

			check := obj.(*fyne.Container).Objects[0].(*widget.Check)
			label := obj.(*fyne.Container).Objects[1].(*widget.Label)
			label.SetText(gFilteredWorkloads[index].Name)

			check.OnChanged = func(checked bool) {
				if checked {
					gSelectedWorkloads = append(gSelectedWorkloads, gFilteredWorkloads[index])
					unselectWorkloadSingle()
				} else {
					for i, w := range gSelectedWorkloads {
						if w.Name == gFilteredWorkloads[index].Name {
							gSelectedWorkloads = append(gSelectedWorkloads[:i], gSelectedWorkloads[i+1:]...)
							break
						}
					}
				}
				updateInfoAreaForMultiSelect()
			}

			isSelected := false
			for _, w := range gSelectedWorkloads {
				if w.Name == gFilteredWorkloads[index].Name {
					isSelected = true
					break
				}
			}
			check.SetChecked(isSelected)
		},
	)

	// 添加服务搜索功能
	gWorkloadSearch.OnChanged = func(s string) {
		gFilteredWorkloads = filterWorkloads(gWorkloads, s)

		// 更新绑定数据
		workloadNames := make([]string, len(gFilteredWorkloads))
		for i, w := range gFilteredWorkloads {
			workloadNames[i] = w.Name
		}
		gWorkloadData.Set(workloadNames)
	}

	gWorkloadList.OnSelected = func(id widget.ListItemID) {
		if len(gSelectedWorkloads) > 0 {
			unselectWorkloadSingle()
			return
		}
		selectWorkloadSingle(gFilteredWorkloads[id])
	}

	workloadScroll := container.NewScroll(gWorkloadList)
	workloadScroll.SetMinSize(fyne.NewSize(200, 350))

	// 创建右侧信息区域
	gInfoArea = widget.NewMultiLineEntry()
	gInfoArea.SetText("")
	gInfoArea.SetMinRowsVisible(15)
	// 将 InfoArea 放入固定大小的容器中
	infoContainer := container.NewScroll(gInfoArea)
	infoContainer.SetMinSize(fyne.NewSize(500, 380))

	// 添加更新pod按钮
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
		if (gSelectedWorkload != rancher.Workload{}) {
			selectWorkloadSingle(gSelectedWorkload)
		} else if (gSelectedWorkload == rancher.Workload{}) && len(gSelectedWorkloads) == 0 && (gSelectedNamespace != rancher.Namespace{}) {
			selectNameSpace(gSelectedNamespace)
		}
	})

	buttonOpen := widget.NewButton("打开", func() {
		var info strings.Builder
		if (gSelectedWorkload != rancher.Workload{}) {
			// 处理单选的情况
			info.WriteString(fmt.Sprintf("打开: %s\t", gSelectedWorkload.Name))
			success := rancher.Scale(*gEnvironment, gSelectedWorkload.Namespace, gSelectedWorkload.Name, 1)
			if success {
				info.WriteString("成功!\n")
			} else {
				info.WriteString("失败!\n")
			}
		} else if len(gSelectedWorkloads) > 0 {
			// 处理多选的情况
			for _, workload := range gSelectedWorkloads {
				info.WriteString(fmt.Sprintf("打开: %s\t", workload.Name))
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
				info.WriteString(fmt.Sprintf("打开: %s\t", workload.Name))
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
		if (gSelectedWorkload != rancher.Workload{}) {
			// 处理单选的情况
			info.WriteString(fmt.Sprintf("关闭: %s\t", gSelectedWorkload.Name))
			success := rancher.Scale(*gEnvironment, gSelectedWorkload.Namespace, gSelectedWorkload.Name, 0)
			if success {
				info.WriteString("成功!\n")
			} else {
				info.WriteString("失败!\n")
			}
		} else if len(gSelectedWorkloads) > 0 {
			// 处理多选的情况
			for _, workload := range gSelectedWorkloads {
				info.WriteString(fmt.Sprintf("关闭: %s\t", workload.Name))
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
				info.WriteString(fmt.Sprintf("关闭: %s\t", workload.Name))
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
		if (gSelectedWorkload != rancher.Workload{}) {
			// 处理单选的情况
			info.WriteString(fmt.Sprintf("重新部署: %s\t", gSelectedWorkload.Name))
			success := rancher.Redeploy(*gEnvironment, gSelectedWorkload.Namespace, gSelectedWorkload.Name)
			if success {
				info.WriteString("成功!\n")
			} else {
				info.WriteString("失败!\n")
			}
		} else if len(gSelectedWorkloads) > 0 {
			// 处理多选的情况
			for _, workload := range gSelectedWorkloads {
				info.WriteString(fmt.Sprintf("重新部署: %s\t", workload.Name))
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
				info.WriteString(fmt.Sprintf("重新部署: %s\t", workload.Name))
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
			namespaceSearch,
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
	gFilteredNamespaces = []rancher.Namespace{}
	gNamespaceList.Refresh()
	gWorkloadSearch.SetText("")

	gWorkloads = []rancher.Workload{}
	gFilteredWorkloads = []rancher.Workload{}
	gWorkloadData.Set([]string{})
	gWorkloadList.Refresh()
	gWorkloadSearch.SetText("")
}

func selectNameSpace(namespace rancher.Namespace) {
	// 清空工作负载搜索框
	gWorkloadSearch.SetText("")

	gSelectedNamespace = namespace
	gEnvironment, _ = rancher.GetEnvironmentFromConfig(gConfig, gSelectedNamespace.Environment)
	workloads, _ := gDb.GetWorkloadsByNamespace(namespace.Name)
	gWorkloads = append(workloads)
	gFilteredWorkloads = filterWorkloads(gWorkloads, "")

	// 清除已选择的workloads（多选和单选）
	gSelectedWorkloads = []rancher.Workload{}
	gSelectedWorkload = rancher.Workload{} // 清除单选

	// 更新绑定数据
	workloadNames := make([]string, len(gFilteredWorkloads))
	for i, w := range gFilteredWorkloads {
		workloadNames[i] = w.Name
	}
	gWorkloadData.Set(workloadNames)
	podCount, _ := gDb.GetPodCountByEnvNamespace(gSelectedNamespace.Environment, gSelectedNamespace.Name)

	unselectWorkloadSingle()
	var info strings.Builder
	info.WriteString(fmt.Sprintf("环境: %s\n", gEnvironment.Name))
	info.WriteString(fmt.Sprintf("命名空间: %s\n", gSelectedNamespace.Name))
	info.WriteString(fmt.Sprintf("项目: %s\n", gSelectedNamespace.Project))
	info.WriteString(fmt.Sprintf("描述: %s\n", gSelectedNamespace.Description))
	info.WriteString(fmt.Sprintf("pod数量: %d\n", podCount))
	gInfoArea.SetText(info.String())
}

func selectWorkloadSingle(workload rancher.Workload) {
	gSelectedWorkload = workload

	podList, _ := gDb.GetPodsByEnvNamespaceWorkload(gSelectedWorkload.Environment, gSelectedWorkload.Namespace, gSelectedWorkload.Name)

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

	// 只有当 NodePort 不为空时才显示，并按逗号分隔成多行
	if workload.NodePort != "" {
		info.WriteString("端口访问:\n")
		ports := strings.Split(workload.NodePort, ",")
		ip := gEnvironment.Ip
		for _, port := range ports {
			info.WriteString(fmt.Sprintf("  %s:%s\n", ip, strings.TrimSpace(port)))
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
	gInfoArea.SetText(info.String())
}

func unselectWorkloadSingle() {
	gWorkloadList.UnselectAll()
	gSelectedWorkload = rancher.Workload{}
}

// 添加过滤函数
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

// 添加新的函数来更新信息区域显示多选内容
func updateInfoAreaForMultiSelect() {
	var info strings.Builder
	info.WriteString(fmt.Sprintf("已选择 %d 个服务:\n", len(gSelectedWorkloads)))

	for _, workload := range gSelectedWorkloads {
		info.WriteString(fmt.Sprintf("\n服务名称: %s\n", workload.Name))
		info.WriteString(fmt.Sprintf("命名空间: %s\n", workload.Namespace))
		info.WriteString(fmt.Sprintf("镜像: %s\n", workload.Image))
	}

	gInfoArea.SetText(info.String())
}
