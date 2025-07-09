package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketConnection handles WebSocket connections to exchanges
type WebSocketConnection struct {
	conn    *websocket.Conn
	url     string
	running bool
}

// NewWebSocketConnection creates a new WebSocket connection
func NewWebSocketConnection(url string) (*WebSocketConnection, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	ws := &WebSocketConnection{
		conn:    conn,
		url:     url,
		running: true,
	}

	// Start ping/pong
	go ws.pingPong()

	return ws, nil
}

// Subscribe subscribes to a specific stream
func (ws *WebSocketConnection) Subscribe(stream string) error {
	subscribeMsg := map[string]interface{}{
		"method": "SUBSCRIBE",
		"params": []string{stream},
		"id":     1,
	}

	data, err := json.Marshal(subscribeMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe message: %w", err)
	}

	if err := ws.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	log.Printf("Subscribed to stream: %s", stream)
	return nil
}

// Close closes the WebSocket connection
func (ws *WebSocketConnection) Close() error {
	ws.running = false
	return ws.conn.Close()
}

// pingPong handles ping/pong to keep connection alive
func (ws *WebSocketConnection) pingPong() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for ws.running {
		select {
		case <-ticker.C:
			if err := ws.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to send ping: %v", err)
				return
			}
		}
	}
}

// ReadMessage reads a message from the WebSocket
func (ws *WebSocketConnection) ReadMessage() ([]byte, error) {
	_, message, err := ws.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}
	return message, nil
}

type WebSocketManager struct {
	conn          *websocket.Conn
	url           string
	subscriptions map[string]func([]byte)
	mu            sync.RWMutex
	reconnectChan chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewWebSocketManager(url string) *WebSocketManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketManager{
		url:           url,
		subscriptions: make(map[string]func([]byte)),
		reconnectChan: make(chan struct{}, 1),
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (w *WebSocketManager) Connect() error {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second

	conn, _, err := dialer.Dial(w.url, nil)
	if err != nil {
		return err
	}

	w.conn = conn
	go w.readMessages()
	go w.handleReconnection()

	return nil
}

func (w *WebSocketManager) readMessages() {
	defer w.conn.Close()

	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			_, message, err := w.conn.ReadMessage()
			if err != nil {
				log.Printf("WebSocket read error: %v", err)
				w.triggerReconnect()
				return
			}

			w.handleMessage(message)
		}
	}
}

func (w *WebSocketManager) handleMessage(message []byte) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	// There should be a message processing logic here.
	// Depending on the type of message, we call the appropriate callback.
	for _, callback := range w.subscriptions {
		callback(message)
	}
}

func (w *WebSocketManager) triggerReconnect() {
	select {
	case w.reconnectChan <- struct{}{}:
	default:
	}
}

func (w *WebSocketManager) handleReconnection() {
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.reconnectChan:
			log.Println("Attempting to reconnect...")
			time.Sleep(5 * time.Second)
			if err := w.Connect(); err != nil {
				log.Printf("Reconnection failed: %v", err)
			}
		}
	}
}
