package operations

import (
	"fmt"
	"strconv"

	"RancherMan/rancher"
	"RancherMan/ui/component"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// 全局变量，由main包设置
var (
	gDb                     *rancher.DatabaseManager
	gConfig                 map[string]interface{}
	gEnvironment            *rancher.Environment
	gJumpHostConfig         *rancher.JumpHostConfig
	gCloneIgnoreTagWorkload []string
	gTaskQueue              *rancher.TaskQueue
	gNamespaces             []rancher.Namespace
	gFilteredNamespaces     []rancher.Namespace
	gSelectedNamespace      rancher.Namespace
	gWorkloads              []rancher.Workload
	gFilteredWorkloads      []rancher.Workload
	gSelectedWorkloads      []rancher.Workload
	gNamespaceList          *widget.List
	gNamespaceSearch        *widget.Entry
	gWorkloadList           *component.MultiSelectList
	gWorkloadSearch         *widget.Entry
	gInfoArea               *widget.Entry
	gApp                    fyne.App
	gTaskStatusBar          *component.TaskStatusBar
	gOperationButtons       []*widget.Button
)

// SetDb 设置数据库管理器
func SetDb(db *rancher.DatabaseManager) {
	gDb = db
}

// GetDb 获取数据库管理器
func GetDb() *rancher.DatabaseManager {
	return gDb
}

// SetConfig 设置配置
func SetConfig(config map[string]interface{}) {
	gConfig = config
}

// GetConfig 获取配置
func GetConfig() map[string]interface{} {
	return gConfig
}

// SetEnvironment 设置环境
func SetEnvironment(env *rancher.Environment) {
	gEnvironment = env
}

// GetEnvironment 获取环境
func GetEnvironment() *rancher.Environment {
	return gEnvironment
}

// SetJumpHostConfig 设置跳板机配置
func SetJumpHostConfig(config *rancher.JumpHostConfig) {
	gJumpHostConfig = config
}

// GetJumpHostConfig 获取跳板机配置
func GetJumpHostConfig() *rancher.JumpHostConfig {
	return gJumpHostConfig
}

// SetCloneIgnoreTagWorkload 设置忽略标签的工作负载列表
func SetCloneIgnoreTagWorkload(list []string) {
	gCloneIgnoreTagWorkload = list
}

// GetCloneIgnoreTagWorkload 获取忽略标签的工作负载列表
func GetCloneIgnoreTagWorkload() []string {
	return gCloneIgnoreTagWorkload
}

// SetTaskQueue 设置任务队列
func SetTaskQueue(queue *rancher.TaskQueue) {
	gTaskQueue = queue
}

// GetTaskQueue 获取任务队列
func GetTaskQueue() *rancher.TaskQueue {
	return gTaskQueue
}

// SetNamespaces 设置命名空间列表
func SetNamespaces(namespaces []rancher.Namespace) {
	gNamespaces = namespaces
}

// GetNamespaces 获取命名空间列表
func GetNamespaces() []rancher.Namespace {
	return gNamespaces
}

// SetFilteredNamespaces 设置过滤后的命名空间列表
func SetFilteredNamespaces(namespaces []rancher.Namespace) {
	gFilteredNamespaces = namespaces
}

// GetFilteredNamespaces 获取过滤后的命名空间列表
func GetFilteredNamespaces() []rancher.Namespace {
	return gFilteredNamespaces
}

// GetSelectedNamespace 获取选中的命名空间
func GetSelectedNamespace() rancher.Namespace {
	return gSelectedNamespace
}

// SetSelectedNamespace 设置选中的命名空间
func SetSelectedNamespace(namespace rancher.Namespace) {
	gSelectedNamespace = namespace
}

// SetWorkloads 设置工作负载列表
func SetWorkloads(workloads []rancher.Workload) {
	gWorkloads = workloads
}

// GetWorkloads 获取工作负载列表
func GetWorkloads() []rancher.Workload {
	return gWorkloads
}

// SetFilteredWorkloads 设置过滤后的工作负载列表
func SetFilteredWorkloads(workloads []rancher.Workload) {
	gFilteredWorkloads = workloads
}

// GetFilteredWorkloads 获取过滤后的工作负载列表
func GetFilteredWorkloads() []rancher.Workload {
	return gFilteredWorkloads
}

// SetSelectedWorkloads 设置选中的工作负载列表
func SetSelectedWorkloads(workloads []rancher.Workload) {
	gSelectedWorkloads = workloads
}

// GetSelectedWorkloads 获取选中的工作负载列表
func GetSelectedWorkloads() []rancher.Workload {
	return gSelectedWorkloads
}

// SetNamespaceList 设置命名空间列表UI组件
func SetNamespaceList(list *widget.List) {
	gNamespaceList = list
}

// GetNamespaceList 获取命名空间列表UI组件
func GetNamespaceList() *widget.List {
	return gNamespaceList
}

// SetNamespaceSearch 设置命名空间搜索框
func SetNamespaceSearch(entry *widget.Entry) {
	gNamespaceSearch = entry
}

// GetNamespaceSearch 获取命名空间搜索框
func GetNamespaceSearch() *widget.Entry {
	return gNamespaceSearch
}

// SetWorkloadList 设置工作负载列表UI组件
func SetWorkloadList(list *component.MultiSelectList) {
	gWorkloadList = list
}

// GetWorkloadList 获取工作负载列表UI组件
func GetWorkloadList() *component.MultiSelectList {
	return gWorkloadList
}

// SetWorkloadSearch 设置工作负载搜索框
func SetWorkloadSearch(entry *widget.Entry) {
	gWorkloadSearch = entry
}

// GetWorkloadSearch 获取工作负载搜索框
func GetWorkloadSearch() *widget.Entry {
	return gWorkloadSearch
}

// SetWorkloadSearchText 设置工作负载搜索框文本
func SetWorkloadSearchText(text string) {
	if gWorkloadSearch != nil {
		gWorkloadSearch.SetText(text)
	}
}

// SetInfoArea 设置信息区域
func SetInfoArea(entry *widget.Entry) {
	gInfoArea = entry
}

// GetInfoArea 获取信息区域
func GetInfoArea() *widget.Entry {
	return gInfoArea
}

// SetTaskStatusBar 设置任务状态栏
func SetTaskStatusBar(statusBar *component.TaskStatusBar) {
	gTaskStatusBar = statusBar
}

// GetTaskStatusBar 获取任务状态栏
func GetTaskStatusBar() *component.TaskStatusBar {
	return gTaskStatusBar
}

// SetOperationButtons 设置操作按钮列表
func SetOperationButtons(buttons []*widget.Button) {
	gOperationButtons = buttons
}

// GetOperationButtons 获取操作按钮列表
func GetOperationButtons() []*widget.Button {
	return gOperationButtons
}

// LoadConfig 加载配置
func LoadConfig(showSuccessTip bool) {
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
	// 解析 clone_ignore_tag_workload
	if ignoreList, exists := gConfig["clone_ignore_tag_workload"].([]interface{}); exists {
		gCloneIgnoreTagWorkload = make([]string, len(ignoreList))
		for i, item := range ignoreList {
			gCloneIgnoreTagWorkload[i] = item.(string)
		}
	}
	if err != nil {
		if gInfoArea != nil {
			gInfoArea.SetText(fmt.Sprintf("从数据库读取配置时出错: %v", err))
		}
	} else {
		if showSuccessTip && gInfoArea != nil {
			gInfoArea.SetText("配置已成功加载")
		}
	}
}

// InitData 初始化数据
func InitData() {
	namespaces, _ := gDb.GetAllNamespacesDetail()
	gNamespaces = append(namespaces)
	gFilteredNamespaces = append(gNamespaces)
	gSelectedNamespace = rancher.Namespace{}
	fmt.Printf("初始化数据完成，共有命名空间 %d 个\n", len(gNamespaces))

	if gNamespaceList != nil {
		gNamespaceList.UnselectAll()
		gNamespaceList.ScrollToTop()
		gNamespaceList.Refresh()
	}
	if gNamespaceSearch != nil {
		gNamespaceSearch.SetText("")
	}

	gWorkloads = []rancher.Workload{}
	gFilteredWorkloads = []rancher.Workload{}

	if gWorkloadList != nil {
		gWorkloadList.RefreshList()
	}
	if gWorkloadSearch != nil {
		gWorkloadSearch.SetText("")
	}
}
