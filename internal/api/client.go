package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

type Client struct {
	base       *url.URL
	httpClient *http.Client
	password   string
}

func New(baseURL, password string) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse station URL: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("station URL must use http or https")
	}
	parsed.Path = strings.TrimSuffix(parsed.Path, "/api")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return &Client{
		base:       parsed,
		password:   password,
		httpClient: &http.Client{Timeout: 6 * time.Second},
	}, nil
}

func (c *Client) StationURL() string {
	return strings.TrimRight(c.base.String(), "/")
}

func (c *Client) StreamURL(mount string) string {
	if mount == "" {
		mount = "/stream.mp3"
	}
	u := *c.base
	u.Path = path.Join(c.base.Path, mount)
	if c.password != "" {
		query := u.Query()
		query.Set("auth", c.password)
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func (c *Client) FetchSnapshot(ctx context.Context) Snapshot {
	snapshot := Snapshot{FetchedAt: time.Now()}
	type result struct {
		name string
		err  error
	}
	results := make(chan result, 3)
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		err := c.get(ctx, "/now-playing", &snapshot.NowPlaying)
		snapshot.NowPlayingOK = err == nil
		results <- result{"now-playing", err}
	}()
	go func() {
		defer wg.Done()
		err := c.get(ctx, "/state", &snapshot.State)
		snapshot.StateOK = err == nil
		results <- result{"state", err}
	}()
	go func() {
		defer wg.Done()
		err := c.get(ctx, "/session", &snapshot.Session)
		snapshot.SessionOK = err == nil
		results <- result{"session", err}
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			snapshot.Warnings = append(snapshot.Warnings, result.name+": "+result.err.Error())
		}
	}
	return snapshot
}

func (c *Client) get(ctx context.Context, endpoint string, target any) error {
	u := *c.base
	u.Path = path.Join(c.base.Path, "/api", endpoint)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "subwave-tune/dev")
	response, err := c.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("%s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
