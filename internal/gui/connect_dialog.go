package gui

import (
	"fmt"
	"strconv"
	"xftp798/internal/transfer"

	"fyne.io/fyne/v2"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ConnectDialog 连接对话框
type ConnectDialog struct {
	window   fyne.Window
	onSubmit func(*transfer.SFTPConfig)
}

// NewConnectDialog 创建新的连接对话框
func NewConnectDialog(window fyne.Window, onSubmit func(*transfer.SFTPConfig)) *ConnectDialog {
	return &ConnectDialog{
		window:   window,
		onSubmit: onSubmit,
	}
}

// Show 显示对话框
func (d *ConnectDialog) Show() {
	// 创建输入框
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("服务器地址")

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("端口号")
	portEntry.SetText("22") // 默认SSH端口

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("用户名")

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("密码")

	// 创建表单项
	items := []*widget.FormItem{
		widget.NewFormItem("服务器", hostEntry),
		widget.NewFormItem("端口", portEntry),
		widget.NewFormItem("用户名", usernameEntry),
		widget.NewFormItem("密码", passwordEntry),
	}

	// 创建对话框
	dialog.ShowForm("连接到服务器", "连接", "取消", items,
		func(confirm bool) {
			if !confirm {
				return
			}

			// 验证输入
			if hostEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("请输入服务器地址"), d.window)
				return
			}
			if usernameEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("请输入用户名"), d.window)
				return
			}
			if passwordEntry.Text == "" {
				dialog.ShowError(fmt.Errorf("请输入密码"), d.window)
				return
			}

			// 解析端口号
			port, err := strconv.Atoi(portEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("端口号无效"), d.window)
				return
			}

			// 创建配置
			config := &transfer.SFTPConfig{
				Host:     hostEntry.Text,
				Port:     port,
				Username: usernameEntry.Text,
				Password: passwordEntry.Text,
			}

			// 回调
			if d.onSubmit != nil {
				d.onSubmit(config)
			}
		},
		d.window,
	)
}
