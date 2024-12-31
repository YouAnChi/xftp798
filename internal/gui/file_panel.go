package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"xftp798/internal/transfer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"image/color"
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
	onTransfer   func(source string, targetPanel *FilePanel, transferType transfer.TransferType)
	remoteFS     transfer.RemoteFS
}

// FileListItem 自定义列表项
type FileListItem struct {
	widget.BaseWidget
	panel    *FilePanel
	file     transfer.FileInfo
	index    int
	icon     *widget.Icon
	name     *widget.Label
	size     *widget.Label
	modTime  *widget.Label
	selected bool
}

// FileListItemRenderer 自定义列表项渲染器
type FileListItemRenderer struct {
	item      *FileListItem
	container *fyne.Container
	bg        *canvas.Rectangle
}

// NewFileListItem 创建新的列表项
func NewFileListItem(panel *FilePanel, file transfer.FileInfo, index int) *FileListItem {
	item := &FileListItem{
		panel: panel,
		file:  file,
		index: index,
	}

	// 创建图标
	item.icon = widget.NewIcon(theme.FileIcon())
	if file.IsDir {
		item.icon.SetResource(theme.FolderIcon())
	}

	// 创建标签
	item.name = widget.NewLabel(file.Name)
	item.size = widget.NewLabel(formatSize(file.Size))
	item.modTime = widget.NewLabel(file.ModTime.Format("2006-01-02 15:04:05"))

	item.ExtendBaseWidget(item)
	return item
}

// CreateRenderer 创建渲染器
func (i *FileListItem) CreateRenderer() fyne.WidgetRenderer {
	// 创建背景，使用主色调并设置透明度
	primaryColor := theme.PrimaryColor()
	r, g, b, _ := primaryColor.RGBA()
	bgColor := &color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: 76, // 透明度 30%（76/255 ≈ 0.3）
	}
	bg := canvas.NewRectangle(bgColor)
	bg.Hide()

	// 创建内容容器，添加内边距
	content := container.NewHBox(i.icon, i.name, i.size, i.modTime)
	container := container.NewPadded(content)

	// 创建渲染器
	renderer := &FileListItemRenderer{
		item:      i,
		container: container,
		bg:        bg,
	}

	return renderer
}

// MinSize 实现渲染器接口
func (r *FileListItemRenderer) MinSize() fyne.Size {
	return r.container.MinSize()
}

// Layout 实现渲染器接口
func (r *FileListItemRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.container.Resize(size)
}

// Refresh 实现渲染器接口
func (r *FileListItemRenderer) Refresh() {
	if r.item.selected {
		r.bg.Show()
	} else {
		r.bg.Hide()
	}
	r.container.Refresh()
}

// Objects 实现渲染器接口
func (r *FileListItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.container}
}

// Destroy 实现渲染器接口
func (r *FileListItemRenderer) Destroy() {
}

// Tapped 处理点击事件
func (i *FileListItem) Tapped(_ *fyne.PointEvent) {
	i.panel.selectedItem = -1
	i.panel.list.Refresh()

	i.panel.selectedItem = i.index
	i.selected = true
	i.Refresh()

	if i.file.IsDir {
		i.panel.SetPath(i.file.Path)
		i.panel.selectedItem = -1
		i.selected = false
		i.Refresh()
	}
}

// TappedSecondary 处理右键点击事件
func (i *FileListItem) TappedSecondary(e *fyne.PointEvent) {
	i.panel.selectedItem = -1
	i.panel.list.Refresh()

	i.panel.selectedItem = i.index
	i.selected = true
	i.Refresh()

	i.panel.showContextMenu(i.file, e.AbsolutePosition)
}

