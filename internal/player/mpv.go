package player

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type MPV struct {
	mu      sync.Mutex
	command *exec.Cmd
	done    chan error
	socket  string
	volume  int
	muted   bool
}

type Status struct {
	Running bool
	Volume  int
	Muted   bool
}

func New(volume int) *MPV {
	return &MPV{volume: volume}
}

func Available() error {
	if _, err := exec.LookPath("mpv"); err != nil {
		return errors.New("mpv was not found in PATH")
	}
	return nil
}

func (m *MPV) Start(streamURL string) error {
	if err := Available(); err != nil {
		return err
	}
	m.mu.Lock()
	if m.runningLocked() {
		m.mu.Unlock()
		return nil
	}
	socket := filepath.Join(os.TempDir(), fmt.Sprintf("subwave-tune-%d.sock", os.Getpid()))
	_ = os.Remove(socket)
	command := exec.Command(
		"mpv",
		"--no-video",
		"--really-quiet",
		"--idle=no",
		"--cache=yes",
		"--cache-secs=30",
		"--input-ipc-server="+socket,
		"--volume="+strconv.Itoa(m.volume),
		streamURL,
	)
	command.Stdin = nil
	command.Stdout = nil
	command.Stderr = nil
	if err := command.Start(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("start mpv: %w", err)
	}
	done := make(chan error, 1)
	m.command = command
	m.done = done
	m.socket = socket
	m.muted = false
	m.mu.Unlock()

	go func() {
		done <- command.Wait()
		close(done)
		_ = os.Remove(socket)
	}()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socket); err == nil {
			return nil
		}
		select {
		case err := <-done:
			if err == nil {
				err = errors.New("mpv exited before opening its IPC socket")
			}
			return err
		default:
		}
		time.Sleep(25 * time.Millisecond)
	}
	m.Stop()
	return errors.New("mpv did not open its IPC socket")
}

func (m *MPV) Stop() {
	m.mu.Lock()
	if !m.runningLocked() {
		m.command = nil
		m.mu.Unlock()
		return
	}
	command := m.command
	done := m.done
	socket := m.socket
	m.mu.Unlock()

	_ = send(socket, []any{"quit"})
	select {
	case <-done:
	case <-time.After(750 * time.Millisecond):
		_ = command.Process.Kill()
		select {
		case <-done:
		case <-time.After(250 * time.Millisecond):
		}
	}
	m.mu.Lock()
	if m.command == command {
		m.command = nil
		m.done = nil
		m.socket = ""
	}
	m.mu.Unlock()
	_ = os.Remove(socket)
}

func (m *MPV) Toggle(streamURL string) (bool, error) {
	if m.Status().Running {
		m.Stop()
		return false, nil
	}
	if err := m.Start(streamURL); err != nil {
		return false, err
	}
	return true, nil
}

func (m *MPV) AdjustVolume(delta int) error {
	m.mu.Lock()
	next := m.volume + delta
	if next < 0 {
		next = 0
	}
	if next > 100 {
		next = 100
	}
	m.volume = next
	socket := m.socket
	running := m.runningLocked()
	m.mu.Unlock()
	if running {
		return send(socket, []any{"set_property", "volume", next})
	}
	return nil
}

func (m *MPV) ToggleMute() error {
	m.mu.Lock()
	m.muted = !m.muted
	muted := m.muted
	socket := m.socket
	running := m.runningLocked()
	m.mu.Unlock()
	if running {
		return send(socket, []any{"set_property", "mute", muted})
	}
	return nil
}

func (m *MPV) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return Status{Running: m.runningLocked(), Volume: m.volume, Muted: m.muted}
}

func (m *MPV) runningLocked() bool {
	if m.command == nil || m.command.Process == nil {
		return false
	}
	select {
	case <-m.done:
		return false
	default:
		return true
	}
}

func send(socket string, command []any) error {
	connection, err := net.DialTimeout("unix", socket, 400*time.Millisecond)
	if err != nil {
		return fmt.Errorf("connect to mpv: %w", err)
	}
	defer connection.Close()
	payload, err := json.Marshal(map[string]any{"command": command})
	if err != nil {
		return err
	}
	if _, err := connection.Write(append(payload, '\n')); err != nil {
		return fmt.Errorf("write mpv command: %w", err)
	}
	_ = connection.SetReadDeadline(time.Now().Add(400 * time.Millisecond))
	_, _ = bufio.NewReader(connection).ReadBytes('\n')
	return nil
}
