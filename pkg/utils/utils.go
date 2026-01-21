package utils

import (
	"crypto/rand"
	"encoding/hex"
	"os"

	"github.com/google/uuid"
)

// GenerateUUID 生成唯一标识符
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateObjectID 生成类似MongoDB ObjectID的24位hex字符串
// 用于保持数据库ID格式兼容
func GenerateObjectID() string {
	b := make([]byte, 12)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to uuid if rand fails (unlikely)
		return hex.EncodeToString([]byte(uuid.New().String()))[:24]
	}
	return hex.EncodeToString(b)
}

// EnsureDir 确保目录存在，如果不存在则创建
func EnsureDir(dirPath string) error {
	return os.MkdirAll(dirPath, os.ModePerm)
}
