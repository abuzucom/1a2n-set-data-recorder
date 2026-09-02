package api

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/events"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/session"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/ws"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

type Dependencies struct {
	Sessions    *session.Manager
	Hub         *ws.Hub
	Logger      *events.Logger
	LogsRoot    string
	Decks       interface{ Decks() []model.DeckState }
	Devices     interface{ Devices() []model.DeviceState }
	DeckUpdates <-chan model.DeckState
}

const maxRequestBodyBytes = 64 * 1024

func Setup(deps Dependencies) *gin.Engine {
	gin.EnableJsonDecoderDisallowUnknownFields()
	var logMu sync.Mutex
	logger := deps.Logger
	persistEvent := func(event model.Event) error {
		logMu.Lock()
		defer logMu.Unlock()
		if logger != nil {
			if err := logger.Write(event); err != nil {
				return err
			}
		}
		deps.Hub.Broadcast(event)
		return nil
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Header("Content-Security-Policy", "default-src 'self'; connect-src 'self' ws: wss:; style-src 'self'; script-src 'self'")
		c.Next()
	})
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxRequestBodyBytes)
		c.Next()
	})
	r.GET("/status", func(c *gin.Context) {
		decks := []model.DeckState{}
		devices := []model.DeviceState{}
		if deps.Decks != nil {
			decks = deps.Decks.Decks()
		}
		if deps.Devices != nil {
			devices = deps.Devices.Devices()
		}
		c.JSON(http.StatusOK, gin.H{"session": deps.Sessions.ActiveSession(), "decks": decks, "devices": devices})
	})
	r.GET("/ws", gin.WrapH(deps.Hub))
	r.GET("/sessions", func(c *gin.Context) {
		sessions, err := events.ListSessions(deps.LogsRoot)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read sessions"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	})
	r.GET("/sessions/:id/events", func(c *gin.Context) {
		events, err := events.ReadEvents(deps.LogsRoot, c.Param("id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "session events not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"events": events})
	})
	if deps.DeckUpdates != nil {
		go func() {
			for deck := range deps.DeckUpdates {
				deps.Hub.Broadcast(deck)
				events, err := deps.Sessions.UpdateDeck(deck)
				if err != nil {
					continue
				}
				for _, event := range events {
					_ = persistEvent(event)
				}
			}
		}()
	}
	r.POST("/sessions", func(c *gin.Context) {
		var body struct {
			Name string `json:"name" binding:"required,max=100"`
		}
		if c.ShouldBindJSON(&body) != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		value, err := deps.Sessions.Start(body.Name, time.Now().UTC())
		if err != nil {
			c.JSON(409, gin.H{"error": err.Error()})
			return
		}
		if deps.LogsRoot != "" {
			logMu.Lock()
			logger, err = events.Open(deps.LogsRoot, value.ID)
			if err == nil {
				startEvent := model.NewEvent(model.EventSessionStart, value.ID, value.StartedAt)
				startEvent.SessionName = value.Name
				err = logger.Write(startEvent)
			}
			logMu.Unlock()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session log"})
				return
			}
		}
		c.JSON(201, value)
	})
	r.POST("/sessions/:id/dj-handoff", func(c *gin.Context) {
		var body struct {
			Previous string `json:"previousDjName"`
			Next     string `json:"nextDjName"`
			External bool   `json:"isExternalSource"`
		}
		if c.ShouldBindJSON(&body) != nil {
			c.JSON(400, gin.H{"error": "invalid request"})
			return
		}
		active := deps.Sessions.ActiveSession()
		if c.Param("id") != active.ID {
			c.JSON(404, gin.H{"error": "unknown session"})
			return
		}
		nextDJName, valid := normalizeName(body.Next)
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "nextDjName must contain 1 to 100 characters"})
			return
		}
		previousDJName, valid := normalizeOptionalName(body.Previous)
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "previousDjName must contain at most 100 characters"})
			return
		}
		event := model.NewEvent(model.EventDJHandoff, active.ID, time.Now())
		event.PreviousDJName = previousDJName
		event.NextDJName = nextDJName
		event.ExternalSource = body.External
		logMu.Lock()
		if logger != nil {
			if err := logger.Write(event); err != nil {
				logMu.Unlock()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write session log"})
				return
			}
		}
		logMu.Unlock()
		deps.Hub.Broadcast(event)
		c.JSON(201, event)
	})
	r.POST("/sessions/:id/time-source", func(c *gin.Context) {
		active := deps.Sessions.ActiveSession()
		if c.Param("id") != active.ID {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown session"})
			return
		}
		var body struct {
			Source string `json:"source" binding:"required,max=100"`
		}
		if c.ShouldBindJSON(&body) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		source, valid := normalizeName(body.Source)
		if !valid {
			c.JSON(http.StatusBadRequest, gin.H{"error": "source must contain 1 to 100 characters"})
			return
		}
		event := model.NewEvent(model.EventTimeSourceChanged, active.ID, time.Now())
		event.TimeSource = source
		if err := persistEvent(event); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write session log"})
			return
		}
		c.JSON(http.StatusCreated, event)
	})
	r.POST("/sessions/:id/end", func(c *gin.Context) {
		active := deps.Sessions.ActiveSession()
		if c.Param("id") != active.ID {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown session"})
			return
		}
		events, err := deps.Sessions.End(time.Now().UTC())
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		logMu.Lock()
		for _, event := range events {
			if logger != nil {
				if err := logger.Write(event); err != nil {
					logMu.Unlock()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write session log"})
					return
				}
			}
			deps.Hub.Broadcast(event)
		}
		if logger != nil {
			_ = logger.Close()
			logger = nil
		}
		logMu.Unlock()
		c.Status(http.StatusNoContent)
	})
	r.POST("/sessions/:id/recording/:action", func(c *gin.Context) {
		active := deps.Sessions.ActiveSession()
		if c.Param("id") != active.ID {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown session"})
			return
		}
		action := c.Param("action")
		if action != "start" && action != "stop" {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown recording action"})
			return
		}
		eventType := model.EventRecordingStart
		if action == "stop" {
			eventType = model.EventRecordingStop
		}
		event := model.NewEvent(eventType, active.ID, time.Now())
		logMu.Lock()
		if logger != nil {
			if err := logger.Write(event); err != nil {
				logMu.Unlock()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write session log"})
				return
			}
		}
		logMu.Unlock()
		deps.Hub.Broadcast(event)
		c.JSON(http.StatusCreated, event)
	})
	r.POST("/sessions/:id/recording/metadata", func(c *gin.Context) {
		active := deps.Sessions.ActiveSession()
		if c.Param("id") != active.ID {
			c.JSON(http.StatusNotFound, gin.H{"error": "unknown session"})
			return
		}
		var body struct {
			AudioPath      string  `json:"audioFilePath" binding:"required"`
			OffsetSeconds  float64 `json:"offsetSeconds"`
			RecordingStart string  `json:"recordingStartTimestamp"`
			RecordingStop  string  `json:"recordingStopTimestamp"`
		}
		if c.ShouldBindJSON(&body) != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		path := filepath.Clean(body.AudioPath)
		if filepath.IsAbs(path) || path == "." || strings.HasPrefix(path, "..") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "audioFilePath must be relative"})
			return
		}
		event := model.NewEvent(model.EventRecordingMetadata, active.ID, time.Now())
		if body.RecordingStart != "" {
			recordingStart, err := time.Parse(time.RFC3339Nano, body.RecordingStart)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recordingStartTimestamp"})
				return
			}
			event.RecordingStart = recordingStart.UTC()
		}
		if body.RecordingStop != "" {
			recordingStop, err := time.Parse(time.RFC3339Nano, body.RecordingStop)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recordingStopTimestamp"})
				return
			}
			event.RecordingStop = recordingStop.UTC()
		}
		if !event.RecordingStart.IsZero() && !event.RecordingStop.IsZero() && event.RecordingStop.Before(event.RecordingStart) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "recordingStopTimestamp precedes recordingStartTimestamp"})
			return
		}
		event.AudioPath = filepath.ToSlash(path)
		event.OffsetSeconds = body.OffsetSeconds
		logMu.Lock()
		if logger != nil {
			if err := logger.Write(event); err != nil {
				logMu.Unlock()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write session log"})
				return
			}
		}
		logMu.Unlock()
		deps.Hub.Broadcast(event)
		c.JSON(http.StatusCreated, event)
	})
	r.StaticFile("/", "ui/dist/index.html")
	r.StaticFile("/style.css", "ui/dist/style.css")
	r.StaticFile("/app.js", "ui/dist/app.js")
	return r
}

func normalizeName(value string) (string, bool) {
	value = strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(value))
	return value, value != "" && len(value) <= 100
}

func normalizeOptionalName(value string) (string, bool) {
	value = strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(value))
	return value, len(value) <= 100
}
