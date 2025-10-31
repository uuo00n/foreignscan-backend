package utils

import (
	"github.com/google/uuid"
)

// GenerateUUID 生成唯一标识符
func GenerateUUID() string {
	return uuid.New().String()
}