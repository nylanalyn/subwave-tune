package ui

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"subwave-tune/internal/api"
	"subwave-tune/internal/player"
)

const pollEvery = 5

var (
	accent     = lipgloss.Color("#ff5f87")
	secondary  = lipgloss.Color("#5fd7ff")
	dim        = lipgloss.Color("#7f849c")
	good       = lipgloss.Color("#5fff87")
	bad        = lipgloss.Color("#ff5f5f")
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	labelStyle = lipgloss.NewStyle().Bold(true).Foreground(secondary)
	dimStyle   = lipgloss.NewStyle().Foreground(dim)
)

type snapshotMsg api.Snapshot
type clockMsg time.Time
type playerMsg struct {
	running bool
	err     error
}

type Model struct {
	client      *api.Client
	player      *player.MPV
	snapshot    api.Snapshot
	width       int
	height      int
	ticks       int
	lastError   string
	streamURL   string
	autoPlay    bool
	initialized bool
}

func New(client *api.Client, mpv *player.MPV, autoPlay bool) Model {
	return Model{
		client:    client,
		player:    mpv,
		streamURL: client.StreamURL("/stream.mp3"),
		autoPlay:  autoPlay,
	}
}

func (m Model) Init() tea.Cmd {
	commands := []tea.Cmd{m.fetch(), tick()}
	if m.autoPlay {
		commands = append(commands, m.startPlayer())
	}
	return tea.Batch(commands...)
}

func (m Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = message.Width, message.Height
	case tea.KeyPressMsg:
		switch message.String() {
		case "q", "ctrl+c":
			m.player.Stop()
			return m, tea.Quit
		case "space":
			return m, m.togglePlayer()
		case "+", "=":
			return m, m.adjustVolume(5)
		case "-", "_":
			return m, m.adjustVolume(-5)
		case "m":
			return m, m.toggleMute()
		case "r":
			m.player.Stop()
			return m, m.startPlayer()
		}
	case snapshotMsg:
		incoming := api.Snapshot(message)
		if !incoming.NowPlayingOK && m.initialized {
			incoming.NowPlaying = m.snapshot.NowPlaying
		}
		if !incoming.StateOK && m.initialized {
			incoming.State = m.snapshot.State
		}
		if !incoming.SessionOK && m.initialized {
			incoming.Session = m.snapshot.Session
		}
		m.snapshot = incoming
		m.initialized = true
		if mount := m.snapshot.NowPlaying.Stream.Mount; mount != "" {
			m.streamURL = m.client.StreamURL(mount)
		}
		if len(m.snapshot.Warnings) > 0 {
			m.lastError = strings.Join(m.snapshot.Warnings, " · ")
		} else {
			m.lastError = ""
		}
	case playerMsg:
		if message.err != nil {
			m.lastError = message.err.Error()
		}
	case clockMsg:
		m.ticks++
		commands := []tea.Cmd{tick()}
		if m.ticks%pollEvery == 0 {
			commands = append(commands, m.fetch())
		}
		return m, tea.Batch(commands...)
	}
	return m, nil
}

func (m Model) View() tea.View {
	width := m.width
	if width < 48 {
		width = 48
	}
	contentWidth := width - 2
	if !m.initialized {
		view := tea.NewView("\n  " + titleStyle.Render("SUB/WAVE TUNE") + "\n\n  Tuning the dial…")
		view.AltScreen = true
		view.WindowTitle = "SUB/WAVE TUNE"
		return view
	}

	header := m.renderHeader(contentWidth)
	now := m.renderNowPlaying(contentWidth)
	booth := m.renderBooth(contentWidth)
	var middle string
	if contentWidth >= 70 {
		left := (contentWidth - 1) / 2
		right := contentWidth - left - 1
		middle = lipgloss.JoinHorizontal(lipgloss.Top, m.renderQueue(left), " ", m.renderHistory(right))
	} else {
		middle = m.renderQueue(contentWidth) + "\n" + m.renderHistory(contentWidth)
	}

	parts := []string{header, now, booth, middle}
	if m.lastError != "" {
		parts = append(parts, lipgloss.NewStyle().Foreground(bad).Width(contentWidth).Render("⚠ "+truncate(m.lastError, contentWidth-3)))
	}
	parts = append(parts, m.renderFooter(contentWidth))
	content := strings.Join(parts, "\n")
	if m.height > 0 {
		content = lipgloss.NewStyle().MaxHeight(m.height).Render(content)
	}
	view := tea.NewView(content)
	view.AltScreen = true
	view.WindowTitle = "SUB/WAVE TUNE"
	return view
}

func (m Model) fetch() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		return snapshotMsg(m.client.FetchSnapshot(ctx))
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(now time.Time) tea.Msg { return clockMsg(now) })
}

func (m Model) startPlayer() tea.Cmd {
	return func() tea.Msg {
		err := m.player.Start(m.streamURL)
		return playerMsg{running: err == nil, err: err}
	}
}

