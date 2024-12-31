package transfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPConfig SFTP连接配置
type SFTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

// SFTPFileSystem SFTP文件系统
type SFTPFileSystem struct {
	config     *SFTPConfig
	sshClient  *ssh.Client
	sftpClient *sftp.Client
}

// NewSFTPFileSystem 创建新的SFTP文件系统
func NewSFTPFileSystem(config *SFTPConfig) *SFTPFileSystem {
	return &SFTPFileSystem{
		config: config,
	}
}

// Connect 连接到SFTP服务器
func (fs *SFTPFileSystem) Connect() error {
	// 创建SSH配置
	sshConfig := &ssh.ClientConfig{
		User: fs.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(fs.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接SSH服务器
	addr := fmt.Sprintf("%s:%d", fs.config.Host, fs.config.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("连接SSH服务器失败: %v", err)
	}
	fs.sshClient = sshClient

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		sshClient.Close()
		return fmt.Errorf("创建SFTP客户端失败: %v", err)
	}
	fs.sftpClient = sftpClient

	return nil
}

// Close 关闭连接
func (fs *SFTPFileSystem) Close() {
	if fs.sftpClient != nil {
		fs.sftpClient.Close()
	}
	if fs.sshClient != nil {
		fs.sshClient.Close()
	}
}

// ListFiles 列出目录下的文件
func (fs *SFTPFileSystem) ListFiles(path string) ([]FileInfo, error) {
	entries, err := fs.sftpClient.ReadDir(path)
	if err != nil {
		return nil, err
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    filepath.Join(path, entry.Name()),
			Size:    entry.Size(),
			ModTime: entry.ModTime(),
			IsDir:   entry.IsDir(),
		})
	}
	return files, nil
}

// CreateDirectory 创建目录
func (fs *SFTPFileSystem) CreateDirectory(path string) error {
	return fs.sftpClient.MkdirAll(path)
}

// DeleteFile 删除文件
func (fs *SFTPFileSystem) DeleteFile(path string) error {
	return fs.sftpClient.Remove(path)
}

// UploadFile 上传文件
func (fs *SFTPFileSystem) UploadFile(localPath, remotePath string, progress func(current, total int64)) error {
	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打开本地文件失败: %v", err)
	}
	defer localFile.Close()

	// 获取文件信息
	fileInfo, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %v", err)
	}
	totalSize := fileInfo.Size()

	// 创建远程文件
	remoteFile, err := fs.sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("创建远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	// 创建进度读取器
	buf := make([]byte, 32*1024)
	var currentSize int64
	for {
		n, err := localFile.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取文件失败: %v", err)
		}

		if _, err := remoteFile.Write(buf[:n]); err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}

		currentSize += int64(n)
		if progress != nil {
			progress(currentSize, totalSize)
		}
	}

	return nil
}

// DownloadFile 下载文件
func (fs *SFTPFileSystem) DownloadFile(remotePath, localPath string, progress func(current, total int64)) error {
	// 打开远程文件
	remoteFile, err := fs.sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("打开远程文件失败: %v", err)
	}
	defer remoteFile.Close()

	// 获取文件信息
	fileInfo, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("获取文件信息失败: %v", err)
	}
	totalSize := fileInfo.Size()

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("创建本地文件失败: %v", err)
	}
	defer localFile.Close()

	// 创建进度读取器
	buf := make([]byte, 32*1024)
	var currentSize int64
	for {
		n, err := remoteFile.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取文件失败: %v", err)
		}

		if _, err := localFile.Write(buf[:n]); err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}

		currentSize += int64(n)
		if progress != nil {
			progress(currentSize, totalSize)
		}
	}

	return nil
}
