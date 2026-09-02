package events

import (
	"encoding/json"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
	"os"
	"path/filepath"
	"sync"
)

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func Open(root, sessionID string) (*Logger, error) {
	if err := os.MkdirAll(root, 0750); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(filepath.Join(root, "session-"+sessionID+".jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return nil, err
	}
	return &Logger{file: file}, nil
}
func (l *Logger) Write(event model.Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err = l.file.Write(append(data, '\n')); err != nil {
		return err
	}
	return l.file.Sync()
}
func (l *Logger) Close() error { l.mu.Lock(); defer l.mu.Unlock(); return l.file.Close() }
