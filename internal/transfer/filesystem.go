package transfer

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

// FileInfo 存储文件信息
type FileInfo struct {
	Name         string    // 文件名
	Path         string    // 完整路径
	Size         int64     // 文件大小
	IsDir        bool      // 是否是目录
	ModTime      time.Time // 修改时间
	Permissions  string    // 权限
}

// FileSystem 提供文件系统操作接口
type FileSystem struct {
	currentPath string
}

// NewFileSystem 创建新的文件系统处理器
func NewFileSystem(initialPath string) *FileSystem {
	if initialPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			initialPath = homeDir
		} else {
			initialPath = "/"
		}
	}
	return &FileSystem{
		currentPath: initialPath,
	}
}

// ListFiles 列出指定目录下的所有文件和文件夹
func (fs *FileSystem) ListFiles(path string) ([]FileInfo, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fileInfos []FileInfo
	for _, file := range files {
		info := FileInfo{
			Name:    file.Name(),
			Path:    filepath.Join(path, file.Name()),
			Size:    file.Size(),
			IsDir:   file.IsDir(),
			ModTime: file.ModTime(),
		}
		
		// 获取文件权限
		if stat, err := os.Stat(info.Path); err == nil {
			info.Permissions = stat.Mode().String()
		}
		
		fileInfos = append(fileInfos, info)
	}

	return fileInfos, nil
}

// CreateDirectory 创建新目录
func (fs *FileSystem) CreateDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

// DeleteFile 删除文件或目录
func (fs *FileSystem) DeleteFile(path string) error {
	return os.RemoveAll(path)
}

// GetCurrentPath 获取当前路径
func (fs *FileSystem) GetCurrentPath() string {
	return fs.currentPath
}

// SetCurrentPath 设置当前路径
func (fs *FileSystem) SetCurrentPath(path string) {
	fs.currentPath = path
}
