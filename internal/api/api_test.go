package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/session"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/ws"
	"github.com/gin-gonic/gin"
)

func TestStartSessionPersistsHistory(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := Setup(Dependencies{
		Sessions: session.NewManager(20 * time.Second),
		Hub:      ws.NewHub(),
		LogsRoot: t.TempDir(),
	})

	request := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(`{"name":"Morning set"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("session creation returned %d, want %d", response.Code, http.StatusCreated)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("session creation returned invalid JSON: %v", err)
	}

	request = httptest.NewRequest(http.MethodGet, "/sessions", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("session history returned %d, want %d", response.Code, http.StatusOK)
	}
	if !bytes.Contains(response.Body.Bytes(), []byte("Morning set")) {
		t.Fatalf("session history did not contain the session name: %s", response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPost, "/sessions/"+created.ID+"/end", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("session end returned %d, want %d", response.Code, http.StatusNoContent)
	}
}
