package main

import (
	"flag"
	"log"
	"net/http"
	"path/filepath"
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
	deps := api.Dependencies{Sessions: session.NewManager(20 * time.Second), Hub: ws.NewHub(), LogsRoot: filepath.Join(*dataDir, "logs")}
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
	log.Fatal(server.ListenAndServe())
}
