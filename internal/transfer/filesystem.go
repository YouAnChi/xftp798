package transfer

import (
	"fmt"
	"os"
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

// RemoteFS 远程文件系统接口
type RemoteFS interface {
	ListFiles(path string) ([]FileInfo, error)
	CreateDirectory(path string) error
	DeleteFile(path string) error
	UploadFile(localPath, remotePath string, progress func(current, total int64)) error
	DownloadFile(remotePath, localPath string, progress func(current, total int64)) error
	Close() error // 修改Close方法签名
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

// SetCurrentPath 设置当前路径
func (fs *FileSystem) SetCurrentPath(path string) {
	fs.currentPath = path
}

// ListFiles 列出目录下的文件
func (fs *FileSystem) ListFiles(path string) ([]FileInfo, error) {
	// 如果有远程文件系统，使用远程文件系统
	if fs.remote != nil {
		return fs.remote.ListFiles(path)
	}

	// 否则使用本地文件系统
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %v", err)
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			IsDir:   entry.IsDir(),
		})
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
