package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestHubBroadcastDeliversMessage(t *testing.T) {
	hub := NewHub()
	server := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	defer server.Close()

	address := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(address, nil)
	if err != nil {
		t.Fatalf("Dial returned an error: %v", err)
	}
	defer conn.Close()

	hub.Broadcast(map[string]string{"type": "deck.status"})
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("SetReadDeadline returned an error: %v", err)
	}
	var message map[string]string
	if err := conn.ReadJSON(&message); err != nil {
		t.Fatalf("ReadJSON returned an error: %v", err)
	}
	if message["type"] != "deck.status" {
		t.Fatalf("ReadJSON returned %#v", message)
	}
}
