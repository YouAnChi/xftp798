package transfer

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// FileInfo 文件信息
type FileInfo struct {
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
	IsDir   bool
}

// FileSystem 文件系统接口
type FileSystem struct {
	currentPath string
	remote      RemoteFS
}

// OpenFile 打开文件
func (fs *FileSystem) OpenFile(path string) error {
	if fs.remote != nil {
		return fmt.Errorf("远程文件系统不支持直接打开文件")
	}

	// 在macOS上使用open命令打开文件
	cmd := exec.Command("open", path)
	return cmd.Run()
}

// Delete 删除文件或目录
func (fs *FileSystem) Delete(path string) error {
	return fs.DeleteFile(path)
}

// Rename 重命名文件或目录
func (fs *FileSystem) Rename(oldPath, newPath string) error {
	return fs.RenameFile(oldPath, newPath)
}

// Download 下载文件
func (fs *FileSystem) Download(remotePath string, writer io.Writer) error {
	if fs.remote == nil {
		return fmt.Errorf("本地文件系统不支持下载操作")
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "xftp798-download-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 下载到临时文件
	err = fs.remote.DownloadFile(remotePath, tmpFile.Name(), nil)
	if err != nil {
		return err
	}

	// 将临时文件内容复制到writer
	_, err = io.Copy(writer, tmpFile)
	return err
}

// RemoteFS 远程文件系统接口
type RemoteFS interface {
	ListFiles(path string) ([]FileInfo, error)
	CreateDirectory(path string) error
	DeleteFile(path string) error
	RenameFile(oldPath, newPath string) error
	UploadFile(localPath, remotePath string, progress func(current, total int64)) error
	DownloadFile(remotePath, localPath string, progress func(current, total int64)) error
	Close() error
}

// NewFileSystem 创建新的文件系统
func NewFileSystem(path string) *FileSystem {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			path = homeDir
		} else {
			path = "/"
		}
	}
	return &FileSystem{
		currentPath: path,
	}
}

// SetRemoteFS 设置远程文件系统
func (fs *FileSystem) SetRemoteFS(remote RemoteFS) {
	fs.remote = remote
}

// GetCurrentPath 获取当前路径
func (fs *FileSystem) GetCurrentPath() string {
	return fs.currentPath
}

// SetPath 设置当前路径
func (fs *FileSystem) SetPath(path string) error {
	// 检查路径是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("路径不存在: %s", path)
	}
	fs.currentPath = path
	return nil
}

// ListFiles 列出目录下的文件
func (fs *FileSystem) ListFiles(path string) ([]FileInfo, error) {
	// 如果有远程文件系统，使用远程文件系统
	if fs.remote != nil {
		return fs.remote.ListFiles(path)
	}

	// 确保路径存在
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("路径不存在: %s", path)
		}
		return nil, fmt.Errorf("无法访问路径: %s, 错误: %v", path, err)
	}

	// 确保是目录
	if !fileInfo.IsDir() {
		return nil, fmt.Errorf("路径不是目录: %s", path)
	}

	// 读取目录内容
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		// 获取完整的文件信息
		info, err := entry.Info()
		if err != nil {
			// 记录错误但继续处理其他文件
			log.Printf("获取文件信息失败 %s: %v\n", entry.Name(), err)
			continue
		}

		// 构建完整路径
		fullPath := filepath.Join(path, entry.Name())

		// 创建文件信息对象
		fileInfo := FileInfo{
			Name:    entry.Name(),
			Path:    fullPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   info.IsDir(),
		}

		// 打印调试信息
		log.Printf("文件信息: %+v\n", fileInfo)

		files = append(files, fileInfo)
	}

	// 如果没有找到任何文件
	if len(files) == 0 {
		log.Printf("目录为空或无法读取任何文件: %s\n", path)
	}

	return files, nil
}

// CreateDirectory 创建目录
func (fs *FileSystem) CreateDirectory(path string) error {
	if fs.remote != nil {
		return fs.remote.CreateDirectory(path)
	}
	return os.MkdirAll(path, 0755)
}

// DeleteFile 删除文件或目录
func (fs *FileSystem) DeleteFile(path string) error {
	if fs.remote != nil {
		return fs.remote.DeleteFile(path)
	}
	return os.Remove(path)
}

// RenameFile 重命名文件或目录
func (fs *FileSystem) RenameFile(oldPath, newPath string) error {
	if fs.remote != nil {
		return fs.remote.RenameFile(oldPath, newPath)
	}
	return os.Rename(oldPath, newPath)
}

// UploadFile 上传文件
func (fs *FileSystem) UploadFile(localPath, remotePath string, progress func(current, total int64)) error {
	if fs.remote != nil {
		return fs.remote.UploadFile(localPath, remotePath, progress)
	}
	return fmt.Errorf("本地文件系统不支持上传操作")
}

// DownloadFile 下载文件
func (fs *FileSystem) DownloadFile(remotePath, localPath string, progress func(current, total int64)) error {
	if fs.remote != nil {
		return fs.remote.DownloadFile(remotePath, localPath, progress)
	}
	return fmt.Errorf("本地文件系统不支持下载操作")
}

// Close 关闭文件系统
func (fs *FileSystem) Close() error {
	if fs.remote != nil {
		return fs.remote.Close()
	}
	return nil
}
