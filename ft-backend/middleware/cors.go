package middleware

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS applies a strict origin allowlist. Same-origin requests usually do not
// carry an Origin header and pass through without extra work.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Vary", "Origin")
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			} else {
				// 与「登录失败」区分：未命中白名单时请求不会进入业务 Handler，易被误认为账号密码错误。
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"code": 403,
					"msg": fmt.Sprintf(
						"CORS：浏览器 Origin「%s」未在 security.cors_allowed_origins 中。请把当前访问地址（含协议与端口，如 http://192.168.1.10:9080）写入 ft-backend/conf/config.yaml 后重启 opsfleet-backend。",
						origin,
					),
				})
				return
			}
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")
		c.Writer.Header().Set("Access-Control-Max-Age", "600")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Content-Type-Options", "nosniff")
		c.Writer.Header().Set("X-Frame-Options", "DENY")
		c.Writer.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Writer.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

func RequireRoles(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(c *gin.Context) {
		role, _ := c.Get("role")
		roleString, _ := role.(string)
		if _, ok := allowed[roleString]; !ok {
			c.JSON(403, gin.H{"code": 403, "msg": "权限不足"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return RequireRoles("admin")
}
