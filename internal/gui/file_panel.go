package gui

import (
	"fmt"
	"os"
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
	onTransfer   func(source string, targetPanel *FilePanel, transferType transfer.TransferType)
	remoteFS     transfer.RemoteFS
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
			return container.NewHBox(
				widget.NewIcon(theme.FileIcon()),
				widget.NewLabel("文件名"),
				widget.NewLabel("大小"),
				widget.NewLabel("修改时间"),
			)
		},
		UpdateItem: func(id widget.ListItemID, item fyne.CanvasObject) {
			if id >= len(panel.currentFiles) {
				return
			}
			file := panel.currentFiles[id]
			box := item.(*fyne.Container)

			// 更新图标
			icon := box.Objects[0].(*widget.Icon)
			if file.IsDir {
				icon.SetResource(theme.FolderIcon())
			} else {
				icon.SetResource(theme.FileIcon())
			}

			// 更新文件信息
			nameLabel := box.Objects[1].(*widget.Label)
			sizeLabel := box.Objects[2].(*widget.Label)
			timeLabel := box.Objects[3].(*widget.Label)

			nameLabel.SetText(file.Name)
			sizeLabel.SetText(formatSize(file.Size))
			timeLabel.SetText(file.ModTime.Format("2006-01-02 15:04:05"))
		},
		OnSelected: func(id widget.ListItemID) {
			panel.selectedItem = int(id)
			if id >= len(panel.currentFiles) {
				return
			}
			file := panel.currentFiles[id]
			if file.IsDir {
				panel.SetPath(file.Path)
				panel.selectedItem = -1
				panel.list.UnselectAll()
			}
		},
		OnUnselected: func(id widget.ListItemID) {
			panel.selectedItem = -1
		},
	}

	// 创建工具栏
	toolbar := widget.NewToolbar(
		// 添加连接按钮
		widget.NewToolbarAction(theme.ComputerIcon(), func() {
			dialog := NewConnectDialog(panel.window, func(config *transfer.SFTPConfig) {
				// 创建SFTP文件系统
				remoteFS := transfer.NewSFTPFileSystem(config)
				
				// 连接服务器
				if err := remoteFS.Connect(); err != nil {
					dialog.ShowError(err, panel.window)
					return
				}

				// 保存远程文件系统
				if panel.remoteFS != nil {
					panel.remoteFS.Close()
				}
				panel.remoteFS = remoteFS
				panel.fileSystem.SetRemoteFS(remoteFS)

				// 切换到远程根目录
				panel.SetPath("/")
			})
			dialog.Show()
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
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			if panel.selectedItem < 0 || panel.selectedItem >= len(panel.currentFiles) {
				return
			}
			file := panel.currentFiles[panel.selectedItem]
			entry := widget.NewEntry()
			entry.SetText(file.Name)
			dialog.ShowForm("重命名", "确定", "取消",
				[]*widget.FormItem{
					widget.NewFormItem("新名称", entry),
				},
				func(confirm bool) {
					if confirm {
						oldPath := file.Path
						newPath := filepath.Join(filepath.Dir(file.Path), entry.Text)
						if err := os.Rename(oldPath, newPath); err != nil {
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
				panel.onTransfer(file.Path, panel, transfer.Copy)
			}
		}),
		widget.NewToolbarAction(theme.ContentCutIcon(), func() {
			if panel.selectedItem < 0 || panel.selectedItem >= len(panel.currentFiles) {
				return
			}
			if panel.onTransfer != nil {
				file := panel.currentFiles[panel.selectedItem]
				panel.onTransfer(file.Path, panel, transfer.Move)
			}
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
func (p *FilePanel) SetTransferCallback(callback func(source string, targetPanel *FilePanel, transferType transfer.TransferType)) {
	p.onTransfer = callback
}

// HandleTransfer 处理文件传输
func (p *FilePanel) HandleTransfer(sourcePath string, transferType transfer.TransferType) error {
	// 获取目标路径
	targetPath := filepath.Join(p.GetCurrentPath(), filepath.Base(sourcePath))

	// 根据传输类型处理
	switch transferType {
	case transfer.Copy:
		if p.remoteFS != nil {
			// 如果目标是远程的，执行上传
			return p.remoteFS.UploadFile(sourcePath, targetPath, func(current, total int64) {
				p.progressBar.Value = float64(current) / float64(total)
				p.progressBar.Show()
				if current == total {
					p.progressBar.Hide()
					p.RefreshFiles()
				}
			})
		} else {
			// 如果源是远程的，执行下载
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
		// 移动文件（暂不实现）
		return fmt.Errorf("暂不支持移动文件")
	default:
		return fmt.Errorf("未知的传输类型")
	}
}

// getSourcePanel 获取源面板
func (p *FilePanel) getSourcePanel() *FilePanel {
	// 这里需要实现获取源面板的逻辑
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
