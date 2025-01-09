package gui

import (
	"fmt"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"xftp798/internal/transfer"
)

// FilePanel 表示文件面板
type FilePanel struct {
	widget.BaseWidget // 继承基础组件
	container         *fyne.Container
	toolbar           *widget.Toolbar
	pathEntry         *widget.Entry
	fileTable         *widget.Table
	statusBar         *fyne.Container

	// 文件列表相关
	columns    []string
	sortColumn int
	sortAsc    bool
	headers    []*widget.Button

	fileSystem   *transfer.FileSystem
	currentFiles []transfer.FileInfo
	window       fyne.Window
	selectedItem int

	// 状态栏组件
	statusLabel  *widget.Label
	progressBar  *widget.ProgressBar
	progressInfo *widget.Label
	speedLabel   *widget.Label
	timeLabel    *widget.Label

	// 传输控制
	pauseButton      *widget.Button
	cancelButton     *widget.Button
	transferMgr      *transfer.TransferManager
	onTransfer       func(source string, targetPanel *FilePanel, transferType transfer.TransferType)
	remoteFS         transfer.RemoteFS
	isTransferPaused bool

	// 连接状态回调
	OnConnected func(bool)

	// 是否为本地面板
	isLocal bool

	// 记录上次点击时间，用于处理双击事件
	lastClick time.Time
}

// TableContainer 自定义容器
type TableContainer struct {
	widget.BaseWidget
	table *widget.Table
	panel *FilePanel
}

// 实现Tappable接口
var _ fyne.Tappable = (*TableContainer)(nil)
var _ fyne.SecondaryTappable = (*TableContainer)(nil)

// NewTableContainer 创建新的自定义容器
func NewTableContainer(table *widget.Table, panel *FilePanel) *TableContainer {
	tc := &TableContainer{
		table: table,
		panel: panel,
	}
	tc.ExtendBaseWidget(tc)
	fmt.Println("创建新的TableContainer") // 添加调试日志
	return tc
}

// CreateRenderer 实现自定义渲染
func (tc *TableContainer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(tc.table)
}

// Tapped 处理点击事件
func (tc *TableContainer) Tapped(e *fyne.PointEvent) {
	fmt.Println("鼠标左键点击事件触发") // 添加调试日志
	// 计算点击位置对应的行
	y := e.Position.Y - tc.table.Position().Y
	row := int(y / 30) // 假设每行高度为30

	if row >= 0 && row < len(tc.panel.currentFiles) {
		tc.panel.selectedItem = row
		tc.table.Select(widget.TableCellID{Row: row, Col: 0})

		// 处理双击
		now := time.Now()
		if row == tc.panel.selectedItem && now.Sub(tc.panel.lastClick) < 500*time.Millisecond {
			file := tc.panel.currentFiles[row]
			if file.IsDir {
				tc.panel.SetPath(file.Path)
			} else {
				if err := tc.panel.fileSystem.OpenFile(file.Path); err != nil {
					dialog.ShowError(err, tc.panel.window)
				}
			}
		}
		tc.panel.lastClick = now
	}
}

