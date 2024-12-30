package transfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// TransferType 定义传输类型
type TransferType int

const (
	Copy TransferType = iota
	Move
)

// TransferProgress 传输进度信息
type TransferProgress struct {
	TotalSize      int64   // 总大小
	TransferredSize int64   // 已传输大小
	Percentage     float64 // 完成百分比
	CurrentFile    string  // 当前传输的文件
	IsCompleted    bool    // 是否完成
	Error          error   // 传输错误
	TransferType   TransferType // 传输类型
}

// TransferManager 文件传输管理器
type TransferManager struct {
	onProgress func(TransferProgress) // 进度回调函数
}

// NewTransferManager 创建新的传输管理器
func NewTransferManager(progressCallback func(TransferProgress)) *TransferManager {
	return &TransferManager{
		onProgress: progressCallback,
	}
}

// Transfer 传输文件或目录
func (tm *TransferManager) Transfer(src, dst string, transferType TransferType) error {
	// 获取源文件信息
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("无法获取源文件信息: %v", err)
	}

	// 创建目标路径
	dstPath := filepath.Join(dst, filepath.Base(src))

	// 如果是移动操作，先尝试直接重命名
	if transferType == Move {
		if err := os.Rename(src, dstPath); err == nil {
			// 重命名成功，直接返回
			if tm.onProgress != nil {
				tm.onProgress(TransferProgress{
					CurrentFile:  filepath.Base(src),
					IsCompleted: true,
					TransferType: transferType,
				})
			}
			return nil
		}
		// 如果重命名失败（可能跨设备），继续使用复制+删除的方式
	}

	// 如果是目录，递归复制
	if srcInfo.IsDir() {
		return tm.transferDir(src, dstPath, transferType)
	}

	// 如果是文件，直接复制
	return tm.transferFile(src, dstPath, transferType)
}

// transferDir 递归传输目录
func (tm *TransferManager) transferDir(src, dst string, transferType TransferType) error {
	// 创建目标目录
	if err := os.MkdirAll(dst, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %v", err)
	}

	// 读取源目录内容
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("读取源目录失败: %v", err)
	}

	// 递归复制每个文件和子目录
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := tm.transferDir(srcPath, dstPath, transferType); err != nil {
				return err
			}
		} else {
			if err := tm.transferFile(srcPath, dstPath, transferType); err != nil {
				return err
			}
		}
	}

	// 如果是移动操作，删除源目录
	if transferType == Move {
		if err := os.RemoveAll(src); err != nil {
			return fmt.Errorf("删除源目录失败: %v", err)
		}
	}

	return nil
}

// transferFile 传输单个文件
func (tm *TransferManager) transferFile(src, dst string, transferType TransferType) error {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer srcFile.Close()

	// 获取文件信息
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %v", err)
	}

	// 创建目标文件
	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer dstFile.Close()

	// 创建进度读取器
	progressReader := &ProgressReader{
		reader: srcFile,
		size:   srcInfo.Size(),
		onProgress: func(transferred int64) {
			if tm.onProgress != nil {
				percentage := float64(transferred) / float64(srcInfo.Size()) * 100
				tm.onProgress(TransferProgress{
					TotalSize:       srcInfo.Size(),
					TransferredSize: transferred,
					Percentage:      percentage,
					CurrentFile:     filepath.Base(src),
					IsCompleted:     transferred == srcInfo.Size(),
					TransferType:    transferType,
				})
			}
		},
	}

	// 复制文件内容
	if _, err := io.Copy(dstFile, progressReader); err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}

	// 如果是移动操作，删除源文件
	if transferType == Move {
		if err := os.Remove(src); err != nil {
			return fmt.Errorf("删除源文件失败: %v", err)
		}
	}

	return nil
}

// ProgressReader 用于跟踪复制进度的读取器
type ProgressReader struct {
	reader      io.Reader
	size        int64
	transferred int64
	onProgress  func(int64)
}

// Read 实现io.Reader接口
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.transferred += int64(n)
		if pr.onProgress != nil {
			pr.onProgress(pr.transferred)
		}
	}
	return n, err
}
