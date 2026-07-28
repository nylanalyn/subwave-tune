package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultURL = "http://localhost:7700"

type Config struct {
	URL      string `json:"url"`
	Password string `json:"password,omitempty"`
	Volume   int    `json:"volume"`
	NoPlay   bool   `json:"-"`
}

func Parse(args []string) (Config, error) {
	path, err := configPath(args)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{URL: defaultURL, Volume: 75}
	if err := loadFile(path, &cfg); err != nil {
		return Config{}, err
	}
	if value := os.Getenv("SUBWAVE_TUNE_URL"); value != "" {
		cfg.URL = value
	}
	if value := os.Getenv("SUBWAVE_TUNE_PASSWORD"); value != "" {
		cfg.Password = value
	}

	fs := flag.NewFlagSet("subwave-tune", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&cfg.URL, "url", cfg.URL, "SUB/WAVE station base URL")
	fs.StringVar(&path, "config", path, "configuration file")
	fs.IntVar(&cfg.Volume, "volume", cfg.Volume, "initial volume (0-100)")
	fs.BoolVar(&cfg.NoPlay, "no-play", false, "open the TUI without starting audio")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() > 1 {
		return Config{}, errors.New("expected at most one station URL")
	}
	if fs.NArg() == 1 {
		cfg.URL = fs.Arg(0)
	}

	cfg.URL = strings.TrimRight(strings.TrimSpace(cfg.URL), "/")
	if cfg.URL == "" {
		return Config{}, errors.New("station URL cannot be empty")
	}
	if !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
		cfg.URL = "https://" + cfg.URL
	}
	if cfg.Volume < 0 || cfg.Volume > 100 {
		return Config{}, fmt.Errorf("volume must be between 0 and 100")
	}
	return cfg, nil
}

func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join(".", "subwave-tune.json")
	}
	return filepath.Join(dir, "subwave-tune", "config.json")
}

func configPath(args []string) (string, error) {
	path := DefaultPath()
	for i, arg := range args {
		if strings.HasPrefix(arg, "--config=") {
			return strings.TrimPrefix(arg, "--config="), nil
		}
		if arg == "--config" {
			if i+1 >= len(args) {
				return "", errors.New("--config requires a path")
			}
			return args[i+1], nil
		}
	}
	return path, nil
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}
