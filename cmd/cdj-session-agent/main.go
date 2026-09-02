package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/api"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/prodjlink"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/session"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/ws"
)

func main() {
	address := flag.String("listen-address", "127.0.0.1:8080", "HTTP listen address")
	dataDir := flag.String("data-dir", "data", "session data directory")
	enableProDJLink := flag.Bool("enable-pro-dj-link", false, "connect to the Pro DJ Link network")
	flag.Parse()
	authToken := os.Getenv("CDJ_SESSION_API_TOKEN")
	if authToken == "" {
		log.Fatal("CDJ_SESSION_API_TOKEN must be set")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	logsRoot := filepath.Join(*dataDir, "logs")
	recordingsRoot := filepath.Join(*dataDir, "recordings")
	if err := os.MkdirAll(recordingsRoot, 0750); err != nil {
		log.Fatal("failed to create recordings directory: ", err)
	}
	deps := api.Dependencies{Context: ctx, AuthToken: authToken, Sessions: session.NewManager(20 * time.Second), Hub: ws.NewHub(), LogsRoot: logsRoot, RecordingsRoot: recordingsRoot}
	if *enableProDJLink {
		client, err := prodjlink.Connect(5 * time.Second)
		if err != nil {
			log.Fatal("failed to connect to Pro DJ Link: ", err)
		}
		deps.Decks = client
		deps.Devices = client
		updates := make(chan model.DeckState, 64)
		client.SetHandler(func(deck model.DeckState) {
			select {
			case updates <- deck:
			default:
			}
		})
		deps.DeckUpdates = updates
	}
	server := &http.Server{
		Addr:              *address,
		Handler:           api.Setup(deps),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Fatal("server shutdown failed: ", err)
		}
	}
}
