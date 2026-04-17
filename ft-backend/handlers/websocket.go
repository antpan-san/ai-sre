package handlers

import (
	"net/http"

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

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
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