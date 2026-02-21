package main

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/choice404/vegas-protocol/vegas-tui/internal"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	zone "github.com/lrstanley/bubblezone"
)

func main() {
	_ = godotenv.Load() // optional .env for SPOTIFY_ID, SPOTIFY_SECRET, etc.
	zone.NewGlobal()

	p := tea.NewProgram(
		internal.NewApp(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// If an update was installed, re-exec the new binary
	if internal.RestartAfterUpdate {
		exe, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to find executable path: %v\n", err)
			os.Exit(1)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to resolve executable path: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Restarting with updated binary...\n")
		if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to restart: %v\n", err)
			os.Exit(1)
		}
	}
}
