package ui

import (
	"RancherMan/ui/component"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// UI组件全局变量
var (
	gNamespaceList    *widget.List
	gNamespaceSearch  *widget.Entry
	gWorkloadList     *component.MultiSelectList
	gWorkloadSearch   *widget.Entry
	gInfoArea         *widget.Entry
	gApp              fyne.App
	gTaskStatusBar    *component.TaskStatusBar
	gOperationButtons []*widget.Button
)
