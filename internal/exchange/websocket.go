package exchange

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

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
