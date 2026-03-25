package utils

import (
	"path"
	"path/filepath"
	"strings"

	"foreignscan/internal/config"
)

// NormalizeToUploadsWebPath 将输入路径规范化为 Web 可访问的 /uploads/... 路径。
// 仅返回可映射到上传目录的路径，无法映射时返回空字符串。
func NormalizeToUploadsWebPath(raw string) string {
	return normalizeToUploadsWebPath(raw, config.Get().UploadDir)
}

// NormalizeToStoredUploadsPath 将路径规范化为存库使用的 uploads/... 形式。
func NormalizeToStoredUploadsPath(raw string) string {
	web := NormalizeToUploadsWebPath(raw)
	if web == "" {
		return ""
	}
	return strings.TrimPrefix(web, "/")
}

func normalizeToUploadsWebPath(raw, uploadDir string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return ""
	}

	raw = strings.ReplaceAll(raw, "\\", "/")

	if filepath.IsAbs(raw) {
		uploadDir = strings.TrimSpace(uploadDir)
		if uploadDir == "" {
			return ""
		}

		absPath := filepath.Clean(raw)
		root := filepath.Clean(uploadDir)
		rel, err := filepath.Rel(root, absPath)
		if err != nil {
			return ""
		}
		rel = filepath.Clean(rel)
		if rel == "." {
			return "/uploads"
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return ""
		}
		rel = strings.TrimPrefix(filepath.ToSlash(rel), "/")
		if rel == "" {
			return "/uploads"
		}
		return "/uploads/" + rel
	}

	cleaned := path.Clean("/" + strings.TrimPrefix(raw, "/"))
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return ""
	}
	if cleaned == "uploads" {
		return "/uploads"
	}
	if strings.HasPrefix(cleaned, "uploads/") {
		return "/" + cleaned
	}
	return "/uploads/" + cleaned
}
