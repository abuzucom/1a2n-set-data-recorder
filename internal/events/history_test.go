package events

import (
	"testing"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
)

func TestReadEventsRejectsTraversal(t *testing.T) {
	_, err := ReadEvents(t.TempDir(), "../outside")
	if err == nil {
		t.Fatal("ReadEvents accepted a traversal session ID")
	}
}

func TestListSessionsReadsStartAndEnd(t *testing.T) {
	root := t.TempDir()
	logger, err := Open(root, "session-1")
	if err != nil {
		t.Fatalf("Open returned an error: %v", err)
	}
	started := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)
	startEvent := model.NewEvent(model.EventSessionStart, "session-1", started)
	startEvent.SessionName = "Morning set"
	if err := logger.Write(startEvent); err != nil {
		t.Fatalf("Write returned an error: %v", err)
	}
	endEvent := model.NewEvent(model.EventSessionEnd, "session-1", started.Add(time.Hour))
	if err := logger.Write(endEvent); err != nil {
		t.Fatalf("Write returned an error: %v", err)
	}
	if err := logger.Close(); err != nil {
		t.Fatalf("Close returned an error: %v", err)
	}
	sessions, err := ListSessions(root)
	if err != nil {
		t.Fatalf("ListSessions returned an error: %v", err)
	}
	if len(sessions) != 1 || sessions[0].Name != "Morning set" || sessions[0].EndedAt.IsZero() {
		t.Fatalf("ListSessions returned %#v", sessions)
	}
}