// TappedSecondary 处理右键点击事件
func (tc *TableContainer) TappedSecondary(e *fyne.PointEvent) {
	fmt.Println("鼠标右键点击事件触发") // 添加调试日志
	// 计算点击位置对应的行
	y := e.Position.Y - tc.table.Position().Y
	row := int(y / 30) // 假设每行高度为30

	if row >= 0 && row < len(tc.panel.currentFiles) {
		tc.panel.selectedItem = row
		tc.table.Select(widget.TableCellID{Row: row, Col: 0})

		// 显示右键菜单
		file := tc.panel.currentFiles[row]
		items := []*fyne.MenuItem{
			fyne.NewMenuItem("打开", func() {
				if file.IsDir {
					tc.panel.SetPath(file.Path)
				} else {
					if err := tc.panel.fileSystem.OpenFile(file.Path); err != nil {
						dialog.ShowError(err, tc.panel.window)
					}
				}
			}),
			fyne.NewMenuItem("重命名", func() {
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
							if err := tc.panel.fileSystem.Rename(oldPath, newPath); err != nil {
								dialog.ShowError(err, tc.panel.window)
							} else {
								tc.panel.RefreshFiles()
							}
						}
					},
					tc.panel.window,
				)
			}),
			fyne.NewMenuItem("删除", func() {
				dialog.ShowConfirm("删除确认",
					fmt.Sprintf("确定要删除 %s 吗？", file.Name),
					func(confirm bool) {
						if confirm {
							if err := tc.panel.fileSystem.Delete(file.Path); err != nil {
								dialog.ShowError(err, tc.panel.window)
							} else {
								tc.panel.RefreshFiles()
							}
						}
					},
					tc.panel.window,
				)
			}),
			fyne.NewMenuItem("属性", func() {
				content := widget.NewForm(
					widget.NewFormItem("名称", widget.NewLabel(file.Name)),
					widget.NewFormItem("路径", widget.NewLabel(file.Path)),
					widget.NewFormItem("大小", widget.NewLabel(formatFileSize(file.Size))),
					widget.NewFormItem("修改时间", widget.NewLabel(file.ModTime.Format("2006-01-02 15:04:05"))),
					widget.NewFormItem("类型", widget.NewLabel(func() string {
						if file.IsDir {
							return "文件夹"
						}
						return "文件"
					}())),
				)
				dialog.ShowCustom("文件属性", "关闭", content, tc.panel.window)
			}),
		}

		// 如果是本地面板，添加上传选项
		if tc.panel.isLocal && tc.panel.remoteFS != nil {
			items = append(items, fyne.NewMenuItem("上传到远程", func() {
				if tc.panel.onTransfer != nil {
					tc.panel.onTransfer(file.Path, tc.panel, transfer.Upload)
				}
			}))
		} else if !tc.panel.isLocal {
			// 如果是远程面板，添加下载选项
			items = append(items, fyne.NewMenuItem("下载到本地", func() {
				if tc.panel.onTransfer != nil {
					tc.panel.onTransfer(file.Path, tc.panel, transfer.Download)
				}
			}))
		}

		menu := fyne.NewMenu("", items...)
		widget.ShowPopUpMenuAtPosition(menu, tc.panel.window.Canvas(), e.AbsolutePosition)
	}
}

// CreateRenderer 实现自定义渲染
func (p *FilePanel) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.container)
}

// Tapped 处理点击事件
func (p *FilePanel) Tapped(e *fyne.PointEvent) {
	// 获取点击位置对应的表格行
	row := p.getRowFromPosition(e.Position)
	if row >= 0 && row < len(p.currentFiles) {
		p.handleClick(row)
	}
}

// TappedSecondary 处理右键点击事件
func (p *FilePanel) TappedSecondary(e *fyne.PointEvent) {
	// 获取点击位置对应的表格行
	row := p.getRowFromPosition(e.Position)
	if row >= 0 && row < len(p.currentFiles) {
		p.selectedItem = row
		p.showContextMenu(e)
	}
}

// getRowFromPosition 根据点击位置计算表格行
func (p *FilePanel) getRowFromPosition(pos fyne.Position) int {
	tablePos := p.fileTable.Position()
	relativeY := pos.Y - tablePos.Y
	rowHeight := float32(30) // 假设每行高度为30
	row := int(relativeY / rowHeight)
	return row
}

// handleClick 处理点击事件
func (p *FilePanel) handleClick(row int) {
	if row < 0 || row >= len(p.currentFiles) {
		return
	}

	// 更新选中项
	p.selectedItem = row
	p.fileTable.Select(widget.TableCellID{Row: row, Col: 0})

	// 处理双击
	now := time.Now()
	if row == p.selectedItem && now.Sub(p.lastClick) < 500*time.Millisecond {
		file := p.currentFiles[row]
		if file.IsDir {
			p.SetPath(file.Path)
		} else {
			if err := p.fileSystem.OpenFile(file.Path); err != nil {
				dialog.ShowError(err, p.window)
			}
		}
	}
	p.lastClick = now
}

