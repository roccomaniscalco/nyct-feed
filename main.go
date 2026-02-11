package main

import (
	"log"
	"nyct-feed/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

// import (
// 	"fmt"
// 	"nyct-feed/internal/gtfs"
// )

var stopIds = []string{
	"A46N",
	"A46S",
	"239N",
	"239S",
}

func main() {
	m := tui.NewModel()
	p := tea.NewProgram(&m, tea.WithAltScreen())

	// f, err := tea.LogToFile("data/debug.log", "debug")
	// if err != nil {
	// 	log.Fatalln("Error setting up log file:", err)
	// }
	// defer f.Close()

	if _, err := p.Run(); err != nil {
		log.Fatalln("Error running program:", err)
	}
}

// func main() {
// 	realtime, _ := gtfs.GetRealtime()
// 	schedule, _ := gtfs.GetSchedule()

// 	departures := gtfs.FindDepartures(stopIds, realtime, schedule)
// 	fmt.Println(departures)
// }
