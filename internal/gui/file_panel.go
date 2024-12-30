package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// FilePanel 表示文件面板
type FilePanel struct {
	container *fyne.Container
	list     *widget.List
	path     string
}

// NewFilePanel 创建新的文件面板
func NewFilePanel() *FilePanel {
	panel := &FilePanel{}
	
	// 创建路径输入框
	pathEntry := widget.NewEntry()
	pathEntry.SetPlaceHolder("当前路径")
	
	// 创建文件列表
	panel.list = widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(nil),
				widget.NewLabel("文件名"),
				widget.NewLabel("大小"),
				widget.NewLabel("修改时间"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
		},
	)
	
	// 创建工具栏
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(nil, func() { /* 刷新 */ }),
		widget.NewToolbarAction(nil, func() { /* 新建文件夹 */ }),
		widget.NewToolbarAction(nil, func() { /* 删除 */ }),
	)
	
	// 组合所有元素
	panel.container = container.NewBorder(
		container.NewVBox(pathEntry, toolbar),
		nil, nil, nil,
		panel.list,
	)
	
	return panel
}

// GetContainer 返回面板的容器
func (p *FilePanel) GetContainer() fyne.CanvasObject {
	return p.container
}

// SetPath 设置当前路径
func (p *FilePanel) SetPath(path string) {
	p.path = path
	// TODO: 更新文件列表
}

// RefreshFiles 刷新文件列表
func (p *FilePanel) RefreshFiles() {
	// TODO: 实现文件列表刷新逻辑
}
