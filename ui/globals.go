package ui

import (
	"RancherMan/ui/component"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// UI组件全局变量
var (
	gNamespaceList   *widget.List
	gNamespaceSearch *widget.Entry
	gWorkloadList    *component.MultiSelectList
	gWorkloadSearch  *widget.Entry
	gInfoArea        *widget.Entry
	gApp             fyne.App
	gInfoAreaStatus  InfoAreaStatus = InfoAreaStatusInfo // 默认显示信息
)

// InfoAreaStatus 信息区域状态
type InfoAreaStatus int

const (
	InfoAreaStatusConfig InfoAreaStatus = iota // 显示配置
	InfoAreaStatusInfo                         // 显示信息
)