func (m Model) togglePlayer() tea.Cmd {
	return func() tea.Msg {
		running, err := m.player.Toggle(m.streamURL)
		return playerMsg{running: running, err: err}
	}
}

func (m Model) adjustVolume(delta int) tea.Cmd {
	return func() tea.Msg {
		err := m.player.AdjustVolume(delta)
		return playerMsg{running: m.player.Status().Running, err: err}
	}
}

func (m Model) toggleMute() tea.Cmd {
	return func() tea.Msg {
		err := m.player.ToggleMute()
		return playerMsg{running: m.player.Status().Running, err: err}
	}
}

func (m Model) renderHeader(width int) string {
	np := m.snapshot.NowPlaying
	status := lipgloss.NewStyle().Foreground(bad).Render("OFF AIR")
	if np.StreamOnline {
		status = lipgloss.NewStyle().Bold(true).Foreground(good).Render("● ON AIR")
	}
	station := np.DJ.Station
	if station == "" {
		station = "SUB/WAVE"
	}
	left := titleStyle.Render(strings.ToUpper(station) + "  TUNE")
	dj := np.DJ.Name
	if dj == "" {
		dj = "the booth"
	}
	show := ""
	if np.ActiveShow != nil && np.ActiveShow.Name != "" {
		show = " · " + np.ActiveShow.Name
	}
	right := fmt.Sprintf("%s  %s%s  ◉ %d", status, dj, show, np.Listeners.Current)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left + "\n" + right
	}
	return left + strings.Repeat(" ", gap) + right
}

