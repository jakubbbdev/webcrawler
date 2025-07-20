package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"web-scraper-api/internal/logger"

	"github.com/gorilla/websocket"
)

type WebSocketManager struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan interface{}
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	mutex      sync.RWMutex
	logger     *logger.Logger
}

type WebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Time    time.Time   `json:"time"`
	Message string      `json:"message,omitempty"`
}

func NewWebSocketManager(logger *logger.Logger) *WebSocketManager {
	return &WebSocketManager{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan interface{}, 100),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		logger:     logger,
	}
}

func (manager *WebSocketManager) Start() {
	for {
		select {
		case client := <-manager.register:
			manager.mutex.Lock()
			manager.clients[client] = true
			manager.mutex.Unlock()
			manager.logger.Infof("WebSocket client connected. Total clients: %d", len(manager.clients))

		case client := <-manager.unregister:
			manager.mutex.Lock()
			if _, ok := manager.clients[client]; ok {
				delete(manager.clients, client)
				client.Close()
			}
			manager.mutex.Unlock()
			manager.logger.Infof("WebSocket client disconnected. Total clients: %d", len(manager.clients))

		case message := <-manager.broadcast:
			manager.mutex.RLock()
			for client := range manager.clients {
				if err := client.WriteJSON(message); err != nil {
					manager.logger.Errorf("Error sending message to client: %v", err)
					client.Close()
					delete(manager.clients, client)
				}
			}
			manager.mutex.RUnlock()
		}
	}
}

func (manager *WebSocketManager) Broadcast(message interface{}) {
	manager.broadcast <- message
}

func (manager *WebSocketManager) BroadcastScrapingUpdate(url string, status string, data interface{}) {
	message := WebSocketMessage{
		Type: "scraping_update",
		Data: map[string]interface{}{
			"url":    url,
			"status": status,
			"data":   data,
		},
		Time: time.Now(),
	}
	manager.Broadcast(message)
}

func (manager *WebSocketManager) BroadcastBatchProgress(total int, completed int, current string) {
	message := WebSocketMessage{
		Type: "batch_progress",
		Data: map[string]interface{}{
			"total":     total,
			"completed": completed,
			"current":   current,
			"progress":  float64(completed) / float64(total) * 100,
		},
		Time: time.Now(),
	}
	manager.Broadcast(message)
}

func (manager *WebSocketManager) BroadcastError(url string, error string) {
	message := WebSocketMessage{
		Type: "error",
		Data: map[string]interface{}{
			"url":   url,
			"error": error,
		},
		Time: time.Now(),
	}
	manager.Broadcast(message)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

func (manager *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		manager.logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	// Register new client
	manager.register <- conn

	// Send welcome message
	welcomeMessage := WebSocketMessage{
		Type:    "connected",
		Message: "Connected to WebCrawler WebSocket",
		Time:    time.Now(),
	}
	if err := conn.WriteJSON(welcomeMessage); err != nil {
		manager.logger.Errorf("Error sending welcome message: %v", err)
		conn.Close()
		return
	}

	// Handle incoming messages
	go func() {
		defer func() {
			manager.unregister <- conn
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					manager.logger.Errorf("WebSocket read error: %v", err)
				}
				break
			}

			// Echo message back (for testing)
			var msg WebSocketMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				msg.Type = "echo"
				msg.Time = time.Now()
				if err := conn.WriteJSON(msg); err != nil {
					manager.logger.Errorf("Error sending echo message: %v", err)
					break
				}
			}
		}
	}()
}
