package main

import (
	"xftp798/internal/gui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
)

func main() {
	// 创建应用
	myApp := app.New()
	window := myApp.NewWindow("XFTP文件传输")

	// 创建左右文件面板
	localPanel := gui.NewFilePanel()
	remotePanel := gui.NewFilePanel()

	// 创建分割面板
	split := container.NewHSplit(
		localPanel.GetContainer(),
		remotePanel.GetContainer(),
	)
	split.SetOffset(0.5) // 设置分割线位置在中间

	// 设置窗口内容
	window.SetContent(split)
	window.Resize(fyne.NewSize(800, 600))

	// 运行应用
	window.ShowAndRun()
}
