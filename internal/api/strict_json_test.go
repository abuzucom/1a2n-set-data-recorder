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

func TestStartSessionRejectsUnknownJSONField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := Setup(Dependencies{Sessions: session.NewManager(time.Second), Hub: ws.NewHub()})
	request := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(`{"name":"Set","unexpected":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("session creation returned %d, want %d", response.Code, http.StatusBadRequest)
	}
}
