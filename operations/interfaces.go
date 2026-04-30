package operations

// InfoDisplayer 信息展示区域接口
// 替代 *widget.Entry，用于信息区域的文本展示和读取
type InfoDisplayer interface {
	SetText(text string)
	Text() string
}

// ListRefresher 列表刷新接口
// *widget.List 已实现全部方法，可直接赋值
type ListRefresher interface {
	UnselectAll()
	ScrollToTop()
	Refresh()
	Select(id int)
}

// TextGetter 文本读取接口
// 替代搜索框的 .Text 字段访问
type TextGetter interface {
	Text() string
}

// UIDispatcher UI线程调度器接口
// 替代 fyne.Do，确保回调在主线程执行
type UIDispatcher interface {
	RunOnUI(fn func())
}
