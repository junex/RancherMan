package operations

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"RancherMan/rancher"
)

// SetDb 设置数据库管理器
func SetDb(db *rancher.DatabaseManager) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.Db = db
}

// GetDb 获取数据库管理器
func GetDb() *rancher.DatabaseManager {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.Db
}

// SetConfig 设置配置
func SetConfig(config map[string]interface{}) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.Config = config
}

// GetConfig 获取配置
func GetConfig() map[string]interface{} {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.Config
}

// SetEnvironment 设置环境
func SetEnvironment(env *rancher.Environment) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.Environment = env
}

// GetEnvironment 获取环境
func GetEnvironment() *rancher.Environment {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.Environment
}

// SetJumpHostConfig 设置跳板机配置
func SetJumpHostConfig(config *rancher.JumpHostConfig) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.JumpHostConfig = config
}

// GetJumpHostConfig 获取跳板机配置
func GetJumpHostConfig() *rancher.JumpHostConfig {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.JumpHostConfig
}

// SetCloneIgnoreTagWorkload 设置忽略标签的工作负载列表
func SetCloneIgnoreTagWorkload(list []string) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.CloneIgnoreTagWorkload = list
}

// GetCloneIgnoreTagWorkload 获取忽略标签的工作负载列表
func GetCloneIgnoreTagWorkload() []string {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.CloneIgnoreTagWorkload
}

// SetTaskQueue 设置任务队列
func SetTaskQueue(queue *rancher.TaskQueue) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.TaskQueue = queue
}

// GetTaskQueue 获取任务队列
func GetTaskQueue() *rancher.TaskQueue {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.TaskQueue
}

// SetNamespaces 设置命名空间列表
func SetNamespaces(namespaces []rancher.Namespace) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.Namespaces = namespaces
}

// GetNamespaces 获取命名空间列表
func GetNamespaces() []rancher.Namespace {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.Namespaces
}

// SetFilteredNamespaces 设置过滤后的命名空间列表
func SetFilteredNamespaces(namespaces []rancher.Namespace) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.FilteredNamespaces = namespaces
}

// GetFilteredNamespaces 获取过滤后的命名空间列表
func GetFilteredNamespaces() []rancher.Namespace {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.FilteredNamespaces
}

// GetSelectedNamespace 获取选中的命名空间
func GetSelectedNamespace() rancher.Namespace {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.SelectedNamespace
}

// SetSelectedNamespace 设置选中的命名空间
func SetSelectedNamespace(namespace rancher.Namespace) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.SelectedNamespace = namespace
}

// SetWorkloads 设置工作负载列表
func SetWorkloads(workloads []rancher.Workload) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.Workloads = workloads
}

// GetWorkloads 获取工作负载列表
func GetWorkloads() []rancher.Workload {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.Workloads
}

// SetFilteredWorkloads 设置过滤后的工作负载列表
func SetFilteredWorkloads(workloads []rancher.Workload) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.FilteredWorkloads = workloads
}

// GetFilteredWorkloads 获取过滤后的工作负载列表
func GetFilteredWorkloads() []rancher.Workload {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.FilteredWorkloads
}

// SetSelectedWorkloads 设置选中的工作负载列表
func SetSelectedWorkloads(workloads []rancher.Workload) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.SelectedWorkloads = workloads
}

// GetSelectedWorkloads 获取选中的工作负载列表
func GetSelectedWorkloads() []rancher.Workload {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.SelectedWorkloads
}

// SetNamespaceList 设置命名空间列表UI组件
func SetNamespaceList(list ListRefresher) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.NamespaceList = list
}

// GetNamespaceList 获取命名空间列表UI组件
func GetNamespaceList() ListRefresher {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.NamespaceList
}

// SetNamespaceSearch 设置命名空间搜索框
func SetNamespaceSearch(entry TextGetter) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.NamespaceSearch = entry
}

// GetNamespaceSearch 获取命名空间搜索框
func GetNamespaceSearch() TextGetter {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.NamespaceSearch
}

// SetInfoArea 设置信息区域
func SetInfoArea(display InfoDisplayer) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.InfoArea = display
}

// GetInfoArea 获取信息区域
func GetInfoArea() InfoDisplayer {
	appState.mu.RLock()
	defer appState.mu.RUnlock()
	return appState.InfoArea
}

