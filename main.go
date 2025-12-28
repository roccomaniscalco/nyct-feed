package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"

	"nyct-feed/internal/tui"
)

var stopIds = []string{
	"A46N",
	"A46S",
	"239N",
	"239S",
}

func main() {
	m := tui.NewModel()
	p := tea.NewProgram(&m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program:", err)
	}
}
