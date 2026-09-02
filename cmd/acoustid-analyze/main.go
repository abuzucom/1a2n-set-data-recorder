package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/abuzucom/1a2n-set-data-recorder/internal/events"
	"github.com/abuzucom/1a2n-set-data-recorder/internal/model"
)

const maxAcoustIDResponseBytes = 1024 * 1024

type fingerprint struct {
	Duration    int    `json:"duration"`
	Fingerprint string `json:"fingerprint"`
}

func main() {
	logsRoot := flag.String("logs-root", "data/logs", "session log directory")
	recordingsRoot := flag.String("recordings-root", "data/recordings", "recording directory")
	fpcalcPath := flag.String("fpcalc", "fpcalc", "fpcalc executable")
	flag.Parse()
	if flag.NArg() != 1 || os.Getenv("ACOUSTID_CLIENT_KEY") == "" {
		fmt.Fprintln(os.Stderr, "provide a session ID and ACOUSTID_CLIENT_KEY")
		os.Exit(2)
	}
	sessionID := flag.Arg(0)
	eventList, err := events.ReadEvents(*logsRoot, sessionID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to read session events:", err)
		os.Exit(1)
	}
	audioPath, recordingStart, offset, ok := recording(eventList, *recordingsRoot)
	if !ok {
		fmt.Fprintln(os.Stderr, "session has no valid recording metadata")
		os.Exit(1)
	}
	outputPath := filepath.Join(*logsRoot, "session-"+sessionID+".identifications.jsonl")
	output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to open output:", err)
		os.Exit(1)
	}
	defer output.Close()
	starts := map[string]model.Event{}
	for _, event := range eventList {
		if event.EventType == model.EventTrackOnAir {
			starts[event.PlayID] = event
			continue
		}
		if event.EventType != model.EventTrackOffAir || event.IsSampleLike {
			continue
		}
		start, found := starts[event.PlayID]
		if !found {
			continue
		}
		position := start.Timestamp.Sub(recordingStart).Seconds() - offset
		if position < 0 || event.DurationSeconds <= 0 {
			continue
		}
		result, err := identify(audioPath, *fpcalcPath, position, event.DurationSeconds)
		if err != nil {
			fmt.Fprintln(os.Stderr, "identification failed:", err)
			continue
		}
		identified := model.NewEvent(model.EventTrackIdentification, sessionID, time.Now())
		identified.PlayID, identified.TrackID, identified.Title, identified.Artist = event.PlayID, event.TrackID, event.Title, event.Artist
		identified.DurationSeconds = event.DurationSeconds
		identified.Identification = result
		data, _ := json.Marshal(identified)
		if _, err := output.Write(append(data, '\n')); err != nil {
			fmt.Fprintln(os.Stderr, "failed to write output:", err)
			os.Exit(1)
		}
	}
}

func recording(eventList []model.Event, root string) (string, time.Time, float64, bool) {
	for _, event := range eventList {
		if event.EventType != model.EventRecordingMetadata {
			continue
		}
		path := filepath.Clean(event.AudioPath)
		if filepath.IsAbs(path) || path == "." || strings.HasPrefix(path, "..") {
			return "", time.Time{}, 0, false
		}
		recordingStart := event.RecordingStart
		if recordingStart.IsZero() {
			recordingStart = event.Timestamp
		}
		return filepath.Join(root, path), recordingStart, event.OffsetSeconds, true
	}
	return "", time.Time{}, 0, false
}

func identify(audioPath, fpcalcPath string, position, duration float64) (json.RawMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	args := []string{"-json", "-offset", strconv.Itoa(int(position)), "-length", strconv.Itoa(int(duration)), audioPath}
	output, err := exec.CommandContext(ctx, fpcalcPath, args...).Output()
	if err != nil {
		return nil, err
	}
	var value fingerprint
	if err := json.Unmarshal(output, &value); err != nil {
		return nil, err
	}
	form := url.Values{"client": {os.Getenv("ACOUSTID_CLIENT_KEY")}, "duration": {strconv.Itoa(value.Duration)}, "fingerprint": {value.Fingerprint}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.acoustid.org/v2/lookup", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AcoustID returned %s", response.Status)
	}
	var result json.RawMessage
	if err := json.NewDecoder(io.LimitReader(response.Body, maxAcoustIDResponseBytes)).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}
