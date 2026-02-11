package main

// import (
// 	"log"
// 	"nyct-feed/internal/tui"
// 	tea "github.com/charmbracelet/bubbletea"
// )

// func main() {
// 	m := tui.NewModel()
// 	p := tea.NewProgram(&m, tea.WithAltScreen())

// 	// f, err := tea.LogToFile("data/debug.log", "debug")
// 	// if err != nil {
// 	// 	log.Fatalln("Error setting up log file:", err)
// 	// }
// 	// defer f.Close()

// 	if _, err := p.Run(); err != nil {
// 		log.Fatalln("Error running program:", err)
// 	}
// }

import (
	"log"
	"nyct-feed/internal/gtfs"
)

var stopIds = []string{
	"A46N",
	"A46S",
	"239N",
	"239S",
}

func main() {
	schedule, _ := gtfs.GetSchedule()
	log.Println("Schedule!")

	gtfs.FindScheduledDepartures("A46", schedule)
	log.Println("done!")
}
