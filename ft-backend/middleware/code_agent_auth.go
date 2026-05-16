package middleware

import (
	"net/http"
	"strings"

	"ft-backend/common/config"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const codeAgentFingerprintHeader = "X-OpsFleet-Agent-Fingerprint"

// CodeAgentAuth authenticates code-agent worker requests (machine identity, not user JWT).
func CodeAgentAuth(cfg *config.Config) gin.HandlerFunc {
	_ = cfg
	return func(c *gin.Context) {
		auth := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "Authorization header format must be Bearer {token}"})
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		fingerprint := strings.TrimSpace(c.GetHeader(codeAgentFingerprintHeader))
		if token == "" || len(fingerprint) != 64 {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid agent token or fingerprint"})
			c.Abort()
			return
		}
		tokenHash := services.HashSecretForAgent(token)
		fpHash := services.HashSecretForAgent(fingerprint)
		binding, err := services.ResolveCodeAgentBinding(tokenHash, fpHash)
		if err != nil || binding == nil {
			cfgTok := config.ResolvedAutoIterationConfig().CodeAgentToken
			if cfgTok != "" && token == cfgTok {
				_ = services.EnsureCodeAgentBinding(cfgTok, fingerprint)
				binding, err = services.ResolveCodeAgentBinding(tokenHash, fpHash)
			}
		}
		if err != nil || binding == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid code agent credentials"})
			c.Abort()
			return
		}
		c.Set("codeAgentBindingID", binding.ID)
		c.Set("codeAgentName", binding.Name)
		c.Next()
	}
}

// CodeAgentBindingID reads the authenticated agent binding from context.
func CodeAgentBindingID(c *gin.Context) uuid.UUID {
	if v, ok := c.Get("codeAgentBindingID"); ok {
		if id, ok := v.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}