// showContextMenu 显示右键菜单
func (p *FilePanel) showContextMenu(e *fyne.PointEvent) {
	if p.selectedItem < 0 || p.selectedItem >= len(p.currentFiles) {
		return
	}

	file := p.currentFiles[p.selectedItem]
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("打开", func() {
			if file.IsDir {
				p.SetPath(file.Path)
			} else {
				if err := p.fileSystem.OpenFile(file.Path); err != nil {
					dialog.ShowError(err, p.window)
				}
			}
		}),
		fyne.NewMenuItem("重命名", func() {
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
						if err := p.fileSystem.Rename(oldPath, newPath); err != nil {
							dialog.ShowError(err, p.window)
						} else {
							p.RefreshFiles()
						}
					}
				},
				p.window,
			)
		}),
		fyne.NewMenuItem("删除", func() {
			dialog.ShowConfirm("删除确认",
				fmt.Sprintf("确定要删除 %s 吗？", file.Name),
				func(confirm bool) {
					if confirm {
						if err := p.fileSystem.Delete(file.Path); err != nil {
							dialog.ShowError(err, p.window)
						} else {
							p.RefreshFiles()
						}
					}
				},
				p.window,
			)
		}),
		fyne.NewMenuItem("属性", func() {
			content := widget.NewForm(
				widget.NewFormItem("名称", widget.NewLabel(file.Name)),
				widget.NewFormItem("路径", widget.NewLabel(file.Path)),
				widget.NewFormItem("大小", widget.NewLabel(formatFileSize(file.Size))),
				widget.NewFormItem("修改时间", widget.NewLabel(file.ModTime.Format("2006-01-02 15:04:05"))),
				widget.NewFormItem("类型", widget.NewLabel(func() string {
					if file.IsDir {
						return "文件夹"
					}
					return "文件"
				}())),
			)
			dialog.ShowCustom("文件属性", "关闭", content, p.window)
		}),
	}

	// 如果是本地面板，添加上传选项
	if p.isLocal && p.remoteFS != nil {
		items = append(items, fyne.NewMenuItem("上传到远程", func() {
			if p.onTransfer != nil {
				p.onTransfer(file.Path, p, transfer.Upload)
			}
		}))
	} else if !p.isLocal {
		// 如果是远程面板，添加下载选项
		items = append(items, fyne.NewMenuItem("下载到本地", func() {
			if p.onTransfer != nil {
				p.onTransfer(file.Path, p, transfer.Download)
			}
		}))
	}

	menu := fyne.NewMenu("", items...)
	widget.ShowPopUpMenuAtPosition(menu, p.window.Canvas(), e.AbsolutePosition)
}

