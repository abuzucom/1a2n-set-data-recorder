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

const (
	maxListedSessions   = 100
	maxSessionTailBytes = 64 * 1024
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
		if len(sessions) == maxListedSessions {
			break
		}
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "session-") || !strings.HasSuffix(entry.Name(), ".jsonl") || strings.HasSuffix(entry.Name(), ".identifications.jsonl") {
			continue
		}
		event, err := readFirstEvent(filepath.Join(root, entry.Name()))
		if err != nil {
			continue
		}
		if event.EventType != model.EventSessionStart {
			continue
		}
		session := model.Session{ID: event.SessionID, Name: event.SessionName, StartedAt: event.Timestamp}
		lastEvent, err := readLastEvent(filepath.Join(root, entry.Name()))
		if err == nil && lastEvent.EventType == model.EventSessionEnd {
			session.EndedAt = lastEvent.Timestamp
		}
		sessions = append(sessions, session)
	}
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].StartedAt.After(sessions[j].StartedAt)
	})
	return sessions, nil
}

func readLastEvent(path string) (model.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.Event{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return model.Event{}, err
	}
	start := info.Size() - maxSessionTailBytes
	if start < 0 {
		start = 0
	}
	if _, err := file.Seek(start, 0); err != nil {
		return model.Event{}, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	var lastEvent model.Event
	for scanner.Scan() {
		var event model.Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			if start != 0 {
				continue
			}
			return model.Event{}, err
		}
		lastEvent = event
	}
	if err := scanner.Err(); err != nil {
		return model.Event{}, err
	}
	if lastEvent.EventID == "" {
		return model.Event{}, errors.New("session log has no events")
	}
	return lastEvent, nil
}

func readFirstEvent(path string) (model.Event, error) {
	file, err := os.Open(path)
	if err != nil {
		return model.Event{}, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	if !scanner.Scan() {
		return model.Event{}, scanner.Err()
	}
	var event model.Event
	if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
		return model.Event{}, err
	}
	return event, nil
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
