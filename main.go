package main

import (
	"fmt"
	"os"
	"strconv"
	"xftp798/internal/gui"
	"xftp798/internal/transfer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// 创建应用
	a := app.New()
	window := a.NewWindow("XFTP文件传输")

	// 创建左右文件面板
	localPanel := gui.NewFilePanel(window, true) // 本地面板
	// 设置本地面板初始路径为用户主目录
	if homeDir, err := os.UserHomeDir(); err == nil {
		localPanel.SetPath(homeDir)
	} else {
		localPanel.SetPath("/")
	}
	remotePanel := gui.NewFilePanel(window, false) // 远程面板

	// 创建进度条面板
	progressBar := widget.NewProgressBar()
	progressBar.Hide()
	progressInfo := widget.NewLabel("")
	progressInfo.Hide()
	speedLabel := widget.NewLabel("")
	speedLabel.Hide()
	timeLabel := widget.NewLabel("")
	timeLabel.Hide()

	progressPanel := container.NewHBox(
		progressBar,
		progressInfo,
		speedLabel,
		timeLabel,
	)

	// 设置传输回调
	localPanel.SetTransferCallback(func(source string, targetPanel *gui.FilePanel, transferType transfer.TransferType) {
		remotePanel.HandleTransfer(source, targetPanel, transferType)
	})

	remotePanel.SetTransferCallback(func(source string, targetPanel *gui.FilePanel, transferType transfer.TransferType) {
		localPanel.HandleTransfer(source, targetPanel, transferType)
	})

	// 创建右侧占位面板
	rightPlaceholder := widget.NewLabel("请点击上方的电脑图标连接到远程服务器")
	rightPlaceholder.Alignment = fyne.TextAlignCenter

	// 创建分割面板
	split := container.NewHSplit(
		localPanel.GetContainer(),
		rightPlaceholder,
	)
	split.SetOffset(0.5) // 设置分割线位置在中间

	// 创建顶部工具栏
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ComputerIcon(), func() {
			// 创建输入框
			hostEntry := widget.NewEntry()
			portEntry := widget.NewEntry()
			portEntry.SetText("22") // 默认端口
			userEntry := widget.NewEntry()
			passEntry := widget.NewPasswordEntry()

			dialog.ShowForm("连接服务器", "连接", "取消", []*widget.FormItem{
				widget.NewFormItem("主机", hostEntry),
				widget.NewFormItem("端口", portEntry),
				widget.NewFormItem("用户名", userEntry),
				widget.NewFormItem("密码", passEntry),
			}, func(confirm bool) {
				if confirm {
					// 转换端口为整数
					port := 22 // 默认端口
					if portEntry.Text != "" {
						if p, err := strconv.Atoi(portEntry.Text); err == nil {
							port = p
						} else {
							dialog.ShowError(fmt.Errorf("端口号必须是数字"), window)
							return
						}
					}

					// 创建SFTP配置
					config := &transfer.SFTPConfig{
						Host:     hostEntry.Text,
						Port:     port,
						Username: userEntry.Text,
						Password: passEntry.Text,
					}

					// 创建SFTP文件系统
					remoteFS := transfer.NewSFTPFileSystem(config)

					// 连接服务器
					if err := remoteFS.Connect(); err != nil {
						dialog.ShowError(err, window)
						if remotePanel.OnConnected != nil {
							remotePanel.OnConnected(false)
						}
						return
					}

					// 设置远程文件系统
					remotePanel.SetRemoteFS(remoteFS)

					// 切换到远程根目录
					remotePanel.SetPath("/")

					// 通知连接状态改变
					if remotePanel.OnConnected != nil {
						remotePanel.OnConnected(true)
					}
				}
			}, window)
		}),
	)

	// 监听远程面板的连接状态
	remotePanel.OnConnected = func(connected bool) {
		if connected {
			split.Trailing = remotePanel.GetContainer()
		} else {
			split.Trailing = rightPlaceholder
		}
		split.Refresh()
	}

	// 创建主界面布局
	mainContainer := container.NewBorder(
		toolbar,       // 顶部
		progressPanel, // 底部
		nil,           // 左侧
		nil,           // 右侧
		split,         // 中间
	)

	// 设置窗口内容
	window.SetContent(mainContainer)
	window.Resize(fyne.NewSize(800, 600))
	window.SetFixedSize(true)

	// 运行应用
	window.ShowAndRun()
}