// NewFilePanel 创建新的文件面板
func NewFilePanel(window fyne.Window, isLocal bool) *FilePanel {
	panel := &FilePanel{
		window:       window,
		columns:      []string{"名称", "大小", "修改时间"},
		sortColumn:   0,
		sortAsc:      true,
		fileSystem:   transfer.NewFileSystem(""),
		selectedItem: -1,
		isLocal:      isLocal,
	}
	panel.ExtendBaseWidget(panel) // 扩展基础组件

	// 创建工具栏
	panel.toolbar = widget.NewToolbar(
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

	// 创建地址栏
	panel.pathEntry = widget.NewEntry()
	panel.pathEntry.OnSubmitted = func(path string) {
		panel.SetPath(path)
	}

	// 创建文件表格
	panel.fileTable = widget.NewTable(
		func() (int, int) {
			return len(panel.currentFiles), len(panel.columns)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Template")
			label.ExtendBaseWidget(label)
			return label
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			label := o.(*widget.Label)
			if i.Row < len(panel.currentFiles) {
				file := panel.currentFiles[i.Row]
				switch i.Col {
				case 0:
					if file.IsDir {
						label.SetText("📁 " + file.Name)
					} else {
						label.SetText("📄 " + file.Name)
					}
				case 1:
					if file.IsDir {
						label.SetText("<DIR>")
					} else {
						label.SetText(formatFileSize(file.Size))
					}
				case 2:
					label.SetText(file.ModTime.Format("2006-01-02 15:04:05"))
				}
			}
		})

	// 调试信息：检查fileTable是否创建成功
	fmt.Println("fileTable 已创建:", panel.fileTable != nil)

	// 设置列宽
	panel.fileTable.SetColumnWidth(0, 300)
	panel.fileTable.SetColumnWidth(1, 100)
	panel.fileTable.SetColumnWidth(2, 200)

	// 设置选中事件处理
	panel.fileTable.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(panel.currentFiles) {
			return
		}
		panel.selectedItem = id.Row
	}

	// 调试信息：检查fileTable是否创建成功
	fmt.Printf("fileTable 已创建: %v\n", panel.fileTable != nil)

	// 创建自定义容器并确保事件绑定
	tableContainer := NewTableContainer(panel.fileTable, panel)
	tableContainer.ExtendBaseWidget(tableContainer)

	// 设置选中事件处理
	panel.fileTable.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(panel.currentFiles) {
			return
		}
		panel.selectedItem = id.Row

		// 处理左键点击
		file := panel.currentFiles[id.Row]
		if file.IsDir {
			panel.SetPath(file.Path)
		} else {
			if err := panel.fileSystem.OpenFile(file.Path); err != nil {
				dialog.ShowError(err, panel.window)
			}
		}
	}

	// 创建右键菜单
	panel.fileTable.OnUnselected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(panel.currentFiles) {
			return
		}

		// 显示右键菜单
		file := panel.currentFiles[id.Row]
		menu := fyne.NewMenu("",
			fyne.NewMenuItem("重命名", func() {
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
							if err := panel.fileSystem.Rename(oldPath, newPath); err != nil {
								dialog.ShowError(err, panel.window)
							} else {
								panel.RefreshFiles()
							}
						}
					},
					panel.window,
				)
			}),
			fyne.NewMenuItem("删除", func() {
				dialog.ShowConfirm("删除确认",
					fmt.Sprintf("确定要删除 %s 吗？", file.Name),
					func(confirm bool) {
						if confirm {
							if err := panel.fileSystem.Delete(file.Path); err != nil {
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
		widget.ShowPopUpMenuAtPosition(menu, panel.window.Canvas(), fyne.CurrentApp().Driver().AbsolutePositionForObject(panel.fileTable))
	}

	// 创建滚动容器
	scrollContainer := container.NewScroll(container.NewMax(tableContainer))
	scrollContainer.SetMinSize(fyne.NewSize(800, 400))

	// 初始化状态栏组件
	panel.statusLabel = widget.NewLabel("")
	panel.progressBar = widget.NewProgressBar()
	panel.progressInfo = widget.NewLabel("")
	panel.speedLabel = widget.NewLabel("")
	panel.timeLabel = widget.NewLabel("")

	panel.statusBar = container.NewHBox(
		panel.statusLabel,
		panel.progressBar,
		panel.progressInfo,
		panel.speedLabel,
		panel.timeLabel,
	)

	panel.container = container.NewBorder(
		container.NewVBox(
			panel.toolbar,
			panel.pathEntry,
		),
		panel.statusBar,
		nil,
		nil,
		scrollContainer,
	)

	panel.RefreshFiles()
	return panel
}

// RefreshFiles 刷新文件列表
func (p *FilePanel) RefreshFiles() {
	// 获取当前路径
	currentPath := p.fileSystem.GetCurrentPath()
	if currentPath == "" {
		currentPath = "/"
	}

	// 更新路径输入框
	p.pathEntry.SetText(currentPath)

	// 读取文件列表
	files, err := p.fileSystem.ListFiles(currentPath)
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}

	p.currentFiles = files
	p.fileTable.Refresh()
}

// SetPath 设置当前路径
func (p *FilePanel) SetPath(path string) {
	p.fileSystem.SetPath(path)
	p.RefreshFiles()
}

// SetTransferCallback 设置传输回调函数
func (p *FilePanel) SetTransferCallback(callback func(source string, targetPanel *FilePanel, transferType transfer.TransferType)) {
	p.onTransfer = callback
}

// HandleTransfer 处理文件传输
func (p *FilePanel) HandleTransfer(source string, targetPanel *FilePanel, transferType transfer.TransferType) {
	if p.onTransfer != nil {
		p.onTransfer(source, targetPanel, transferType)
	}
}

// GetContainer 获取容器
func (p *FilePanel) GetContainer() *fyne.Container {
	return p.container
}

// SetRemoteFS 设置远程文件系统
func (p *FilePanel) SetRemoteFS(fs transfer.RemoteFS) {
	p.remoteFS = fs
}

// formatFileSize 格式化文件大小
func formatFileSize(size int64) string {
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
