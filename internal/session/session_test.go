package session

import (
	"testing"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
)

func TestManagerEmitsTrackEvents(t *testing.T) {
	manager := NewManager(20 * time.Second)
	start := time.Date(2026, time.September, 2, 12, 0, 0, 0, time.UTC)

	if _, err := manager.Start("day-set", start); err != nil {
		t.Fatalf("Start returned an error: %v", err)
	}

	events, err := manager.UpdateDeck(model.DeckState{
		DeckID:    "player-1",
		Player:    1,
		TrackID:   42,
		Title:     "Track",
		IsOnAir:   true,
		UpdatedAt: start,
	})
	if err != nil {
		t.Fatalf("UpdateDeck returned an error: %v", err)
	}
	if len(events) != 1 || events[0].EventType != model.EventTrackOnAir {
		t.Fatalf("UpdateDeck returned %#v, want one track-on-air event", events)
	}

	events, err = manager.UpdateDeck(model.DeckState{
		DeckID:    "player-1",
		Player:    1,
		TrackID:   42,
		Title:     "Track",
		UpdatedAt: start.Add(31 * time.Second),
	})
	if err != nil {
		t.Fatalf("UpdateDeck returned an error: %v", err)
	}
	if len(events) != 1 || events[0].EventType != model.EventTrackOffAir {
		t.Fatalf("UpdateDeck returned %#v, want one track-off-air event", events)
	}
	if events[0].IsSampleLike {
		t.Fatal("track-off-air event marked a 31-second play as a sample")
	}
}
