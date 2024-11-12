package ui

import (
	"RancherMan/rancher"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// 添加新的函数来创建和显示自定义对话框
func ShowSelectNamespaceDialog(window fyne.Window, db *rancher.DatabaseManager, needNewTag bool, onSelect func(namespace rancher.Namespace, tag string)) {
	// 创建搜索框
	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("搜索命名空间...")

	namespaces, _ := db.GetAllNamespacesDetail()
	// 添加一个变量来存储过滤后的命名空间
	filteredNamespaces := namespaces
	selectedNamespace := rancher.Namespace{}

	// 创建列表
	list := widget.NewList(
		func() int { return len(filteredNamespaces) },
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			label := item.(*widget.Label)
			label.SetText(filteredNamespaces[id].Name)
		},
	)

	// 添加列表选择事件
	list.OnSelected = func(id widget.ListItemID) {
		selectedNamespace = filteredNamespaces[id]
	}

	// 添加搜索框事件
	searchEntry.OnChanged = func(searchText string) {
		filteredNamespaces = nil
		for _, ns := range namespaces {
			if strings.Contains(strings.ToLower(ns.Name), strings.ToLower(searchText)) {
				filteredNamespaces = append(filteredNamespaces, ns)
			}
		}
		list.Refresh()
		if len(filteredNamespaces) >= 1 {
			list.UnselectAll()
			list.Select(0)
		} else {
			selectedNamespace = rancher.Namespace{}
			list.UnselectAll()
		}
	}

	// 创建tag输入框（仅当needNewTag为true时）
	var tagEntry *widget.Entry
	var content *fyne.Container
	if needNewTag {
		tagEntry = widget.NewEntry()
		tagEntry.SetPlaceHolder("输入新的tag...")

		content = container.NewBorder(
			searchEntry, // 顶部放置搜索框
			tagEntry,    // 底部放置tag输入框
			nil,
			nil,
			list, // 中间区域放置列表
		)
	} else {
		content = container.NewBorder(
			searchEntry, // 顶部放置搜索框
			nil,         // 不需要tag输入框
			nil,
			nil,
			list, // 中间区域放置列表
		)
	}

	// 创建对话框
	dialog := dialog.NewCustom("选择目标命名空间", "取消",
		content,
		window,
	)

	submitFunc := func(text string) {
		dialog.Hide()
		var tagText string
		if tagEntry != nil {
			tagText = tagEntry.Text
		}
		onSelect(selectedNamespace, tagText)
	}
	searchEntry.OnSubmitted = submitFunc
	tagEntry.OnSubmitted = submitFunc

	// 添加确定按钮
	dialog.SetButtons([]fyne.CanvasObject{
		widget.NewButton("确定", func() {
			dialog.Hide()
			var tagText string
			if tagEntry != nil {
				tagText = tagEntry.Text
			}
			onSelect(selectedNamespace, tagText)
		}),
		widget.NewButton("取消", func() {
			dialog.Hide()
		}),
	})

	dialog.Resize(fyne.NewSize(300, 400))
	dialog.Show()
}
