package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type Track struct {
	ID        string   `json:"subsonic_id"`
	Title     string   `json:"title"`
	Artist    string   `json:"artist"`
	Album     string   `json:"album"`
	Genre     string   `json:"genre"`
	Genres    []string `json:"genres"`
	BPM       float64  `json:"bpm"`
	Key       string   `json:"musicalKey"`
	Moods     []string `json:"moods"`
	Energy    string   `json:"energy"`
	Year      int      `json:"year"`
	Duration  float64  `json:"duration"`
	StartedAt string   `json:"startedAt"`
	EndedAt   string   `json:"endedAt"`
}

type ListenerCount struct {
	Current int
	Peak    int
}

func (l *ListenerCount) UnmarshalJSON(data []byte) error {
	var object struct {
		Current int `json:"current"`
		Count   int `json:"count"`
		Peak    int `json:"peak"`
	}
	if len(data) > 0 && data[0] == '{' {
		if err := json.Unmarshal(data, &object); err != nil {
			return fmt.Errorf("decode listener count: %w", err)
		}
		l.Current = object.Current
		if l.Current == 0 {
			l.Current = object.Count
		}
		l.Peak = object.Peak
		return nil
	}
	var number float64
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("decode listener count: %w", err)
	}
	l.Current = int(number)
	return nil
}

type Persona struct {
	Name    string `json:"name"`
	Tagline string `json:"tagline"`
	Station string `json:"station"`
}

type Show struct {
	Name    string `json:"name"`
	Persona struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"persona"`
}

type Stream struct {
	Mount         string  `json:"mount"`
	Format        string  `json:"format"`
	Bitrate       int     `json:"bitrate"`
	BufferSeconds float64 `json:"bufferSeconds"`
	OpusEnabled   bool    `json:"opusEnabled"`
	FLACEnabled   bool    `json:"flacEnabled"`
	AACEnabled    bool    `json:"aacEnabled"`
}

type NowPlayingResponse struct {
	NowPlaying   *Track        `json:"nowPlaying"`
	DJ           Persona       `json:"dj"`
	ActiveShow   *Show         `json:"activeShow"`
	Listeners    ListenerCount `json:"listeners"`
	StreamOnline bool          `json:"streamOnline"`
	Stream       Stream        `json:"stream"`
	Timezone     string        `json:"timezone"`
}

type QueueTrack struct {
	Track
	RequestedBy string `json:"requestedBy"`
	Source      string `json:"source"`
}

type DJLogEntry struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Text    string `json:"text"`
	Time    string `json:"t"`
}

type StateResponse struct {
	Current  *QueueTrack  `json:"current"`
	Upcoming []QueueTrack `json:"upcoming"`
	History  []QueueTrack `json:"history"`
	DJLog    []DJLogEntry `json:"djLog"`
}

type SessionTurn struct {
	Time string         `json:"t"`
	Role string         `json:"role"`
	Kind string         `json:"kind"`
	Text string         `json:"text"`
	Meta map[string]any `json:"meta"`
}

func (t SessionTurn) Timestamp() time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, t.Time)
	if !parsed.IsZero() {
		return parsed
	}
	if millis, err := strconv.ParseInt(t.Time, 10, 64); err == nil {
		return time.UnixMilli(millis)
	}
	return time.Time{}
}

type SessionResponse struct {
	Session  map[string]any `json:"session"`
	Messages []SessionTurn  `json:"messages"`
}

type LikeStatus struct {
	Enabled bool   `json:"enabled"`
	SongID  string `json:"songId"`
	Liked   bool   `json:"liked"`
	Count   int    `json:"count"`
}

type LikeResult struct {
	OK           bool   `json:"ok"`
	SongID       string `json:"songId"`
	Liked        bool   `json:"liked"`
	AlreadyLiked bool   `json:"alreadyLiked"`
	Count        int    `json:"count"`
	Error        string `json:"error"`
}

type Snapshot struct {
	NowPlaying   NowPlayingResponse
	State        StateResponse
	Session      SessionResponse
	Like         LikeStatus
	FetchedAt    time.Time
	Warnings     []string
	NowPlayingOK bool
	StateOK      bool
	SessionOK    bool
	LikeOK       bool
}
