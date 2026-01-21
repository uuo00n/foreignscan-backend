package utils

import (
	"github.com/google/uuid"
	"os"
)

// GenerateUUID 生成唯一标识符
func GenerateUUID() string {
	return uuid.New().String()
}

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, os.ModePerm)
}
