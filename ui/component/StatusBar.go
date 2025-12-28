package component

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// TaskStatusBar 任务状态栏组件
type TaskStatusBar struct {
	*fyne.Container
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
	cancelButton.Disable() // 初始禁用

	// 状态栏布局: [状态] [点动效] | [任务信息] | [数量] | [取消按钮]
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

	// 添加内边距和顶部分隔线，确保状态栏有足够高度且可见
	statusBarWithPadding := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(statusBar),
	)

	bar := &TaskStatusBar{
		Container:     statusBarWithPadding,
		statusLabel:   statusLabel,
		dotsLabel:     dotsLabel,
		taskInfoLabel: taskInfoLabel,
		countLabel:    countLabel,
		cancelButton:  cancelButton,
	}

	return bar
}

// Update 更新状态栏显示
func (bar *TaskStatusBar) Update(status string, dots string, taskInfo string, current int, total int) {
	// 添加日志确认方法被调用
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
	bar.Refresh()
}
