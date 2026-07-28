package ui

import (
	"strings"
	"testing"
	"time"

	"subwave-tune/internal/api"
)

func TestDisplayTrackUsesPreviousDuringListenerLag(t *testing.T) {
	now := time.Now()
	model := Model{
		snapshot: api.Snapshot{
			NowPlaying: api.NowPlayingResponse{
				NowPlaying: &api.Track{ID: "new", Title: "New"},
				Stream:     api.Stream{BufferSeconds: 22},
			},
			State: api.StateResponse{
				Current: &api.QueueTrack{Track: api.Track{StartedAt: now.Add(-5 * time.Second).Format(time.RFC3339Nano)}},
				History: []api.QueueTrack{{Track: api.Track{ID: "old", Title: "Old", StartedAt: now.Add(-3 * time.Minute).Format(time.RFC3339Nano)}}},
			},
		},
	}

	track, _, _ := model.displayTrack()
	if track == nil || track.ID != "old" {
		t.Fatalf("displayed track = %+v", track)
	}
}

func TestWrap(t *testing.T) {
	got := wrap("one two three four", 9)
	if !strings.Contains(got, "\n") {
		t.Fatalf("expected wrapped text, got %q", got)
	}
}

func TestLikeLabelRequiresDisplayedTrackMatch(t *testing.T) {
	model := Model{
		snapshot: api.Snapshot{
			LikeOK: true,
			Like:   api.LikeStatus{Enabled: true, SongID: "live", Count: 2},
		},
	}
	if got := model.likeLabel(&api.Track{ID: "buffered"}); !strings.Contains(got, "syncing") {
		t.Fatalf("likeLabel() = %q", got)
	}
	if got := model.likeLabel(&api.Track{ID: "live"}); !strings.Contains(got, "♡ 2") {
		t.Fatalf("likeLabel() = %q", got)
	}
}
