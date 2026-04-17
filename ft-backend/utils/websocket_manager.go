package utils

import (
	"encoding/json"
	"sync"

	"ft-backend/common/logger"

	"github.com/gorilla/websocket"
)

// GlobalWebSocketManager 全局WebSocket管理器
var GlobalWebSocketManager *WebSocketManager

// WebSocketManager WebSocket连接管理器
type WebSocketManager struct {
	clients    map[string]*WebSocketClient
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	broadcast  chan []byte
	mutex      sync.Mutex
}

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	ID      string
	Conn    *websocket.Conn
	Manager *WebSocketManager
	Send    chan []byte
}

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type    string      `json:"type"`
	UserID  string      `json:"user_id,omitempty"`
	FileID  string      `json:"file_id,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// NewWebSocketManager 创建新的WebSocket管理器
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[string]*WebSocketClient),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		broadcast:  make(chan []byte),
	}
}

// Start 启动WebSocket管理器
func (manager *WebSocketManager) Start() {
	for {
		select {
		case client := <-manager.register:
			manager.mutex.Lock()
			manager.clients[client.ID] = client
			manager.mutex.Unlock()
			logger.Info("WebSocket client registered: %s", client.ID)

		case client := <-manager.unregister:
			manager.mutex.Lock()
			if _, ok := manager.clients[client.ID]; ok {
				delete(manager.clients, client.ID)
				close(client.Send)
			}
			manager.mutex.Unlock()
			logger.Info("WebSocket client unregistered: %s", client.ID)

		case message := <-manager.broadcast:
			manager.mutex.Lock()
			for _, client := range manager.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(manager.clients, client.ID)
				}
			}
			manager.mutex.Unlock()
		}
	}
}

// RegisterClient 注册客户端
func (manager *WebSocketManager) RegisterClient(client *WebSocketClient) {
	manager.register <- client
}

// UnregisterClient 注销客户端
func (manager *WebSocketManager) UnregisterClient(client *WebSocketClient) {
	manager.unregister <- client
}

// Broadcast 广播消息
func (manager *WebSocketManager) Broadcast(message WebSocketMessage) {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		logger.Error("Failed to marshal broadcast message: %v", err)
		return
	}

	manager.broadcast <- jsonMessage
}

// SendToClient 发送消息给特定客户端
func (manager *WebSocketManager) SendToClient(clientID string, message WebSocketMessage) error {
	manager.mutex.Lock()
	client, ok := manager.clients[clientID]
	manager.mutex.Unlock()

	if !ok {
		return nil // 客户端不存在，忽略
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return err
	}

	client.Send <- jsonMessage
	return nil
}

// NewWebSocketClient 创建新的WebSocket客户端
func NewWebSocketClient(id string, conn *websocket.Conn, manager *WebSocketManager) *WebSocketClient {
	return &WebSocketClient{
		ID:      id,
		Conn:    conn,
		Manager: manager,
		Send:    make(chan []byte, 256),
	}
}

// ReadPump 从WebSocket连接读取消息
func (client *WebSocketClient) ReadPump() {
	defer func() {
		client.Manager.UnregisterClient(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(512) // 限制消息大小

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error("WebSocket read error: %v", err)
			}
			break
		}

		// 处理接收到的消息
		logger.Debug("Received message from client %s: %s", client.ID, message)
	}
}

// WritePump 向WebSocket连接写入消息
func (client *WebSocketClient) WritePump() {
	defer func() {
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			if !ok {
				// 通道已关闭
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error("WebSocket write error: %v", err)
				return
			}
		}
	}
}