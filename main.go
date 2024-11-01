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
	"fyne.io/fyne/v2/widget"
	"github.com/flopp/go-findfont"
)

// 数据
var gNamespaces []string
var gFilteredNamespaces []string
var gSelectedNamespace string
var gWorkloads []rancher.Workload
var gFilteredWorkloads []rancher.Workload
var gSelectedWorkload rancher.Workload

// UI组件
var gNamespaceList *widget.List
var gWorkloadList *widget.List
var gInfoArea *widget.Entry

// 数据库
var db *rancher.DatabaseManager

func main() {
	//// 创建数据库管理器实例
	database, err := rancher.NewDatabaseManager("")
	db = database
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	initFont()
	window := initView()
	go refreshNamespace()
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
	myWindow := myApp.NewWindow("RancherMan")

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
			item.(*widget.Label).SetText(gFilteredNamespaces[id])
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

	// 创建服务搜索框
	workloadSearch := widget.NewEntry()
	workloadSearch.SetPlaceHolder("搜索服务...")

	// 创建过滤后的服务列表
	gFilteredWorkloads = gWorkloads

	// 创建中间的服务列表
	gWorkloadList = widget.NewList(
		func() int { return len(gFilteredWorkloads) },
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Service")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			item.(*widget.Label).SetText(gFilteredWorkloads[id].Name)
		},
	)

	// 添加服务搜索功能
	workloadSearch.OnChanged = func(s string) {
		gFilteredWorkloads = filterWorkloads(gFilteredWorkloads, s)
		gWorkloadList.Refresh()
	}

	gWorkloadList.OnSelected = func(id widget.ListItemID) {
		selectWorkload(gFilteredWorkloads[id])
	}

	workloadScroll := container.NewScroll(gWorkloadList)
	workloadScroll.SetMinSize(fyne.NewSize(200, 300))

	// 创建右侧的信息区域
	gInfoArea = widget.NewMultiLineEntry()
	gInfoArea.SetText("")
	gInfoArea.SetMinRowsVisible(15)
	// 将 InfoArea 放入固定大小的容器中
	infoContainer := container.NewScroll(gInfoArea)
	infoContainer.SetMinSize(fyne.NewSize(500, 330))

	// 创建按钮
	button1 := widget.NewButton("加载配置", func() {})
	button2 := widget.NewButton("更新数据", func() {})
	button3 := widget.NewButton("打开", func() {})
	button4 := widget.NewButton("关闭", func() {})
	button5 := widget.NewButton("重新部署", func() {

	})

	// 创建布局
	content := container.NewHBox(
		container.NewVBox(
			widget.NewLabel("命名空间"),
			namespaceSearch,
			namespaceScroll,
		),
		container.NewVBox(
			widget.NewLabel("服务"),
			workloadSearch,
			workloadScroll,
		),
		container.NewVBox(
			container.NewHBox(button1, button2, button3, button4, button5),
			infoContainer, // 使用包装后的容器
		),
	)
	myWindow.SetContent(content)
	return myWindow
}

func refreshNamespace() {
	namespaces, _ := db.GetAllNamespaces()
	gNamespaces = append(namespaces)
	gFilteredNamespaces = filterNamespaces(gNamespaces, "")
	gNamespaceList.Refresh()
}

func selectNameSpace(namespace string) {
	gSelectedNamespace = namespace
	workloads, _ := db.GetWorkloadsByNamespace(namespace)
	gWorkloads = append(workloads)
	gFilteredWorkloads = filterWorkloads(gWorkloads, "")
	gWorkloadList.Refresh()
}

func selectWorkload(workload rancher.Workload) {
	gSelectedWorkload = workload

	// 构建信息字符串
	var info strings.Builder

	info.WriteString(fmt.Sprintf("环境: %s\n", workload.Environment))
	info.WriteString(fmt.Sprintf("命名空间: %s\n", workload.Namespace))
	info.WriteString(fmt.Sprintf("名称: %s\n", workload.Name))
	info.WriteString(fmt.Sprintf("镜像: %s\n", workload.Image))

	// 只有当 NodePort 不为空时才显示
	if workload.NodePort != "" {
		info.WriteString(fmt.Sprintf("端口: %s\n", workload.NodePort))
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

// 添加过滤函数
func filterNamespaces(items []string, filter string) []string {
	if filter == "" {
		return items
	}
	var filtered []string
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), strings.ToLower(filter)) {
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
