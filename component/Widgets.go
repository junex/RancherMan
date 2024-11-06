package component

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// MultiSelectList 是一个可点击的标签组件
type MultiSelectList struct {
	*widget.List
	onSelectMulti func(ids []int)
	selectedIds   map[int]struct{}
}

func NewList(length func() int, createItem func() fyne.CanvasObject, updateItem func(widget.ListItemID, fyne.CanvasObject)) *MultiSelectList {
	list := widget.NewList(length, createItem, updateItem)
	list.ExtendBaseWidget(list)
	ml := &MultiSelectList{List: list, onSelectMulti: func(ids []int) {}, selectedIds: make(map[int]struct{})}
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
	return ml
}

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
}

func (t *MultiSelectList) MultiSelectedOne(id widget.ListItemID) {
	t.selectedIds[id] = struct{}{}
	t.MultiSelected()
}

func (t *MultiSelectList) UnMultiSelectedOne(id widget.ListItemID) {
	delete(t.selectedIds, id)
	t.MultiSelected()
}

func (t *MultiSelectList) RefreshList() {
	t.ScrollToTop()
	for i := range t.Length() {
		if i < 10 {
			t.RefreshItem(i)
		}
	}
}
