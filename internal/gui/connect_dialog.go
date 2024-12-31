package gui

import (
	"fmt"
	"strconv"
	"xftp798/internal/transfer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ConnectDialog 表示连接对话框
type ConnectDialog struct {
	window    fyne.Window
	onConnect func(*transfer.SFTPConfig)
}

// NewConnectDialog 创建新的连接对话框
func NewConnectDialog(window fyne.Window, onConnect func(*transfer.SFTPConfig)) *ConnectDialog {
	return &ConnectDialog{
		window:    window,
		onConnect: onConnect,
	}
}

// Show 显示连接对话框
func (d *ConnectDialog) Show() {
	// 创建输入框，设置更大的尺寸
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("请输入服务器地址")
	hostEntry.Resize(fyne.NewSize(300, 40))

	portEntry := widget.NewEntry()
	portEntry.SetText("22")
	portEntry.Resize(fyne.NewSize(300, 40))

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("请输入用户名")
	usernameEntry.Resize(fyne.NewSize(300, 40))

	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("请输入密码")
	passwordEntry.Resize(fyne.NewSize(300, 40))

	// 创建表单项
	items := []*widget.FormItem{
		widget.NewFormItem("服务器", hostEntry),
		widget.NewFormItem("端口", portEntry),
		widget.NewFormItem("用户名", usernameEntry),
		widget.NewFormItem("密码", passwordEntry),
	}

	// 创建对话框
	formDialog := dialog.NewForm(
		"连接到服务器",
		"连接",
		"取消",
		items,
		func(confirm bool) {
			if !confirm {
				return
			}

			// 验证输入
			if hostEntry.Text == "" || portEntry.Text == "" || usernameEntry.Text == "" || passwordEntry.Text == "" {
				dialog.ShowError(
					fmt.Errorf("所有字段都必须填写"),
					d.window,
				)
				return
			}

			// 验证端口号
			port, err := strconv.Atoi(portEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("端口号必须是数字"), d.window)
				return
			}

			// 创建配置
			config := &transfer.SFTPConfig{
				Host:     hostEntry.Text,
				Port:     port,
				Username: usernameEntry.Text,
				Password: passwordEntry.Text,
			}

			// 调用回调
			d.onConnect(config)

			// 清空密码
			passwordEntry.SetText("")
		},
		d.window,
	)

	// 设置对话框大小
	formDialog.Resize(fyne.NewSize(400, 300))
	formDialog.Show()
}
