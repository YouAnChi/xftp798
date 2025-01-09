# XFTP798 项目架构思维导图

```mermaid
mindmap
  root((XFTP798))
    CoreFeatures((核心功能))
      FileManagement((文件管理))
        Local((本地文件管理))
        Remote((远程文件管理))
      FileTransfer((文件传输))
        Upload((上传))
        Download((下载))
        Progress((进度管理))
      Connection((连接管理))
        SFTP((SFTP连接))
        SSH((SSH安全))
    TechStack((技术架构))
      GUI((图形界面))
        Fyne((Fyne框架))
      Network((网络通信))
        SFTP((SFTP协议))
        SSH((SSH协议))
      Language((编程语言))
        Go((Go语言))
    Modules((主要模块))
      GUI((界面模块))
        FilePanel((文件面板))
        ConnectDialog((连接对话框))
      Transfer((传输模块))
        FileSystem((文件系统接口))
        SFTPFileSystem((SFTP实现))
        TransferManager((传输管理))
    Dependencies((依赖关系))
      Fyne((Fyne GUI框架))
      SFTP((pkg/sftp库))
      SSH((crypto/ssh库))
    Roadmap((未来规划))
      Features((功能增强))
        Queue((传输队列))
        Resume((断点续传))
        SpeedControl((速度控制))
      UI((界面优化))
        Theme((主题定制))
        Language((多语言支持))
        Shortcut((快捷键))
```

## 使用说明

1. 该思维导图使用Mermaid语法编写
2. 可以在支持Mermaid的Markdown编辑器中查看
3. 主要包含以下部分：
   - 核心功能：项目的主要功能模块
   - 技术架构：使用的技术栈和框架
   - 主要模块：代码层面的模块划分
   - 依赖关系：项目依赖的第三方库
   - 未来规划：计划实现的功能和改进

## 更新记录

- 2024-01-01 创建初始版本
