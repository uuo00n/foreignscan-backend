package middleware

import (
	"strings"
	"time"

	"foreignscan/internal/config"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func parseAllowedOrigins(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{"http://localhost:8080", "http://127.0.0.1:8080"}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"http://localhost:8080", "http://127.0.0.1:8080"}
	}
	return out
}

func hasWildcard(origins []string) bool {
	for _, o := range origins {
		if o == "*" {
			return true
		}
	}
	return false
}

// Setup 设置中间件
func Setup(r *gin.Engine) {
	cfg := config.Get()
	origins := parseAllowedOrigins(cfg.AllowedOrigins)

	// 配置CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With", "X-CSRF-Token"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Authorization"},
		AllowCredentials: !hasWildcard(origins),
		MaxAge:           12 * time.Hour,
	}))

	// 添加日志中间件（使用 Zap）
	r.Use(ZapLogger())

	// 添加恢复中间件
	r.Use(gin.Recovery())
}
