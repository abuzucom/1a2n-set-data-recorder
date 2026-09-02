package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/session"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/ws"
	"github.com/gin-gonic/gin"
)

func TestRecordingMetadataPersistsStartTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := Setup(Dependencies{Sessions: session.NewManager(time.Second), Hub: ws.NewHub(), LogsRoot: t.TempDir()})
	sessionID := startTestSession(t, router)
	start := "2026-09-02T12:00:00.123456789Z"
	body := `{"audioFilePath":"recordings/set.wav","offsetSeconds":0.25,"recordingStartTimestamp":"` + start + `"}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/"+sessionID+"/recording/metadata", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("recording metadata returned %d, want %d", response.Code, http.StatusCreated)
	}
	var event model.Event
	if err := json.Unmarshal(response.Body.Bytes(), &event); err != nil {
		t.Fatalf("recording metadata returned invalid JSON: %v", err)
	}
	if event.RecordingStart.Format(time.RFC3339Nano) != start {
		t.Fatalf("recording metadata returned %s, want %s", event.RecordingStart.Format(time.RFC3339Nano), start)
	}
	endTestSession(t, router, sessionID)
}

func startTestSession(t *testing.T, router http.Handler) string {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(`{"name":"Test set"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	var created struct {
		ID string `json:"id"`
	}
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &created) != nil || created.ID == "" {
		t.Fatalf("session creation returned %d with %s", response.Code, response.Body.String())
	}
	return created.ID
}

func endTestSession(t *testing.T, router http.Handler, sessionID string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/sessions/"+sessionID+"/end", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("session end returned %d", response.Code)
	}
}