func (m Model) renderNowPlaying(width int) string {
	track, startedAt, duration := m.displayTrack()
	if track == nil {
		return panel("NOW PLAYING", dimStyle.Render("Waiting for the transmitter…"), width)
	}
	title := track.Title
	if title == "" {
		title = "Unknown title"
	}
	body := titleStyle.Copy().Foreground(lipgloss.Color("#ffffff")).Render(title)
	body += "\n" + labelStyle.Render(track.Artist)
	if track.Album != "" {
		body += dimStyle.Render("  —  " + track.Album)
	}

	listenerNow := time.Now().Add(-m.listenerLag())
	elapsed := listenerNow.Sub(startedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if duration > 0 {
		barWidth := width - 28
		if barWidth < 12 {
			barWidth = 12
		}
		body += "\n" + progress(elapsed, time.Duration(duration*float64(time.Second)), barWidth)
		body += fmt.Sprintf("  %s / %s", clock(elapsed), clock(time.Duration(duration*float64(time.Second))))
	}
	meta := m.trackMeta(track)
	if meta != "" {
		body += "\n" + dimStyle.Render(meta)
	}
	return panel("NOW PLAYING", body, width)
}

func (m Model) renderBooth(width int) string {
	cutoff := time.Now().Add(-m.listenerLag())
	track, _, _ := m.displayTrack()
	var spoken, reason string
	for i := len(m.snapshot.Session.Messages) - 1; i >= 0; i-- {
		turn := m.snapshot.Session.Messages[i]
		if timestamp := turn.Timestamp(); !timestamp.IsZero() && timestamp.After(cutoff) {
			continue
		}
		switch turn.Role {
		case "segment":
			if spoken == "" {
				spoken = turn.Text
			}
		case "dj":
			if reason != "" {
				continue
			}
			trackID, _ := turn.Meta["trackId"].(string)
			if track == nil || trackID == "" || trackID == track.ID {
				reason = turn.Text
			}
		}
		if spoken != "" && reason != "" {
			break
		}
	}
	lines := []string{}
	if spoken != "" {
		prefix := m.snapshot.NowPlaying.DJ.Name + ": "
		if m.snapshot.NowPlaying.DJ.Name == "" {
			prefix = "ON AIR: "
		}
		if m.height > 0 && m.height < 28 {
			lines = append(lines, labelStyle.Render(prefix)+truncate(spoken, width-len(prefix)-6))
		} else {
			lines = append(lines, labelStyle.Render(prefix)+wrapLimit(spoken, width-len(prefix)-6, 2))
		}
	}
	if reason != "" {
		if m.height > 0 && m.height < 28 {
			lines = append(lines, dimStyle.Render("WHY  ")+truncate(reason, width-11))
		} else {
			lines = append(lines, dimStyle.Render("WHY THIS TRACK  ")+wrapLimit(reason, width-18, 2))
		}
	}
	if len(lines) == 0 {
		lines = append(lines, dimStyle.Render("The booth is quiet."))
	}
	return panel("DJ BOOTH", strings.Join(lines, "\n"), width)
}

func (m Model) renderQueue(width int) string {
	lines := make([]string, 0, 4)
	for i, track := range m.snapshot.State.Upcoming {
		limit := 4
		if m.height > 0 && m.height < 28 {
			limit = 3
		}
		if i == limit {
			break
		}
		lines = append(lines, fmt.Sprintf("%s %s", dimStyle.Render(fmt.Sprintf("%d.", i+1)), trackLine(track.Track, width-8)))
	}
	if len(lines) == 0 {
		lines = append(lines, dimStyle.Render("The next record is still being chosen."))
	}
	return panel("UP NEXT", strings.Join(lines, "\n"), width)
}

func (m Model) renderHistory(width int) string {
	lines := make([]string, 0, 4)
	for i, track := range m.snapshot.State.History {
		limit := 4
		if m.height > 0 && m.height < 28 {
			limit = 3
		}
		if i == limit {
			break
		}
		lines = append(lines, dimStyle.Render("‹")+" "+trackLine(track.Track, width-8))
	}
	if len(lines) == 0 {
		lines = append(lines, dimStyle.Render("No recent spins."))
	}
	return panel("RECENTLY", strings.Join(lines, "\n"), width)
}

func (m Model) renderFooter(width int) string {
	status := m.player.Status()
	play := "stopped"
	if status.Running {
		play = "playing"
	}
	if status.Muted {
		play += " · muted"
	}
	left := fmt.Sprintf("SPACE %s  +/− volume  M mute  R reconnect  Q quit", play)
	right := fmt.Sprintf("%d%% · %s", status.Volume, m.snapshot.NowPlaying.Stream.Format)
	gap := width - len(left) - len(right)
	if gap < 1 {
		return dimStyle.Render(left)
	}
	return dimStyle.Render(left + strings.Repeat(" ", gap) + right)
}

func (m Model) displayTrack() (*api.Track, time.Time, float64) {
	np := m.snapshot.NowPlaying.NowPlaying
	if np == nil {
		return nil, time.Time{}, 0
	}
	startedAt := parseTime(m.snapshot.State.Current)
	listenerNow := time.Now().Add(-m.listenerLag())
	if !startedAt.IsZero() && startedAt.After(listenerNow) && len(m.snapshot.State.History) > 0 {
		previous := m.snapshot.State.History[0].Track
		return &previous, parseRFC3339(previous.StartedAt), previous.Duration
	}
	return np, startedAt, np.Duration
}

func (m Model) listenerLag() time.Duration {
	seconds := m.snapshot.NowPlaying.Stream.BufferSeconds
	if seconds < 0 {
		seconds = 0
	}
	return time.Duration(seconds * float64(time.Second))
}

func (m Model) trackMeta(track *api.Track) string {
	parts := []string{}
	if track.Genre != "" {
		parts = append(parts, track.Genre)
	} else if len(track.Genres) > 0 {
		parts = append(parts, strings.Join(track.Genres, ", "))
	}
	if track.BPM > 0 {
		parts = append(parts, fmt.Sprintf("%.0f BPM", track.BPM))
	}
	if track.Key != "" {
		parts = append(parts, track.Key)
	}
	if track.Energy != "" {
		parts = append(parts, track.Energy+" energy")
	}
	if len(track.Moods) > 0 {
		parts = append(parts, strings.Join(track.Moods, " · "))
	}
	return strings.Join(parts, "  •  ")
}

func panel(title, body string, width int) string {
	if width < 8 {
		width = 8
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#45475a")).
		Padding(0, 1).
		Width(width - 2).
		Render(titleStyle.Copy().Foreground(secondary).Render(title) + "\n" + body)
}

func progress(elapsed, duration time.Duration, width int) string {
	ratio := float64(elapsed) / float64(duration)
	ratio = math.Max(0, math.Min(1, ratio))
	filled := int(math.Round(float64(width) * ratio))
	return lipgloss.NewStyle().Foreground(accent).Render(strings.Repeat("━", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#45475a")).Render(strings.Repeat("─", width-filled))
}

func clock(value time.Duration) string {
	if value < 0 {
		value = 0
	}
	total := int(value.Seconds())
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}

func parseTime(current *api.QueueTrack) time.Time {
	if current == nil {
		return time.Time{}
	}
	return parseRFC3339(current.StartedAt)
}

func parseRFC3339(value string) time.Time {
	parsed, _ := time.Parse(time.RFC3339Nano, value)
	return parsed
}

func trackLine(track api.Track, width int) string {
	return truncate(track.Title+" — "+track.Artist, width)
}

func truncate(value string, width int) string {
	runes := []rune(value)
	if width <= 0 {
		return ""
	}
	if len(runes) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

func wrap(value string, width int) string {
	return wrapLimit(value, width, 0)
}

func wrapLimit(value string, width, maxLines int) string {
	if width < 10 {
		width = 10
	}
	words := strings.Fields(value)
	if len(words) == 0 {
		return ""
	}
	var lines []string
	line := words[0]
	for wordIndex, word := range words[1:] {
		if len([]rune(line))+1+len([]rune(word)) <= width {
			line += " " + word
			continue
		}
		lines = append(lines, line)
		if maxLines > 0 && len(lines) == maxLines {
			remainder := strings.Join(words[wordIndex+1:], " ")
			lines[maxLines-1] = truncate(lines[maxLines-1]+" "+remainder, width)
			return strings.Join(lines, "\n")
		}
		line = word
	}
	return strings.Join(append(lines, line), "\n")
}
