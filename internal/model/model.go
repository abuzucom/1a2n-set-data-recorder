package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"time"
)

const (
	EventSessionStart        = "session.started"
	EventSessionEnd          = "session.ended"
	EventTrackOnAir          = "track.on_air"
	EventTrackOffAir         = "track.off_air"
	EventDJHandoff           = "dj.handoff"
	EventSilencePeriod       = "silence.period"
	EventRecordingStart      = "recording.started"
	EventRecordingStop       = "recording.stopped"
	EventRecordingMetadata   = "recording.metadata"
	EventTimeSourceChanged   = "time.source_changed"
	EventTrackIdentification = "track.identification"
)

type Session struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	StartedAt time.Time `json:"startedAt"`
	EndedAt   time.Time `json:"endedAt,omitempty"`
}

type DeckState struct {
	DeckID     string    `json:"deckId"`
	Player     int       `json:"playerNumber"`
	Model      string    `json:"model"`
	TrackID    uint32    `json:"trackId"`
	Title      string    `json:"title"`
	Artist     string    `json:"artist"`
	Album      string    `json:"album"`
	Key        string    `json:"key"`
	BPM        float64   `json:"bpm"`
	Pitch      float64   `json:"pitch"`
	Beat       uint32    `json:"beat"`
	BeatInBar  uint8     `json:"beatInBar"`
	SourceSlot string    `json:"sourceSlot"`
	PlayState  string    `json:"playState"`
	IsOnAir    bool      `json:"isOnAir"`
	IsMaster   bool      `json:"isMaster"`
	IsSync     bool      `json:"isSync"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type DeviceState struct {
	DeviceID   int       `json:"deviceId"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	Address    string    `json:"address"`
	LastActive time.Time `json:"lastActive"`
}

type Event struct {
	EventID         string          `json:"eventId"`
	EventType       string          `json:"eventType"`
	Timestamp       time.Time       `json:"timestamp"`
	SessionID       string          `json:"sessionId"`
	SessionName     string          `json:"sessionName,omitempty"`
	PlayID          string          `json:"playId,omitempty"`
	DeckID          string          `json:"deckId,omitempty"`
	Player          int             `json:"playerNumber,omitempty"`
	TrackID         uint32          `json:"trackId,omitempty"`
	Title           string          `json:"title,omitempty"`
	Artist          string          `json:"artist,omitempty"`
	DurationSeconds float64         `json:"durationSeconds,omitempty"`
	IsSampleLike    bool            `json:"isSampleLike"`
	PreviousDJName  string          `json:"previousDjName,omitempty"`
	NextDJName      string          `json:"nextDjName,omitempty"`
	ExternalSource  bool            `json:"isExternalSource,omitempty"`
	AudioPath       string          `json:"audioPath,omitempty"`
	OffsetSeconds   float64         `json:"offsetSeconds,omitempty"`
	RecordingStart  time.Time       `json:"recordingStartTimestamp,omitempty"`
	RecordingStop   time.Time       `json:"recordingStopTimestamp,omitempty"`
	TimeSource      string          `json:"timeSource,omitempty"`
	Identification  json.RawMessage `json:"identification,omitempty"`
	SchemaVersion   int             `json:"schemaVersion"`
}

func NewEvent(eventType, sessionID string, timestamp time.Time) Event {
	var randomBytes [16]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		panic("crypto/rand failed")
	}
	return Event{
		EventID:       hex.EncodeToString(randomBytes[:]),
		EventType:     eventType,
		Timestamp:     timestamp.UTC(),
		SessionID:     sessionID,
		SchemaVersion: 1,
	}
}
