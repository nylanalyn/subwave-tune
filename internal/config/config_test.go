package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePrecedence(t *testing.T) {
	t.Setenv("SUBWAVE_TUNE_URL", "https://env.example")
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"url":"https://file.example","volume":42}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Parse([]string{"--config", path, "--url", "https://flag.example", "--volume", "55"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL != "https://flag.example" || cfg.Volume != 55 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestParsePositionalURL(t *testing.T) {
	t.Setenv("SUBWAVE_TUNE_URL", "")
	t.Setenv("SUBWAVE_TUNE_PASSWORD", "")
	temp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", temp)
	t.Setenv("AppData", temp)
	cfg, err := Parse([]string{"radio.example"})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL != "https://radio.example" {
		t.Fatalf("URL = %q", cfg.URL)
	}
}
