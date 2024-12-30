package gui

import (
	"fmt"
	"path/filepath"
	"xftp798/internal/transfer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// FilePanel 表示文件面板
type FilePanel struct {
	container    *fyne.Container
	list         *widget.List
	pathEntry    *widget.Entry
	fileSystem   *transfer.FileSystem
	currentFiles []transfer.FileInfo
	window       fyne.Window
	selectedItem int
	progressBar  *widget.ProgressBar
	transferMgr  *transfer.TransferManager
	onTransfer   func(source string, targetPanel *FilePanel) // 传输回调
}

// NewFilePanel 创建新的文件面板
func NewFilePanel(window fyne.Window) *FilePanel {
	panel := &FilePanel{
		window:       window,
		fileSystem:   transfer.NewFileSystem(""),
		selectedItem: -1,
	}

	// 创建进度条
	panel.progressBar = widget.NewProgressBar()
	panel.progressBar.Hide()

	// 创建传输管理器
	panel.transferMgr = transfer.NewTransferManager(func(progress transfer.TransferProgress) {
		panel.progressBar.Value = progress.Percentage / 100
		if progress.IsCompleted {
			panel.progressBar.Hide()
			panel.RefreshFiles()
		} else {
			panel.progressBar.Show()
		}
	})

	// 创建路径输入框
	panel.pathEntry = widget.NewEntry()
	panel.pathEntry.SetText(panel.fileSystem.GetCurrentPath())
	panel.pathEntry.OnSubmitted = func(path string) {
		panel.SetPath(path)
	}

	// 创建文件列表
	panel.list = widget.NewList(
		func() int {
			return len(panel.currentFiles)
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewIcon(theme.FileIcon()),
				widget.NewLabel("文件名"),
				widget.NewLabel("大小"),
				widget.NewLabel("修改时间"),
			)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(panel.currentFiles) {
				return
			}
			file := panel.currentFiles[id]

			// 更新图标
			icon := item.(*fyne.Container).Objects[0].(*widget.Icon)
			if file.IsDir {
				icon.SetResource(theme.FolderIcon())
			} else {
				icon.SetResource(theme.FileIcon())
			}

			// 更新文件信息
			item.(*fyne.Container).Objects[1].(*widget.Label).SetText(file.Name)
			item.(*fyne.Container).Objects[2].(*widget.Label).SetText(formatSize(file.Size))
			item.(*fyne.Container).Objects[3].(*widget.Label).SetText(file.ModTime.Format("2006-01-02 15:04:05"))
		},
	)

	// 添加列表选择事件
	panel.list.OnSelected = func(id widget.ListItemID) {
		panel.selectedItem = int(id)
		if id >= len(panel.currentFiles) {
			return
		}
		file := panel.currentFiles[id]
		if file.IsDir {
			panel.SetPath(file.Path)
			panel.selectedItem = -1
		}
	}

	// 创建工具栏
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			panel.RefreshFiles()
		}),
		widget.NewToolbarAction(theme.FolderNewIcon(), func() {
			entry := widget.NewEntry()
			entry.SetPlaceHolder("输入文件夹名称")
			dialog.ShowForm("新建文件夹", "创建", "取消",
				[]*widget.FormItem{
					widget.NewFormItem("名称", entry),
				},
				func(confirm bool) {
					if confirm {
						path := filepath.Join(panel.fileSystem.GetCurrentPath(), entry.Text)
						if err := panel.fileSystem.CreateDirectory(path); err != nil {
							dialog.ShowError(err, panel.window)
						} else {
							panel.RefreshFiles()
						}
					}
				},
				panel.window,
			)
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			if panel.selectedItem < 0 || panel.selectedItem >= len(panel.currentFiles) {
				return
			}
			file := panel.currentFiles[panel.selectedItem]
			dialog.ShowConfirm("删除确认",
				fmt.Sprintf("确定要删除 %s 吗？", file.Name),
				func(confirm bool) {
					if confirm {
						if err := panel.fileSystem.DeleteFile(file.Path); err != nil {
							dialog.ShowError(err, panel.window)
						} else {
							panel.selectedItem = -1
							panel.RefreshFiles()
						}
					}
				},
				panel.window,
			)
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ContentCopyIcon(), func() {
			if panel.selectedItem < 0 || panel.selectedItem >= len(panel.currentFiles) {
				return
			}
			if panel.onTransfer != nil {
				file := panel.currentFiles[panel.selectedItem]
				panel.onTransfer(file.Path, panel)
			}
		}),
	)

	// 组合所有元素
	panel.container = container.NewBorder(
		container.NewVBox(panel.pathEntry, toolbar, panel.progressBar),
		nil, nil, nil,
		panel.list,
	)

	// 初始加载文件列表
	panel.RefreshFiles()

	return panel
}

// GetContainer 返回面板的容器
func (p *FilePanel) GetContainer() fyne.CanvasObject {
	return p.container
}

// SetPath 设置当前路径
func (p *FilePanel) SetPath(path string) {
	p.fileSystem.SetCurrentPath(path)
	p.pathEntry.SetText(path)
	p.RefreshFiles()
}

// GetCurrentPath 获取当前路径
func (p *FilePanel) GetCurrentPath() string {
	return p.fileSystem.GetCurrentPath()
}

// RefreshFiles 刷新文件列表
func (p *FilePanel) RefreshFiles() {
	files, err := p.fileSystem.ListFiles(p.fileSystem.GetCurrentPath())
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}
	p.currentFiles = files
	p.list.Refresh()
}

// SetTransferCallback 设置传输回调
func (p *FilePanel) SetTransferCallback(callback func(source string, targetPanel *FilePanel)) {
	p.onTransfer = callback
}

// HandleTransfer 处理文件传输
func (p *FilePanel) HandleTransfer(sourcePath string) error {
	return p.transferMgr.Transfer(sourcePath, p.GetCurrentPath(), transfer.Copy)
}

// formatSize 格式化文件大小显示
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
