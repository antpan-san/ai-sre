package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// StripOptionalFtAPIPrefix 剥离 URL 中的 /ft-api 前缀。
// 浏览器经 Nginx 访问时常为 /ft-api/api/...；若反代未去掉前缀，Gin 仅注册 /api 会 404。
// 在 r 的最前面使用本中间件后，两种反代写法均可命中路由。
func StripOptionalFtAPIPrefix() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/ft-api" || strings.HasPrefix(path, "/ft-api/") {
			c.Request.URL.Path = strings.TrimPrefix(path, "/ft-api")
			if c.Request.URL.Path == "" {
				c.Request.URL.Path = "/"
			}
		}
		c.Next()
	}
}
