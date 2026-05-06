package component

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"log"
)

// MultiSelectList 是一个支持多选的列表组件
type MultiSelectList struct {
	widget.BaseWidget // 添加这个基础组件
	list              *widget.List
	onSelectMulti     func(ids []int)
	selectedIds       map[int]struct{}
}

func NewList(length func() int, createItem func() fyne.CanvasObject, updateItem func(widget.ListItemID, fyne.CanvasObject)) *MultiSelectList {
	ml := &MultiSelectList{
		onSelectMulti: func(ids []int) {},
		selectedIds:   make(map[int]struct{}),
	}

	list := widget.NewList(length, createItem, updateItem)
	ml.list = list

	list.OnSelected = func(id int) {
		// 检查id是否已存在于selectedIds中
		if _, exists := ml.selectedIds[id]; exists {
			// 如果存在则删除
			delete(ml.selectedIds, id)
		} else {
			// 如果不存在则添加
			ml.selectedIds[id] = struct{}{}
		}
		// 调用多选回调函数
		ml.MultiSelected()
		list.Unselect(id)
		list.RefreshItem(id)
	}

	ml.ExtendBaseWidget(ml)
	return ml
}

// CreateRenderer 实现 fyne.Widget 接口
func (t *MultiSelectList) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.list)
}

// MinSize 返回最小尺寸，委托给内部的 list
func (t *MultiSelectList) MinSize() fyne.Size {
	return t.list.MinSize()
}

// 以下是委托方法，转发到内部的 list
func (t *MultiSelectList) Select(id widget.ListItemID) {
	t.list.Select(id)
}

func (t *MultiSelectList) Unselect(id widget.ListItemID) {
	t.list.Unselect(id)
}

func (t *MultiSelectList) UnselectAll() {
	t.list.UnselectAll()
}

func (t *MultiSelectList) ScrollToTop() {
	t.list.ScrollToTop()
}

func (t *MultiSelectList) RefreshItem(id widget.ListItemID) {
	t.list.RefreshItem(id)
}

func (t *MultiSelectList) Length() int {
	return t.list.Length()
}

// 多选相关方法
func (t *MultiSelectList) OnMultiSelected(selectMulti func(ids []int)) {
	t.onSelectMulti = selectMulti
}

func (t *MultiSelectList) MultiSelected() {
	if t.onSelectMulti != nil {
		ids := make([]int, 0, len(t.selectedIds))
		for id := range t.selectedIds {
			ids = append(ids, id)
		}
		t.onSelectMulti(ids)
	}
}

func (t *MultiSelectList) UnselectMulti() {
	t.selectedIds = make(map[int]struct{})
	t.MultiSelected()
	log.Println("UnselectMulti")
}

func (t *MultiSelectList) MultiSelectedOne(id widget.ListItemID) {
	t.selectedIds[id] = struct{}{}
	t.MultiSelected()
	log.Println("MultiSelectedOne id:", id)
}

func (t *MultiSelectList) UnMultiSelectedOne(id widget.ListItemID) {
	delete(t.selectedIds, id)
	t.MultiSelected()
	log.Println("UnMultiSelectedOne id:", id)
}

func (t *MultiSelectList) RefreshList() {
	t.ScrollToTop()
	for i := range t.Length() {
		if i < 10 {
			t.RefreshItem(i)
		}
	}
}

// RefreshAllItems 刷新列表全部可见项，不改变滚动位置
func (t *MultiSelectList) RefreshAllItems() {
	t.BaseWidget.Refresh()
}

func (t *MultiSelectList) IsSelected(pos int) bool {
	return t.selectedIds[pos] != struct{}{}
}
