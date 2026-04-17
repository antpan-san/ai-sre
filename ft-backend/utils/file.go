package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// GenerateUniqueFilename 生成唯一文件名
func GenerateUniqueFilename(originalFilename string) string {
	extension := filepath.Ext(originalFilename)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s%d", originalFilename, time.Now().UnixNano())))
	return fmt.Sprintf("%x%s", hash[:16], extension)
}

// GetFileExtension 获取文件扩展名
func GetFileExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if len(ext) > 0 {
		return ext[1:] // 去掉点号
	}
	return ""
}

// ValidateFileExtension 验证文件扩展名
func ValidateFileExtension(filename string, allowedExtensions []string) bool {
	ext := GetFileExtension(filename)
	if ext == "" {
		return false
	}

	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			return true
		}
	}

	return false
}

// CalculateFileHash 计算文件哈希值
func CalculateFileHash(file multipart.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	// 重置文件指针到开头
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// SaveUploadedFile 保存上传的文件
func SaveUploadedFile(file multipart.File, header *multipart.FileHeader, uploadDir, uniqueFilename string) (string, int64, error) {
	// 创建上传目录（如果不存在）
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create upload directory: %w", err)
	}

	filePath := filepath.Join(uploadDir, uniqueFilename)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容
	fileSize, err := io.Copy(dst, file)
	if err != nil {
		return "", 0, fmt.Errorf("failed to save file: %w", err)
	}

	return filePath, fileSize, nil
}

// DeleteFile 删除文件
func DeleteFile(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，视为删除成功
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}