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

func TestMutationAuthorizationRejectsMissingTokenAndForeignOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := Setup(Dependencies{AuthToken: "test-token", Sessions: session.NewManager(time.Second), Hub: ws.NewHub()})
	body := bytes.NewBufferString(`{"name":"Set"}`)
	request := httptest.NewRequest(http.MethodPost, "/sessions", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing token returned %d, want %d", response.Code, http.StatusUnauthorized)
	}

	request = httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewBufferString(`{"name":"Set"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer test-token")
	request.Header.Set("Origin", "http://example.invalid")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("foreign origin returned %d, want %d", response.Code, http.StatusForbidden)
	}
}
