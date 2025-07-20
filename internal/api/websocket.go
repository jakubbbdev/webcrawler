package api

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"web-scraper-api/internal/logger"
	"web-scraper-api/internal/scheduler"

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
	Type string      `json:"type"`
	Data interface{} `json:"data"`
	Time time.Time   `json:"time"`
}

type ScrapingUpdate struct {
	URL    string      `json:"url"`
	Status string      `json:"status"`
	Data   interface{} `json:"data,omitempty"`
}

type BatchProgress struct {
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	Progress  int    `json:"progress"`
	Current   string `json:"current"`
}

type ErrorMessage struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}

type ScheduledJobUpdate struct {
	JobID     string `json:"job_id"`
	JobName   string `json:"job_name"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at,omitempty"`
	EndedAt   string `json:"ended_at,omitempty"`
	Duration  string `json:"duration,omitempty"`
	Error     string `json:"error,omitempty"`
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

func (w *WebSocketManager) Start() {
	for {
		select {
		case client := <-w.register:
			w.mutex.Lock()
			w.clients[client] = true
			w.mutex.Unlock()
			w.logger.Infof("WebSocket client connected. Total clients: %d", len(w.clients))

			// Send welcome message
			welcomeMsg := WebSocketMessage{
				Type: "connected",
				Data: map[string]interface{}{
					"message": "Connected to WebCrawler WebSocket",
					"time":    time.Now(),
				},
				Time: time.Now(),
			}
			w.sendToClient(client, welcomeMsg)

		case client := <-w.unregister:
			w.mutex.Lock()
			delete(w.clients, client)
			w.mutex.Unlock()
			client.Close()
			w.logger.Infof("WebSocket client disconnected. Total clients: %d", len(w.clients))

		case message := <-w.broadcast:
			w.broadcastMessage(message)
		}
	}
}

func (w *WebSocketManager) HandleWebSocket(writer http.ResponseWriter, request *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}

	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		w.logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}

	w.register <- conn

	// Handle incoming messages
	go func() {
		defer func() {
			w.unregister <- conn
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					w.logger.Errorf("WebSocket read error: %v", err)
				}
				break
			}

			// Echo message back for testing
			var msg map[string]interface{}
			if err := json.Unmarshal(message, &msg); err == nil {
				echoMsg := WebSocketMessage{
					Type: "echo",
					Data: msg,
					Time: time.Now(),
				}
				w.sendToClient(conn, echoMsg)
			}
		}
	}()
}

func (w *WebSocketManager) broadcastMessage(message interface{}) {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	for client := range w.clients {
		w.sendToClient(client, message)
	}
}

func (w *WebSocketManager) sendToClient(client *websocket.Conn, message interface{}) {
	err := client.WriteJSON(message)
	if err != nil {
		w.logger.Errorf("Failed to send WebSocket message: %v", err)
		client.Close()
		delete(w.clients, client)
	}
}

func (w *WebSocketManager) BroadcastScrapingUpdate(url, status string, data interface{}) {
	msg := WebSocketMessage{
		Type: "scraping_update",
		Data: ScrapingUpdate{
			URL:    url,
			Status: status,
			Data:   data,
		},
		Time: time.Now(),
	}
	w.broadcast <- msg
}

func (w *WebSocketManager) BroadcastBatchProgress(total, completed int, current string) {
	progress := 0
	if total > 0 {
		progress = (completed * 100) / total
	}

	msg := WebSocketMessage{
		Type: "batch_progress",
		Data: BatchProgress{
			Total:     total,
			Completed: completed,
			Progress:  progress,
			Current:   current,
		},
		Time: time.Now(),
	}
	w.broadcast <- msg
}

func (w *WebSocketManager) BroadcastError(url, error string) {
	msg := WebSocketMessage{
		Type: "error",
		Data: ErrorMessage{
			URL:   url,
			Error: error,
		},
		Time: time.Now(),
	}
	w.broadcast <- msg
}

// New methods for scheduled job updates
func (w *WebSocketManager) BroadcastScheduledJobStart(jobResult *scheduler.JobResult) {
	msg := WebSocketMessage{
		Type: "scheduled_job_start",
		Data: ScheduledJobUpdate{
			JobID:     jobResult.JobID,
			JobName:   jobResult.JobName,
			Status:    string(jobResult.Status),
			StartedAt: jobResult.StartedAt.Format(time.RFC3339),
		},
		Time: time.Now(),
	}
	w.broadcast <- msg
}

func (w *WebSocketManager) BroadcastScheduledJobComplete(jobResult *scheduler.JobResult) {
	msg := WebSocketMessage{
		Type: "scheduled_job_complete",
		Data: ScheduledJobUpdate{
			JobID:     jobResult.JobID,
			JobName:   jobResult.JobName,
			Status:    string(jobResult.Status),
			StartedAt: jobResult.StartedAt.Format(time.RFC3339),
			EndedAt:   jobResult.EndedAt.Format(time.RFC3339),
			Duration:  jobResult.Duration.String(),
		},
		Time: time.Now(),
	}
	w.broadcast <- msg
}

func (w *WebSocketManager) BroadcastScheduledJobError(jobResult *scheduler.JobResult) {
	msg := WebSocketMessage{
		Type: "scheduled_job_error",
		Data: ScheduledJobUpdate{
			JobID:     jobResult.JobID,
			JobName:   jobResult.JobName,
			Status:    string(jobResult.Status),
			StartedAt: jobResult.StartedAt.Format(time.RFC3339),
			EndedAt:   jobResult.EndedAt.Format(time.RFC3339),
			Duration:  jobResult.Duration.String(),
			Error:     jobResult.Error,
		},
		Time: time.Now(),
	}
	w.broadcast <- msg
}

func (w *WebSocketManager) BroadcastScheduledJobList(jobs []*scheduler.ScheduledJob) {
	msg := WebSocketMessage{
		Type: "scheduled_jobs_list",
		Data: map[string]interface{}{
			"jobs":  jobs,
			"count": len(jobs),
		},
		Time: time.Now(),
	}
	w.broadcast <- msg
}

func (w *WebSocketManager) BroadcastScheduledJobStats(stats map[string]interface{}) {
	msg := WebSocketMessage{
		Type: "scheduled_jobs_stats",
		Data: stats,
		Time: time.Now(),
	}
	w.broadcast <- msg
}
