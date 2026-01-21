package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

var (
	once   sync.Once
	cached *Config
)

func Get() *Config {
	once.Do(func() {
		cached = Load()
	})
	return cached
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func findRepoRoot(start string) (string, bool) {
	dir := filepath.Clean(start)
	for i := 0; i < 15; i++ {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func normalizeUploadDir(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return p
	}
	p = filepath.Clean(p)
	if filepath.IsAbs(p) {
		return p
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

func stripOptionalQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func loadDotEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := stripOptionalQuotes(line[idx+1:])
		if key == "" {
			continue
		}
		if os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, val)
	}
}

func loadDotEnv() {
	if wd, err := os.Getwd(); err == nil {
		if repoRoot, ok := findRepoRoot(wd); ok {
			loadDotEnvFile(filepath.Join(repoRoot, ".env"))
			return
		}
		loadDotEnvFile(filepath.Join(wd, ".env"))
	}
	if exe, err := os.Executable(); err == nil {
		loadDotEnvFile(filepath.Join(filepath.Dir(exe), ".env"))
	}
}

func defaultUploadDir() string {
	if wd, err := os.Getwd(); err == nil {
		if repoRoot, ok := findRepoRoot(wd); ok {
			return filepath.Join(repoRoot, "cmd", "server", "uploads")
		}
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), "uploads")
	}
	return filepath.Join("cmd", "server", "uploads")
}

// Config 应用配置结构
type Config struct {
	Port             int    // 服务器端口
	PostgresDSN      string // PostgreSQL连接DSN
	UploadDir        string // 上传目录
	AllowedOrigins   string // 允许的CORS源
	DetectServiceURL string // YOLO服务地址
}

// Load 加载配置
func Load() *Config {
	loadDotEnv()

	// 默认配置
	cfg := &Config{
		Port:             3000,
		PostgresDSN:      "host=localhost user=postgres password=postgres dbname=foreignscan port=5432 sslmode=disable TimeZone=Asia/Shanghai",
		UploadDir:        defaultUploadDir(),
		AllowedOrigins:   "*",
		DetectServiceURL: "http://127.0.0.1:8077",
	}

	// 从环境变量加载配置（如果存在）
	if port := os.Getenv("PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		cfg.PostgresDSN = dsn
	}

	if origins := os.Getenv("ALLOWED_ORIGINS"); origins != "" {
		cfg.AllowedOrigins = origins
	}

	if svc := os.Getenv("DETECT_SERVICE_URL"); svc != "" {
		cfg.DetectServiceURL = svc
	}

	if uploadDir := os.Getenv("UPLOAD_DIR"); uploadDir != "" {
		cfg.UploadDir = normalizeUploadDir(uploadDir)
	}

	return cfg
}
