package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// entryAdapter 包装 *widget.Entry，使其满足 operations.InfoDisplayer 和 operations.TextGetter 接口
// widget.Entry.Text 是公开字段不是方法，所以必须通过适配器提供 Text() 方法
// 所有 Fyne widget 操作都通过 fyne.Do 调度到 UI 主线程，确保并发安全
type entryAdapter struct {
	entry *widget.Entry
}

func (a *entryAdapter) SetText(text string) {
	fyne.Do(func() {
		a.entry.SetText(text)
	})
}

func (a *entryAdapter) Text() string {
	return a.entry.Text
}

// fyneDispatcher 包装 fyne.Do，使其满足 operations.UIDispatcher 接口
type fyneDispatcher struct{}

func (d *fyneDispatcher) RunOnUI(fn func()) {
	fyne.Do(fn)
}
