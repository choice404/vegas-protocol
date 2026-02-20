package main

import (
	"fmt"
	"os"

	"rebel-hacks-tui/internal"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(internal.NewApp(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
