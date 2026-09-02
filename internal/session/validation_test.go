package session

import (
	"testing"
	"time"
)

func TestStartRemovesLineBreaksFromSessionName(t *testing.T) {
	manager := NewManager(time.Second)
	session, err := manager.Start("Morning\r\nset", time.Now())
	if err != nil {
		t.Fatalf("Start returned an error: %v", err)
	}
	if session.Name != "Morningset" {
		t.Fatalf("Start returned %q, want %q", session.Name, "Morningset")
	}
}
