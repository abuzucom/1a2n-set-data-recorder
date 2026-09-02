package events

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
)

func ListSessions(root string) ([]model.Session, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []model.Session{}, nil
	}
	if err != nil {
		return nil, err
	}
	sessions := make([]model.Session, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "session-") || !strings.HasSuffix(entry.Name(), ".jsonl") || strings.HasSuffix(entry.Name(), ".identifications.jsonl") {
			continue
		}
		events, err := ReadEvents(root, strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "session-"), ".jsonl"))
		if err != nil || len(events) == 0 {
			continue
		}
		session := model.Session{ID: events[0].SessionID, Name: events[0].SessionName, StartedAt: events[0].Timestamp}
		for _, event := range events {
			if event.EventType == model.EventSessionEnd {
				session.EndedAt = event.Timestamp
			}
		}
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt.After(sessions[j].StartedAt)
	})
	return sessions, nil
}

func ReadEvents(root, sessionID string) ([]model.Event, error) {
	if sessionID == "" || strings.ContainsAny(sessionID, `\\/:`) || strings.Contains(sessionID, "..") {
		return nil, errors.New("invalid session ID")
	}
	path := filepath.Join(root, "session-"+sessionID+".jsonl")
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	result := make([]model.Event, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		var event model.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, err
		}
		if event.SessionID == sessionID {
			result = append(result, event)
		}
	}
	return result, scanner.Err()
}