// SetUIDispatcher 设置UI线程调度器
func SetUIDispatcher(d UIDispatcher) {
	appState.mu.Lock()
	defer appState.mu.Unlock()
	appState.UIDispatcher = d
}

// LoadConfig 加载配置
func LoadConfig(showSuccessTip bool) {
	appState.mu.Lock()
	defer appState.mu.Unlock()

	var err error
	appState.Config, err = rancher.LoadConfigFromDb(appState.Db)

	if appState.Config != nil {
		if jumpHost, exists := appState.Config["jump_host"].(map[interface{}]interface{}); exists {
			appState.JumpHostConfig = &rancher.JumpHostConfig{
				Ip:       jumpHost["ip"].(string),
				Port:     strconv.Itoa(jumpHost["port"].(int)),
				Username: jumpHost["username"].(string),
				Password: jumpHost["password"].(string),
				RootPath: jumpHost["root_path"].(string),
			}
		}
		if ignoreList, exists := appState.Config["clone_ignore_tag_workload"].([]interface{}); exists {
			appState.CloneIgnoreTagWorkload = make([]string, len(ignoreList))
			for i, item := range ignoreList {
				appState.CloneIgnoreTagWorkload[i] = item.(string)
			}
		}
	}

	if err != nil {
		if appState.InfoArea != nil {
			appState.InfoArea.SetText(fmt.Sprintf("从数据库读取配置时出错: %v", err))
		}
	} else {
		if showSuccessTip && appState.InfoArea != nil {
			appState.InfoArea.SetText("配置已成功加载")
		}
	}
}

// InitData 初始化数据
func InitData() {
	// 数据部分：持写锁
	appState.mu.Lock()
	namespaces, _ := appState.Db.GetAllNamespacesDetail()
	appState.Namespaces = namespaces

	searchText := ""
	if appState.NamespaceSearch != nil {
		searchText = appState.NamespaceSearch.Text()
	}
	appState.FilteredNamespaces = FilterNamespaces(appState.Namespaces, searchText)
	log.Printf("[InitData] 初始化数据完成，共有命名空间 %d 个\n", len(appState.Namespaces))

	selectIndex := -1
	for i := range appState.FilteredNamespaces {
		ns := appState.FilteredNamespaces[i]
		if appState.SelectedNamespace.Name == ns.Name && appState.SelectedNamespace.Environment == ns.Environment {
			selectIndex = i
			break
		}
	}
	if selectIndex < 0 {
		appState.SelectedNamespace = rancher.Namespace{}
	}

	nsList := appState.NamespaceList
	dispatcher := appState.UIDispatcher
	appState.mu.Unlock()

	// UI 部分：必须通过 dispatcher 在主 goroutine 执行
	if nsList != nil && dispatcher != nil {
		dispatcher.RunOnUI(func() {
			if selectIndex < 0 {
				nsList.UnselectAll()
				nsList.ScrollToTop()
				nsList.Refresh()
			} else {
				nsList.Refresh()
				time.AfterFunc(time.Millisecond*20, func() {
					dispatcher.RunOnUI(func() {
						nsList.Select(selectIndex)
					})
				})
			}
		})
	}
}

// GetEnvironmentFromConfig 根据环境名称从配置中获取环境信息
func GetEnvironmentFromConfig(config map[string]interface{}, envName string) (*rancher.Environment, error) {
	return rancher.GetEnvironmentFromConfig(config, envName)
}

// SaveConfig 保存配置到数据库
func SaveConfig(content string) {
	appState.mu.RLock()
	db := appState.Db
	appState.mu.RUnlock()
	rancher.SaveConfigToDb(db, content)
}

// GetConfigContent 获取配置内容
func GetConfigContent() (string, error) {
	appState.mu.RLock()
	db := appState.Db
	appState.mu.RUnlock()
	return db.GetConfigContent(1)
}

// GetWorkloadsByNamespace 根据命名空间获取工作负载列表
func GetWorkloadsByNamespace(namespace string) ([]rancher.Workload, error) {
	appState.mu.RLock()
	db := appState.Db
	appState.mu.RUnlock()
	return db.GetWorkloadsByNamespace(namespace)
}

// GetAllNamespacesDetail 获取所有命名空间详细信息
func GetAllNamespacesDetail() ([]rancher.Namespace, error) {
	appState.mu.RLock()
	db := appState.Db
	appState.mu.RUnlock()
	return db.GetAllNamespacesDetail()
}
