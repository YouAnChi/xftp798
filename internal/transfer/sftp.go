package transfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPConfig SFTP配置
type SFTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

// SFTPFileSystem SFTP文件系统实现
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
	config := &ssh.ClientConfig{
		User: fs.config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(fs.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// 连接到SSH服务器
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", fs.config.Host, fs.config.Port), config)
	if err != nil {
		return fmt.Errorf("连接SSH服务器失败: %v", err)
	}

	// 创建SFTP客户端
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		client.Close()
		return fmt.Errorf("创建SFTP客户端失败: %v", err)
	}

	fs.sshClient = client
	fs.sftpClient = sftpClient
	return nil
}

// Close 关闭连接
func (fs *SFTPFileSystem) Close() error {
	var err error
	if fs.sftpClient != nil {
		if e := fs.sftpClient.Close(); e != nil {
			err = e
		}
	}
	if fs.sshClient != nil {
		if e := fs.sshClient.Close(); e != nil && err == nil {
			err = e
		}
	}
	return err
}

// ListFiles 列出目录下的文件
func (fs *SFTPFileSystem) ListFiles(path string) ([]FileInfo, error) {
	files, err := fs.sftpClient.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var fileInfos []FileInfo
	for _, file := range files {
		fileInfos = append(fileInfos, FileInfo{
			Name:    file.Name(),
			Path:    filepath.Join(path, file.Name()),
			Size:    file.Size(),
			ModTime: file.ModTime(),
			IsDir:   file.IsDir(),
		})
	}
	return fileInfos, nil
}

// UploadFile 上传文件或目录
func (fs *SFTPFileSystem) UploadFile(localPath, remotePath string, progress func(current, total int64)) error {
	// 获取本地文件信息
	info, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	// 如果是目录，递归上传
	if info.IsDir() {
		return fs.uploadDirectory(localPath, remotePath, progress)
	}

	// 创建远程目录
	if err := fs.createRemoteDirectory(filepath.Dir(remotePath)); err != nil {
		return err
	}

	// 打开本地文件
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// 创建远程文件
	remoteFile, err := fs.sftpClient.Create(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	// 获取文件大小
	fileSize := info.Size()
	var uploaded int64

	// 创建带进度的读取器
	reader := &progressReader{
		reader: localFile,
		progress: func(n int64) {
			uploaded += n
			if progress != nil {
				progress(uploaded, fileSize)
			}
		},
	}

	// 复制文件内容
	_, err = io.Copy(remoteFile, reader)
	return err
}

// uploadDirectory 递归上传目录
func (fs *SFTPFileSystem) uploadDirectory(localPath, remotePath string, progress func(current, total int64)) error {
	// 创建远程目录
	if err := fs.createRemoteDirectory(remotePath); err != nil {
		return err
	}

	// 遍历本地目录
	return filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对路径
		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return err
		}

		// 构建远程路径
		remoteFilePath := filepath.Join(remotePath, relPath)

		// 如果是目录，创建远程目录
		if info.IsDir() {
			return fs.createRemoteDirectory(remoteFilePath)
		}

		// 上传文件
		return fs.UploadFile(path, remoteFilePath, progress)
	})
}

// DownloadFile 下载文件或目录
func (fs *SFTPFileSystem) DownloadFile(remotePath, localPath string, progress func(current, total int64)) error {
	// 获取远程文件信息
	info, err := fs.sftpClient.Stat(remotePath)
	if err != nil {
		return err
	}

	// 如果是目录，递归下载
	if info.IsDir() {
		return fs.downloadDirectory(remotePath, localPath, progress)
	}

	// 创建本地目录
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	// 打开远程文件
	remoteFile, err := fs.sftpClient.Open(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	// 创建本地文件
	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	// 获取文件大小
	fileSize := info.Size()
	var downloaded int64

	// 创建带进度的读取器
	reader := &progressReader{
		reader: remoteFile,
		progress: func(n int64) {
			downloaded += n
			if progress != nil {
				progress(downloaded, fileSize)
			}
		},
	}

	// 复制文件内容
	_, err = io.Copy(localFile, reader)
	return err
}

// downloadDirectory 递归下载目录
func (fs *SFTPFileSystem) downloadDirectory(remotePath, localPath string, progress func(current, total int64)) error {
	// 创建本地目录
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return err
	}

	// 列出远程目录内容
	files, err := fs.sftpClient.ReadDir(remotePath)
	if err != nil {
		return err
	}

	// 遍历并下载每个文件/目录
	for _, file := range files {
		remoteFilePath := filepath.Join(remotePath, file.Name())
		localFilePath := filepath.Join(localPath, file.Name())

		if file.IsDir() {
			// 递归下载子目录
			if err := fs.downloadDirectory(remoteFilePath, localFilePath, progress); err != nil {
				return err
			}
		} else {
			// 下载文件
			if err := fs.DownloadFile(remoteFilePath, localFilePath, progress); err != nil {
				return err
			}
		}
	}

	return nil
}

// createRemoteDirectory 创建远程目录
func (fs *SFTPFileSystem) createRemoteDirectory(path string) error {
	// 尝试创建目录
	err := fs.sftpClient.MkdirAll(path)
	if err != nil {
		return fmt.Errorf("创建远程目录失败: %v", err)
	}
	return nil
}

// progressReader 用于跟踪读取进度的io.Reader
type progressReader struct {
	reader   io.Reader
	progress func(int64)
}

func (r *progressReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if n > 0 && r.progress != nil {
		r.progress(int64(n))
	}
	return n, err
}

// CreateDirectory 创建目录
func (fs *SFTPFileSystem) CreateDirectory(path string) error {
	return fs.createRemoteDirectory(path)
}

// DeleteFile 删除文件或目录
func (fs *SFTPFileSystem) DeleteFile(path string) error {
	// 获取文件信息
	info, err := fs.sftpClient.Stat(path)
	if err != nil {
		return err
	}

	// 如果是目录，先删除所有内容
	if info.IsDir() {
		files, err := fs.sftpClient.ReadDir(path)
		if err != nil {
			return err
		}

		for _, file := range files {
			filePath := filepath.Join(path, file.Name())
			if err := fs.DeleteFile(filePath); err != nil {
				return err
			}
		}

		// 删除空目录
		return fs.sftpClient.RemoveDirectory(path)
	}

	// 删除文件
	return fs.sftpClient.Remove(path)
}
