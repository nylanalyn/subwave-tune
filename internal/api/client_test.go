package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/now-playing":
			fmt.Fprint(w, `{"nowPlaying":{"title":"Roads","artist":"Portishead"},"listeners":2,"streamOnline":true,"stream":{"mount":"/stream.mp3","bufferSeconds":22}}`)
		case "/api/state":
			fmt.Fprint(w, `{"upcoming":[{"title":"Glory Box"}],"history":[],"djLog":[]}`)
		case "/api/session":
			fmt.Fprint(w, `{"session":{"id":"one"},"messages":[{"role":"segment","text":"Stay with us."}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := client.FetchSnapshot(context.Background())
	if len(snapshot.Warnings) != 0 {
		t.Fatalf("warnings: %v", snapshot.Warnings)
	}
	if !snapshot.NowPlayingOK || !snapshot.StateOK || !snapshot.SessionOK {
		t.Fatalf("successful endpoints not marked valid: %+v", snapshot)
	}
	if snapshot.NowPlaying.NowPlaying.Title != "Roads" || snapshot.NowPlaying.Listeners.Current != 2 {
		t.Fatalf("unexpected now playing: %+v", snapshot.NowPlaying)
	}
	if snapshot.State.Upcoming[0].Title != "Glory Box" {
		t.Fatalf("unexpected state: %+v", snapshot.State)
	}
}

func TestZeroListenerObject(t *testing.T) {
	var count ListenerCount
	if err := count.UnmarshalJSON([]byte(`{"current":0,"peak":0}`)); err != nil {
		t.Fatal(err)
	}
	if count.Current != 0 {
		t.Fatalf("current = %d", count.Current)
	}
}

func TestStreamURLWithPassword(t *testing.T) {
	client, err := New("https://radio.example/api", "a b&c")
	if err != nil {
		t.Fatal(err)
	}
	got := client.StreamURL("/stream.mp3")
	if !strings.Contains(got, "auth=a+b%26c") {
		t.Fatalf("StreamURL() = %q", got)
	}
}
