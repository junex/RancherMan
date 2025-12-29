package component

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// TaskStatusBar 任务状态栏组件
type TaskStatusBar struct {
	container     *fyne.Container
	statusLabel   *widget.Label
	dotsLabel     *widget.Label
	taskInfoLabel *widget.Label
	countLabel    *widget.Label
	cancelButton  *widget.Button
}

// NewTaskStatusBar 创建任务状态栏
func NewTaskStatusBar(onCancel func()) *TaskStatusBar {
	statusLabel := widget.NewLabel("空闲")
	dotsLabel := widget.NewLabel("")
	taskInfoLabel := widget.NewLabel("")
	countLabel := widget.NewLabel("")

	cancelButton := widget.NewButton("取消所有", onCancel)
	cancelButton.Disable()

	// === 左侧内容 ===
	left := container.NewHBox(
		statusLabel,
		widget.NewSeparator(),
		taskInfoLabel,
		dotsLabel,
	)

	// === 右侧内容（始终靠右） ===
	right := container.NewHBox(
		countLabel,
		widget.NewSeparator(),
		cancelButton,
	)

	// === 整体状态栏 ===
	statusBar := container.NewHBox(
		left,
		layout.NewSpacer(), // 🔥 核心：把 right 顶到最右
		right,
	)

	// 上边框 + 内边距
	root := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(statusBar),
	)

	return &TaskStatusBar{
		container:     root,
		statusLabel:   statusLabel,
		dotsLabel:     dotsLabel,
		taskInfoLabel: taskInfoLabel,
		countLabel:    countLabel,
		cancelButton:  cancelButton,
	}
}

// GetContainer 返回可渲染的容器
func (bar *TaskStatusBar) GetContainer() *fyne.Container {
	return bar.container
}

// Update 更新状态栏显示
func (bar *TaskStatusBar) Update(
	status string,
	dots string,
	taskInfo string,
	current int,
	total int,
) {
	bar.statusLabel.SetText(status)
	bar.dotsLabel.SetText(dots)
	bar.taskInfoLabel.SetText(taskInfo)

	if total > 0 {
		bar.countLabel.SetText(fmt.Sprintf("%d/%d", current, total))
	} else {
		bar.countLabel.SetText("")
	}

	if status == "执行" {
		bar.cancelButton.Enable()
	} else {
		bar.cancelButton.Disable()
	}

	bar.container.Refresh()
}
