package ws

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	clientQueueSize = 32
	writeTimeout    = 5 * time.Second
	pingInterval    = 30 * time.Second
)

type client struct {
	conn *websocket.Conn
	done chan struct{}
	send chan any
	once sync.Once
}

type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

func (h *Hub) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	conn, err := (&websocket.Upgrader{}).Upgrade(writer, request, nil)
	if err != nil {
		return
	}
	client := &client{conn: conn, done: make(chan struct{}), send: make(chan any, clientQueueSize)}
	h.add(client)
	writerDone := make(chan struct{})
	go h.writeClient(client, writerDone)
	h.readClient(client)
	client.close()
	<-writerDone
	h.remove(client)
}

func (h *Hub) Broadcast(value any) {
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.Unlock()
	for _, client := range clients {
		select {
		case <-client.done:
		case client.send <- value:
		default:
			client.close()
		}
	}
}

func (h *Hub) add(client *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = struct{}{}
}

func (h *Hub) remove(client *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client)
}

func (h *Hub) readClient(client *client) {
	client.conn.SetReadLimit(1024)
	client.conn.SetReadDeadline(time.Now().Add(pingInterval * 2))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(pingInterval * 2))
	})
	for {
		if _, _, err := client.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Hub) writeClient(client *client, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-client.done:
			return
		case value := <-client.send:
			if !writeJSON(client.conn, value) {
				client.close()
				return
			}
		case <-ticker.C:
			if err := client.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeTimeout)); err != nil {
				client.close()
				return
			}
		}
	}
}

func writeJSON(conn *websocket.Conn, value any) bool {
	if err := conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
		return false
	}
	return conn.WriteJSON(value) == nil
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}
