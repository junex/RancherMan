package operations

import (
	"sync"

	"RancherMan/rancher"
)

// AppState 应用全局状态，使用 RWMutex 保护并发读写
type AppState struct {
	mu sync.RWMutex

	Db                     *rancher.DatabaseManager
	Config                 map[string]interface{}
	Environment            *rancher.Environment
	JumpHostConfig         *rancher.JumpHostConfig
	CloneIgnoreTagWorkload []string
	TaskQueue              *rancher.TaskQueue
	Namespaces             []rancher.Namespace
	FilteredNamespaces     []rancher.Namespace
	SelectedNamespace      rancher.Namespace
	Workloads              []rancher.Workload
	FilteredWorkloads      []rancher.Workload
	SelectedWorkloads      []rancher.Workload

	// UI依赖（通过接口注入，解除对 Fyne 的具体依赖）
	NamespaceList   ListRefresher
	NamespaceSearch TextGetter
	InfoArea        InfoDisplayer
	UIDispatcher    UIDispatcher
}

var appState = &AppState{}
