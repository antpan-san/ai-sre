package handlers

import (
	"net/http"
	"strings"

	"ft-backend/common/config"
	"ft-backend/common/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"ft-backend/utils"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketHandler WebSocket连接处理
func WebSocketHandler(c *gin.Context) {
	userID := c.Param("user_id")
	cfg := c.MustGet("config").(*config.Config)

	if !isAllowedWebSocketOrigin(c.Request.Header.Get("Origin"), cfg.Security.CORSAllowedOrigins) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "msg": "WebSocket origin not allowed"})
		return
	}

	token, subprotocol := bearerTokenFromSubprotocol(c.GetHeader("Sec-WebSocket-Protocol"))
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "Invalid WebSocket token"})
		return
	}
	claims, err := utils.ValidateToken(token, cfg.JWT.SecretKey)
	if err != nil || claims.UserID != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "Invalid WebSocket token"})
		return
	}

	responseHeader := http.Header{}
	if subprotocol != "" {
		responseHeader.Set("Sec-WebSocket-Protocol", subprotocol)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, responseHeader)
	if err != nil {
		logger.Error("Failed to upgrade to WebSocket: %v", err)
		return
	}

	// 创建新的WebSocket客户端
	client := utils.NewWebSocketClient(userID, conn, utils.GlobalWebSocketManager)

	// 注册客户端
	client.Manager.RegisterClient(client)

	// 发送连接成功消息
	welcomeMsg := utils.WebSocketMessage{
		Type:    "connected",
		UserID:  userID,
		Message: "WebSocket connection established",
	}

	client.Send <- utils.MustMarshalJSON(welcomeMsg)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()
}

func bearerTokenFromSubprotocol(header string) (string, string) {
	for _, protocol := range strings.Split(header, ",") {
		protocol = strings.TrimSpace(protocol)
		if strings.HasPrefix(protocol, "bearer.") {
			return strings.TrimPrefix(protocol, "bearer."), protocol
		}
	}
	return "", ""
}

func isAllowedWebSocketOrigin(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return true
	}
	for _, allowed := range allowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
