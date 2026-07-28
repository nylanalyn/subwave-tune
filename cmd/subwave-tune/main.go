package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"subwave-tune/internal/api"
	"subwave-tune/internal/config"
	"subwave-tune/internal/player"
	"subwave-tune/internal/ui"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		return
	}
	cfg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "subwave-tune:", err)
		os.Exit(2)
	}
	client, err := api.New(cfg.URL, cfg.Password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "subwave-tune:", err)
		os.Exit(2)
	}
	mpv := player.New(cfg.Volume)
	defer mpv.Stop()

	model := ui.New(client, mpv, !cfg.NoPlay)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "subwave-tune:", err)
		os.Exit(1)
	}
}