// MinSize 返回最小尺寸
func (i *FileListItem) MinSize() fyne.Size {
	return fyne.NewSize(400, 40)
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
	panel.list = &widget.List{
		Length: func() int {
			return len(panel.currentFiles)
		},
		CreateItem: func() fyne.CanvasObject {
			return NewFileListItem(panel, transfer.FileInfo{}, 0)
		},
		UpdateItem: func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(panel.currentFiles) {
				return
			}
			fileItem := item.(*FileListItem)
			fileItem.file = panel.currentFiles[id]
			fileItem.index = int(id)
			fileItem.selected = int(id) == panel.selectedItem

			// 更新图标
			if fileItem.file.IsDir {
				fileItem.icon.SetResource(theme.FolderIcon())
			} else {
				fileItem.icon.SetResource(theme.FileIcon())
			}

			// 更新标签
			fileItem.name.SetText(fileItem.file.Name)
			fileItem.size.SetText(formatSize(fileItem.file.Size))
			fileItem.modTime.SetText(fileItem.file.ModTime.Format("2006-01-02 15:04:05"))
		},
		OnUnselected: func(id widget.ListItemID) {
			panel.selectedItem = -1
		},
	}

	// 创建工具栏
	toolbar := widget.NewToolbar(
		// 添加连接按钮
		widget.NewToolbarAction(theme.ComputerIcon(), func() {
			connectDialog := NewConnectDialog(panel.window, func(config *transfer.SFTPConfig) {
				// 创建SFTP文件系统
				remoteFS := transfer.NewSFTPFileSystem(config)

				// 连接服务器
				if err := remoteFS.Connect(); err != nil {
					dialog.ShowError(err, panel.window)
					return
				}

				// 保存远程文件系统
				if panel.remoteFS != nil {
					if err := panel.remoteFS.Close(); err != nil {
						dialog.ShowError(fmt.Errorf("关闭连接失败: %v", err), panel.window)
					}
				}
				panel.remoteFS = remoteFS
				panel.fileSystem.SetRemoteFS(remoteFS)

				// 切换到远程根目录
				panel.SetPath("/")
			})
			connectDialog.Show()
		}),
		widget.NewToolbarSeparator(),
		// 添加返回上一级按钮
		widget.NewToolbarAction(theme.NavigateBackIcon(), func() {
			currentPath := panel.fileSystem.GetCurrentPath()
			parentPath := filepath.Dir(currentPath)
			panel.SetPath(parentPath)
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			panel.RefreshFiles()
		}),
		widget.NewToolbarAction(theme.FolderNewIcon(), func() {
			entry := widget.NewEntry()
			entry.SetPlaceHolder("输入文件夹名称")
			dialog.ShowForm("新建文件夹",
				"创建",
				"取消",
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
	)

	// 组合所有元素
	panel.container = container.NewBorder(
		container.NewVBox(
			panel.pathEntry,
			toolbar,
			panel.progressBar,
		),
		nil, nil, nil,
		container.NewScroll(panel.list),
	)

	// 初始加载文件列表
	panel.RefreshFiles()

	return panel
}

// showContextMenu 显示右键菜单
func (p *FilePanel) showContextMenu(file transfer.FileInfo, pos fyne.Position) {
	// 创建菜单项
	var menuItems []*fyne.MenuItem

	// 打开/进入
	if file.IsDir {
		menuItems = append(menuItems, fyne.NewMenuItem("进入", func() {
			p.SetPath(file.Path)
		}))
	} else {
		menuItems = append(menuItems, fyne.NewMenuItem("打开", func() {
			// 这里可以添加打开文件的逻辑
			dialog.ShowInformation("提示", "此功能暂未实现", p.window)
		}))
	}

	// 复制
	menuItems = append(menuItems, fyne.NewMenuItem("复制", func() {
		if p.onTransfer != nil {
			p.onTransfer(file.Path, p, transfer.Copy)
		}
	}))

	// 剪切
	menuItems = append(menuItems, fyne.NewMenuItem("剪切", func() {
		if p.onTransfer != nil {
			p.onTransfer(file.Path, p, transfer.Move)
		}
	}))

	// 删除
	menuItems = append(menuItems, fyne.NewMenuItem("删除", func() {
		dialog.ShowConfirm("删除确认",
			fmt.Sprintf("确定要删除 %s 吗？", file.Name),
			func(confirm bool) {
				if confirm {
					if err := p.fileSystem.DeleteFile(file.Path); err != nil {
						dialog.ShowError(err, p.window)
					} else {
						p.selectedItem = -1
						p.RefreshFiles()
					}
				}
			},
			p.window,
		)
	}))

	// 重命名
	menuItems = append(menuItems, fyne.NewMenuItem("重命名", func() {
		entry := widget.NewEntry()
		entry.SetText(file.Name)
		dialog.ShowForm("重命名",
			"确定",
			"取消",
			[]*widget.FormItem{
				widget.NewFormItem("新名称", entry),
			},
			func(confirm bool) {
				if confirm {
					oldPath := file.Path
					newPath := filepath.Join(filepath.Dir(file.Path), entry.Text)
					if err := os.Rename(oldPath, newPath); err != nil {
						dialog.ShowError(err, p.window)
					} else {
						p.RefreshFiles()
					}
				}
			},
			p.window,
		)
	}))

	// 属性
	menuItems = append(menuItems, fyne.NewMenuItem("属性", func() {
		info := fmt.Sprintf(
			"名称：%s\n"+
				"类型：%s\n"+
				"大小：%s\n"+
				"修改时间：%s\n"+
				"路径：%s",
			file.Name,
			func() string {
				if file.IsDir {
					return "文件夹"
				}
				return "文件"
			}(),
			formatSize(file.Size),
			file.ModTime.Format("2006-01-02 15:04:05"),
			file.Path,
		)
		dialog.ShowInformation("文件属性", info, p.window)
	}))

	// 显示菜单
	menu := fyne.NewMenu("", menuItems...)
	widget.ShowPopUpMenuAtPosition(menu, p.window.Canvas(), pos)
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
	p.selectedItem = -1
	p.currentFiles = files
	p.list.Refresh()
}

// SetTransferCallback 设置传输回调
func (p *FilePanel) SetTransferCallback(callback func(source string, targetPanel *FilePanel, transferType transfer.TransferType)) {
	p.onTransfer = callback
}

// HandleTransfer 处理文件传输
func (p *FilePanel) HandleTransfer(sourcePath string, transferType transfer.TransferType) error {
	targetPath := filepath.Join(p.GetCurrentPath(), filepath.Base(sourcePath))

	switch transferType {
	case transfer.Copy:
		if p.remoteFS != nil {
			return p.remoteFS.UploadFile(sourcePath, targetPath, func(current, total int64) {
				p.progressBar.Value = float64(current) / float64(total)
				p.progressBar.Show()
				if current == total {
					p.progressBar.Hide()
					p.RefreshFiles()
				}
			})
		} else {
			sourcePanel := p.getSourcePanel()
			if sourcePanel != nil && sourcePanel.remoteFS != nil {
				return sourcePanel.remoteFS.DownloadFile(sourcePath, targetPath, func(current, total int64) {
					p.progressBar.Value = float64(current) / float64(total)
					p.progressBar.Show()
					if current == total {
						p.progressBar.Hide()
						p.RefreshFiles()
					}
				})
			}
			return fmt.Errorf("不支持本地文件传输")
		}
	case transfer.Move:
		return fmt.Errorf("暂不支持移动文件")
	default:
		return fmt.Errorf("未知的传输类型")
	}
}

// getSourcePanel 获取源面板
func (p *FilePanel) getSourcePanel() *FilePanel {
	return nil
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
