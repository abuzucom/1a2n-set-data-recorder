package prodjlink

import (
	"fmt"
	"sort"
	"sync"
	"time"

	prolink "go.evanpurkhiser.com/prolink"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
)

type Client struct {
	mu         sync.RWMutex
	decks      map[string]model.DeckState
	devices    map[prolink.DeviceID]model.DeviceState
	lastTracks map[string]uint32
	remoteDB   *prolink.RemoteDB
	handler    func(model.DeckState)
}

func (c *Client) SetHandler(handler func(model.DeckState)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handler = handler
}

func Connect(configureWait time.Duration) (*Client, error) {
	network, err := prolink.Connect()
	if err != nil {
		return nil, err
	}
	if err := network.AutoConfigure(configureWait); err != nil {
		return nil, err
	}
	client := &Client{
		decks:      make(map[string]model.DeckState),
		devices:    make(map[prolink.DeviceID]model.DeviceState),
		lastTracks: make(map[string]uint32),
		remoteDB:   network.RemoteDB(),
	}
	manager := network.DeviceManager()
	manager.OnDeviceAdded("cdj-session-agent", prolink.DeviceListenerFunc(client.addDevice))
	manager.OnDeviceRemoved("cdj-session-agent", prolink.DeviceListenerFunc(client.removeDevice))
	network.CDJStatusMonitor().AddStatusHandler(prolink.StatusHandlerFunc(client.updateStatus))
	return client, nil
}

func (c *Client) Devices() []model.DeviceState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	devices := make([]model.DeviceState, 0, len(c.devices))
	for _, device := range c.devices {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool {
		return devices[i].DeviceID < devices[j].DeviceID
	})
	return devices
}

func (c *Client) Decks() []model.DeckState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	decks := make([]model.DeckState, 0, len(c.decks))
	for _, deck := range c.decks {
		decks = append(decks, deck)
	}
	sort.Slice(decks, func(i, j int) bool {
		return decks[i].Player < decks[j].Player
	})
	return decks
}

func (c *Client) updateStatus(status *prolink.CDJStatus) {
	deckID := fmt.Sprintf("player-%d", status.PlayerID)
	deck := model.DeckState{
		DeckID: deckID, Player: int(status.PlayerID), TrackID: status.TrackID,
		BPM: float64(status.TrackBPM), SourceSlot: status.TrackSlot.String(),
		Pitch: float64(status.EffectivePitch), Beat: status.Beat, BeatInBar: status.BeatInMeasure,
		PlayState: status.PlayState.String(), IsOnAir: status.IsOnAir, IsSync: status.IsSync,
		IsMaster: status.IsMaster, UpdatedAt: time.Now().UTC(),
	}
	c.mu.Lock()
	if device, ok := c.devices[status.PlayerID]; ok {
		deck.Model = device.Name
	}
	c.decks[deckID] = deck
	lookupMetadata := status.TrackID != 0 && c.lastTracks[deckID] != status.TrackID
	c.lastTracks[deckID] = status.TrackID
	handler := c.handler
	c.mu.Unlock()
	if handler != nil {
		handler(deck)
	}
	if lookupMetadata {
		c.enrichMetadata(deckID, status.TrackID, status.TrackKey())
	}
}

func (c *Client) enrichMetadata(deckID string, trackID uint32, key *prolink.TrackKey) {
	if key == nil || !c.remoteDB.IsLinked(key.DeviceID) {
		return
	}
	track, err := c.remoteDB.GetTrack(key)
	if err != nil {
		return
	}
	c.mu.Lock()
	deck, ok := c.decks[deckID]
	if !ok || deck.TrackID != trackID {
		c.mu.Unlock()
		return
	}
	deck.Title = track.Title
	deck.Artist = track.Artist
	deck.Album = track.Album
	deck.Key = track.Key
	c.decks[deckID] = deck
	handler := c.handler
	c.mu.Unlock()
	if handler != nil {
		handler(deck)
	}
}

func (c *Client) addDevice(device *prolink.Device) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.devices[device.ID] = model.DeviceState{DeviceID: int(device.ID), Name: device.Name, Type: device.Type.String(), Address: device.IP.String(), LastActive: device.LastActive.UTC()}
}

func (c *Client) removeDevice(device *prolink.Device) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.devices, device.ID)
}
