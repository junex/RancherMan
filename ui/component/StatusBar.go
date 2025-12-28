package component

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TaskStatusBar 任务状态栏组件
type TaskStatusBar struct {
	container     *fyne.Container // 改为私有字段
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

	statusBar := container.NewHBox(
		statusLabel,
		dotsLabel,
		widget.NewSeparator(),
		taskInfoLabel,
		widget.NewSeparator(),
		countLabel,
		widget.NewSeparator(),
		cancelButton,
	)

	statusBarWithPadding := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(statusBar),
	)

	bar := &TaskStatusBar{
		container:     statusBarWithPadding,
		statusLabel:   statusLabel,
		dotsLabel:     dotsLabel,
		taskInfoLabel: taskInfoLabel,
		countLabel:    countLabel,
		cancelButton:  cancelButton,
	}

	return bar
}

// GetContainer 返回可渲染的容器（添加这个方法）
func (bar *TaskStatusBar) GetContainer() *fyne.Container {
	return bar.container
}

// Update 更新状态栏显示
func (bar *TaskStatusBar) Update(status string, dots string, taskInfo string, current int, total int) {
	fmt.Printf("[StatusBar.Update] status=%s dots=%s taskInfo=%s current=%d total=%d\n", status, dots, taskInfo, current, total)

	bar.statusLabel.SetText(status)
	bar.dotsLabel.SetText(dots)
	bar.taskInfoLabel.SetText(taskInfo)

	if total > 0 {
		bar.countLabel.SetText(fmt.Sprintf("%d/%d", current, total))
	} else {
		bar.countLabel.SetText("")
	}

	hasRunning := (status == "执行")
	if hasRunning {
		bar.cancelButton.Enable()
	} else {
		bar.cancelButton.Disable()
	}

	// 刷新容器而不是自身
	bar.container.Refresh()
}
