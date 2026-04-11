// Package websocket implements the WebSocket connection hub for real-time notifications.
package websocket

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-wego/wego/internal/entity"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// upgrader promotes an HTTP connection to WebSocket.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins in development; tighten in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// client represents a single WebSocket connection from a user.
type client struct {
	userID uuid.UUID
	conn   *websocket.Conn
	send   chan *entity.Notification
}

// Hub manages all active WebSocket clients and fans out notifications.
// It implements the notification.Publisher interface.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID][]*client // userID → connections (multi-device)
	logger  *zap.Logger
}

// NewHub creates an initialised Hub.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients: make(map[uuid.UUID][]*client),
		logger:  logger,
	}
}

// ——— Publisher interface implementation ——————————————————————————————————————

// Publish fans a notification out to all connections for the given userID.
func (h *Hub) Publish(userID uuid.UUID, n *entity.Notification) {
	h.mu.RLock()
	conns := h.clients[userID]
	h.mu.RUnlock()

	for _, c := range conns {
		select {
		case c.send <- n:
		default:
			// Slow client — drop message to avoid blocking the hub
			h.logger.Warn("ws: slow client, dropping notification",
				zap.String("user_id", userID.String()),
			)
		}
	}
}

// ——— HTTP upgrade handler —————————————————————————————————————————————————————

// ServeWS upgrades an HTTP request to WebSocket and registers the client.
// userID must be extracted from the auth middleware before calling this.
func (h *Hub) ServeWS(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("ws: upgrade failed", zap.Error(err))
		return
	}

	cl := &client{
		userID: userID,
		conn:   conn,
		send:   make(chan *entity.Notification, 32),
	}

	h.register(cl)
	go h.writePump(cl)
	h.readPump(cl) // blocks until the connection closes
}

// ——— Internal helpers —————————————————————————————————————————————————————————

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c.userID] = append(h.clients[c.userID], c)
	h.mu.Unlock()
	h.logger.Info("ws: client connected", zap.String("user_id", c.userID.String()))
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	conns := h.clients[c.userID]
	for i, conn := range conns {
		if conn == c {
			h.clients[c.userID] = append(conns[:i], conns[i+1:]...)
			break
		}
	}
	if len(h.clients[c.userID]) == 0 {
		delete(h.clients, c.userID)
	}
	close(c.send)
	h.logger.Info("ws: client disconnected", zap.String("user_id", c.userID.String()))
}

// writePump drains the send channel and writes JSON messages to the WebSocket.
func (h *Hub) writePump(c *client) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case n, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(n); err != nil {
				h.logger.Warn("ws: write error", zap.Error(err))
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump reads from the WebSocket (mainly to detect disconnects and pong messages).
func (h *Hub) readPump(c *client) {
	defer h.unregister(c)

	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			break
		}
	}
}
