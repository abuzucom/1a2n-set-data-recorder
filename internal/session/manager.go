package session

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
)

const sampleDuration = 30 * time.Second

type activePlay struct {
	playID    string
	startedAt time.Time
	state     model.DeckState
}

type Manager struct {
	mu             sync.Mutex
	session        model.Session
	plays          map[string]activePlay
	silenceStarted time.Time
	silenceAfter   time.Duration
}

func NewManager(silenceAfter time.Duration) *Manager {
	return &Manager{plays: make(map[string]activePlay), silenceAfter: silenceAfter}
}

func (m *Manager) Start(name string, startedAt time.Time) (model.Session, error) {
	name = strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(name))
	if name == "" || len(name) > 100 {
		return model.Session{}, errors.New("session name must contain 1 to 100 characters")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session.ID != "" && m.session.EndedAt.IsZero() {
		return model.Session{}, errors.New("a session is already active")
	}
	m.session = model.Session{ID: newID(), Name: name, StartedAt: startedAt.UTC()}
	m.plays = make(map[string]activePlay)
	m.silenceStarted = time.Time{}
	return m.session, nil
}

func (m *Manager) ActiveSession() model.Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.session
}

func (m *Manager) End(endedAt time.Time) ([]model.Event, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session.ID == "" || !m.session.EndedAt.IsZero() {
		return nil, errors.New("no active session")
	}
	events := m.closeAllPlays(endedAt.UTC())
	if !m.silenceStarted.IsZero() {
		events = append(events, m.silenceEvent(endedAt.UTC()))
	}
	m.session.EndedAt = endedAt.UTC()
	events = append(events, m.event(model.EventSessionEnd, endedAt.UTC()))
	return events, nil
}

func (m *Manager) UpdateDeck(state model.DeckState) ([]model.Event, error) {
	if state.DeckID == "" || state.Player < 1 {
		return nil, errors.New("deck ID and player number are required")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session.ID == "" || !m.session.EndedAt.IsZero() {
		return nil, errors.New("no active session")
	}
	state.UpdatedAt = state.UpdatedAt.UTC()
	current, exists := m.plays[state.DeckID]
	events := make([]model.Event, 0, 3)
	if exists && (!state.IsOnAir || current.state.TrackID != state.TrackID) {
		events = append(events, m.trackOffEvent(current, state.UpdatedAt))
		delete(m.plays, state.DeckID)
	}
	if state.IsOnAir && (!exists || current.state.TrackID != state.TrackID) {
		play := activePlay{playID: newID(), startedAt: state.UpdatedAt, state: state}
		m.plays[state.DeckID] = play
		events = append(events, m.trackOnEvent(play))
	}
	if len(m.plays) == 0 && m.silenceStarted.IsZero() {
		m.silenceStarted = state.UpdatedAt
	}
	if len(m.plays) > 0 && !m.silenceStarted.IsZero() {
		events = append(events, m.silenceEvent(state.UpdatedAt))
		m.silenceStarted = time.Time{}
	}
	return filterEmpty(events), nil
}

func (m *Manager) closeAllPlays(endedAt time.Time) []model.Event {
	events := make([]model.Event, 0, len(m.plays))
	for deckID, play := range m.plays {
		events = append(events, m.trackOffEvent(play, endedAt))
		delete(m.plays, deckID)
	}
	return events
}

func (m *Manager) trackOnEvent(play activePlay) model.Event {
	event := m.event(model.EventTrackOnAir, play.startedAt)
	event.PlayID = play.playID
	event.DeckID = play.state.DeckID
	event.Player = play.state.Player
	event.TrackID = play.state.TrackID
	event.Title = play.state.Title
	event.Artist = play.state.Artist
	return event
}

func (m *Manager) trackOffEvent(play activePlay, endedAt time.Time) model.Event {
	duration := endedAt.Sub(play.startedAt)
	event := m.trackOnEvent(play)
	event.EventID = newID()
	event.EventType = model.EventTrackOffAir
	event.Timestamp = endedAt
	event.DurationSeconds = duration.Seconds()
	event.IsSampleLike = duration < sampleDuration
	return event
}

func (m *Manager) silenceEvent(endedAt time.Time) model.Event {
	duration := endedAt.Sub(m.silenceStarted)
	if duration < m.silenceAfter {
		return model.Event{}
	}
	event := m.event(model.EventSilencePeriod, endedAt)
	event.DurationSeconds = duration.Seconds()
	return event
}

func (m *Manager) event(eventType string, timestamp time.Time) model.Event {
	return model.Event{EventID: newID(), EventType: eventType, Timestamp: timestamp, SessionID: m.session.ID, SchemaVersion: 1}
}

func filterEmpty(events []model.Event) []model.Event {
	result := events[:0]
	for _, event := range events {
		if event.EventID != "" {
			result = append(result, event)
		}
	}
	return result
}

func newID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		panic("crypto/rand failed")
	}
	return hex.EncodeToString(bytes[:])
}
