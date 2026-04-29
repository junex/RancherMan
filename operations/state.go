package operations

import (
	"sync"

	"RancherMan/rancher"
	"RancherMan/ui/component"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
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
	NamespaceList          *widget.List
	NamespaceSearch        *widget.Entry
	WorkloadList           *component.MultiSelectList
	WorkloadSearch         *widget.Entry
	InfoArea               *widget.Entry
	App                    fyne.App
	TaskStatusBar          *component.TaskStatusBar
	OperationButtons       []*widget.Button
}

var appState = &AppState{}
