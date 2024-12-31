package main

import (
	"xftp798/internal/gui"
	"xftp798/internal/transfer"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	// 创建应用
	a := app.New()
	window := a.NewWindow("XFTP文件传输")

	// 创建左右文件面板
	leftPanel := gui.NewFilePanel(window)
	rightPanel := gui.NewFilePanel(window)

	// 设置传输回调
	leftPanel.SetTransferCallback(func(source string, targetPanel *gui.FilePanel, transferType transfer.TransferType) {
		if err := rightPanel.HandleTransfer(source, transferType); err != nil {
			dialog.ShowError(err, window)
		}
	})

	rightPanel.SetTransferCallback(func(source string, targetPanel *gui.FilePanel, transferType transfer.TransferType) {
		if err := leftPanel.HandleTransfer(source, transferType); err != nil {
			dialog.ShowError(err, window)
		}
	})

	// 创建分割面板
	split := container.NewHSplit(
		leftPanel.GetContainer(),
		rightPanel.GetContainer(),
	)
	split.SetOffset(0.5) // 设置分割线位置在中间

	// 设置窗口内容
	window.SetContent(split)
	window.Resize(fyne.NewSize(1024, 768))

	// 运行应用
	window.ShowAndRun()
}
