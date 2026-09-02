package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/session"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/ws"
	"github.com/gin-gonic/gin"
)

func TestRecordingMetadataRejectsStopBeforeStart(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := Setup(Dependencies{Sessions: session.NewManager(time.Second), Hub: ws.NewHub(), LogsRoot: t.TempDir()})
	sessionID := startTestSession(t, router)
	body := `{"audioFilePath":"set.wav","recordingStartTimestamp":"2026-09-02T12:00:01Z","recordingStopTimestamp":"2026-09-02T12:00:00Z"}`
	request := httptest.NewRequest(http.MethodPost, "/sessions/"+sessionID+"/recording/metadata", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("recording metadata returned %d, want %d", response.Code, http.StatusBadRequest)
	}
	endTestSession(t, router, sessionID)
}
