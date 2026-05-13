package middleware

import (
	"net/http"
	"time"

	"ft-backend/database"
	"ft-backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RequireEntitlementOrAdmin enforces feature billing for non-admin users.
func RequireEntitlementOrAdmin(featureKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, _ := c.Get("role")
		role, _ := roleVal.(string)
		if role == "admin" {
			c.Next()
			return
		}

		var fs models.FeatureBillingSetting
		if err := database.DB.Where("feature_key = ?", featureKey).First(&fs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.Next()
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "计费配置读取失败"})
			c.Abort()
			return
		}
		if !fs.BillingEnabled {
			c.Next()
			return
		}

		uid := models.UserIDFromContext(c.MustGet("userID"))
		if uid == uuid.Nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
			c.Abort()
			return
		}

		var ent models.Entitlement
		err := database.DB.Where("user_id = ? AND feature_key = ?", uid, featureKey).
			Where("valid_until IS NULL OR valid_until > ?", time.Now().UTC()).
			Order("valid_until DESC NULLS FIRST").
			First(&ent).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusForbidden, gin.H{
					"code": 403,
					"msg":  "当前功能需订阅后使用",
					"biz":  "PAYWALL_" + featureKey,
				})
				c.Abort()
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "权益校验失败"})
			c.Abort()
			return
		}
		c.Next()
	}
}
