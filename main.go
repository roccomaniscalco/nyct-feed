package main

import (
	"log"
	"nyct-feed/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m := tui.NewModel()
	p := tea.NewProgram(&m, tea.WithAltScreen())

	f, err := tea.LogToFile("data/debug.log", "debug")
	if err != nil {
		log.Fatalln("Error setting up log file:", err)
	}
	defer f.Close()

	if _, err := p.Run(); err != nil {
		log.Fatalln("Error running program:", err)
	}
}
