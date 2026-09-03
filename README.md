# 1a2n-set-data-recorder

Local set data recorder for DJs using Pioneer CDJ equipment. The service watches
CDJs over the local Pro DJ Link network and builds a timestamped record of a DJ
set. A browser dashboard shows the current session, deck status, connected
devices, and event timeline.

The project records what happened during a set. It does not record audio. Use a
separate recorder for the mix, then link that audio file to the session.

## What It Records

With Pro DJ Link active during a session, the service records these details:

- Tracks that come on air and go off air.
- Player number and deck identity.
- Track ID, title, artist, album, key, BPM, pitch, and play state when CDJ data
  provides those values.
- Whether a deck is on air, master, or synchronized.
- Periods with no track on air that last at least 20 seconds.
- Tracks that played for less than 30 seconds as sample-like events.

Dashboard controls also record:

- Session name and start time.
- Session end time.
- DJ handoffs.
- Manual time-source changes.
- External recording start and stop markers.
- The audio file path, recording timestamps, and alignment offset.

The service writes durable JSON Lines files. Each line represents one event.
This format keeps the data portable for scripts, analysis tools, and future
interfaces.

## Before Starting

The current implementation requires:

- Go 1.27.0.
- A computer running the service.
- A modern web browser.
- A separate audio recorder for saved set audio.
- A wired or wireless local network for CDJ integration.

The service can run without CDJs for session controls and manual event entry.
Enable Pro DJ Link only when the computer shares the CDJ network.

## Quick Start

1. Install Go 1.27.0.
2. Create an opaque access token for this local service.
3. Set the token in the process environment as `CDJ_SESSION_API_TOKEN`.
4. Start the service:

   ```text
   go run ./cmd/cdj-session-agent
   ```

5. Open `http://127.0.0.1:8080` in a browser.
6. Enter the same token in the dashboard access token field.
7. Enter a name for the set and select `Start`.
8. Select `End` when the set finishes.

The default address accepts connections only from the same computer. Keep this
setting unless another device must open the dashboard. Set
`--listen-address` explicitly before exposing the service on a network.

## CDJ Setup

1. Connect the computer and CDJs to the same local network.
2. Confirm that the CDJs can see each other through Pro DJ Link.
3. Start the service with the Pro DJ Link option:

   ```text
   go run ./cmd/cdj-session-agent --enable-pro-dj-link
   ```

4. Open the dashboard and check the `Network devices` section.
5. Start a session before playing tracks.

The Pro DJ Link client announces a virtual player on the network. Do not run
the service with a player number already in use. A Pro DJ Link connection can
fail when the computer uses the wrong network adapter, a firewall blocks local
traffic, or the devices are on different networks.

Track metadata may appear shortly after deck status. The service receives deck
status first and then attempts optional metadata enrichment. The session keeps
recording when that enrichment is unavailable.

## Recording Audio Separately

The dashboard does not control an audio recorder. Start and stop the external
recorder through its own controls, then select `Recording started` and
`Recording stopped` in the dashboard to add matching timeline events.

After the recording exists, add its metadata:

1. Place the audio file under `data/recordings`.
2. Enter its relative path, such as `friday-night/set.wav`.
3. Enter recording start and stop times when those times differ from the session
   times.
4. Enter an alignment offset in seconds when the audio and CDJ clocks differ.
5. Select `Save metadata`.

Recording paths must stay relative to `data/recordings`. The server rejects
absolute paths and paths that leave that directory. Timestamps use UTC
RFC3339Nano values.

## Where Data Goes

The default data directory is `data`:

```text
data/
  logs/
    session-<session-id>.jsonl
    session-<session-id>.identifications.jsonl
  recordings/
    <audio files supplied by the DJ>
```

Use `--data-dir` to choose another local root:

```text
go run ./cmd/cdj-session-agent --data-dir /path/to/set-data
```

Back up the selected data directory after important sets. The service stores
session history locally and does not provide cloud storage or automatic sync.

## Access And Safety

The service requires `CDJ_SESSION_API_TOKEN` before it starts. Enter that same
token in the dashboard before using controls that change session data. Keep the
token private. Anyone who has the token can use the mutation endpoints.

The default bind address is `127.0.0.1:8080`. Use a network bind address only
when the dashboard must be reachable from another device. Use a trusted local
network and avoid exposing the service directly to the public internet.

## Audio Identification

The optional AcoustID command analyzes sections of the external recording. It
uses Chromaprint `fpcalc` and the AcoustID service.

1. Install Chromaprint `fpcalc` 1.6.1 outside this repository.
2. Set `ACOUSTID_CLIENT_KEY` in the process environment.
3. Add valid recording metadata to the session.
4. Run:

   ```text
   go run ./cmd/acoustid-analyze <session-id>
   ```

The command reads `data/logs/session-<session-id>.jsonl` and the audio file
under `data/recordings`. It appends results to
`data/logs/session-<session-id>.identifications.jsonl`.

Use alternate locations when required:

```text
go run ./cmd/acoustid-analyze \
  --logs-root /path/to/set-data/logs \
  --recordings-root /path/to/set-data/recordings \
  --fpcalc /path/to/fpcalc <session-id>
```

## HTTP API

The API supports dashboards and custom integrations. `GET` endpoints do not
require the access token. Every `POST` endpoint requires
`Authorization: Bearer <token>`.

- `GET /status` returns the active session, current deck status, and devices.
- `GET /ws` opens a WebSocket for live status and event updates.
- `GET /sessions` lists recorded sessions.
- `GET /sessions/:id/events` returns the event history for one session.
- `POST /sessions` starts a session with `{"name":"..."}`.
- `POST /sessions/:id/end` ends the active session.
- `POST /sessions/:id/dj-handoff` records a DJ handoff.
- `POST /sessions/:id/time-source` records a time-source change.
- `POST /sessions/:id/recording/start` records an external recording start.
- `POST /sessions/:id/recording/stop` records an external recording stop.
- `POST /sessions/:id/recording/metadata` accepts `audioFilePath`, optional
  `recordingStartTimestamp`, `recordingStopTimestamp`, and `offsetSeconds`.

The Go server serves the browser interface from `ui/dist`. Implementers can
replace it with another client that uses the same endpoints and WebSocket.

## Current Status

Implemented:

- Local Go web server with a browser dashboard.
- Session lifecycle and JSONL event history.
- Pro DJ Link device discovery and deck status.
- DJ handoff, recording markers, and recording metadata.
- Optional AcoustID analysis command.

Not yet validated:

- Hardware behavior against physical CDJs and mixers.
- AcoustID behavior against a complete production set.

The repository does not include a packaged installer, hosted service, audio
recorder, or cloud backup system.

## Development

The Go module is `github.com/abuzucom/1a2n-set-data-recorder`. Main entry points
are:

- `cmd/cdj-session-agent` for the local web server.
- `cmd/acoustid-analyze` for optional audio identification.

Run the repository checks with:

```text
make lint
make test
```

`AGENTS.md` defines repository contribution and security policy. The policy
requires explicit authorization for dependency changes and destructive actions.
