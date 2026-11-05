# subwave-tune

A terminal radio for [SUB/WAVE](https://github.com/perminder-klair/subwave).
It plays the station through `mpv` while showing the current track, live DJ
and show, spoken Booth text, track-selection reasoning, queue, and history.

`subwave-tune` is an independent listener client. It does not import SUB/WAVE
code or need access to a station's filesystem or admin API.

## Requirements

- Go 1.24 or newer to build
- `mpv` available in `PATH` (`mpv` / `mpv.exe` on Windows; install via winget, scoop, or chocolatey)
- A reachable SUB/WAVE station
- On Windows, a VT-capable console such as Windows Terminal for the TUI

## Build and run

Linux / macOS:

```bash
go build -o bin/subwave-tune ./cmd/subwave-tune
./bin/subwave-tune https://radio.example.com
```

Windows (PowerShell):

```powershell
go build -o bin\subwave-tune.exe .\cmd\subwave-tune
.\bin\subwave-tune.exe https://radio.example.com
```

For a local default SUB/WAVE installation:

```bash
./bin/subwave-tune
```

```powershell
.\bin\subwave-tune.exe
```

The default station URL is `http://localhost:7700`. A positional URL,
`--url`, or `SUBWAVE_TUNE_URL` overrides it.

## Controls

| Key | Action |
| --- | --- |
| `Space` | Connect or disconnect playback |
| `l` | Like the audible track |
| `+` / `-` | Raise or lower volume |
| `m` | Mute |
| `r` | Drop the current stream and reconnect at the live edge |
| `q` | Quit |

Stopping playback deliberately terminates `mpv`. Starting again makes a fresh
stream connection instead of resuming stale buffered radio.

## Configuration

The optional configuration file lives at:

- Linux / macOS: `~/.config/subwave-tune/config.json`
- Windows: `%AppData%\subwave-tune\config.json`

```json
{
  "url": "https://radio.example.com",
  "volume": 75
}
```

Use `--config /path/to/config.json` (or a Windows path) to select another
file. Environment variables override the file:

- `SUBWAVE_TUNE_URL`
- `SUBWAVE_TUNE_PASSWORD`

For a station with Icecast listener authentication, put `password` in the
configuration file or export `SUBWAVE_TUNE_PASSWORD`. On Linux / macOS, prefer
a mode-`0600` file; on Windows, keep the file under your user profile so
normal NTFS ACLs limit access. There is deliberately no password command-line
flag, so it cannot leak into shell history or the process list.

Use `--no-play` to inspect the station without starting audio.

## How it works

The client polls SUB/WAVE's public listener API:

- `/api/now-playing` for the track, DJ, show, listeners and stream descriptor
- `/api/state` for the queue and recent history
- `/api/session` for spoken links and DJ selection reasoning
- `/api/like` for listener-specific like state and likes

Audio is played directly from the advertised Icecast mount. Displayed track
and Booth changes are shifted by the station's advertised listener-buffer
depth, keeping the screen aligned with what reaches the speakers.

## Development

```bash
go test ./...
go vet ./...
```

This project uses Bubble Tea v2 and Lip Gloss v2.
