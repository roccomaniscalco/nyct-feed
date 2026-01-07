package main

import (
	"context"
	"log"
	"nyct-feed/internal/db"
	"nyct-feed/internal/gtfs"
)

// import (
// 	"fmt"
// 	"nyct-feed/internal/gtfs"
// )

// var stopIds = []string{
// 	"A46N",
// 	"A46S",
// 	"239N",
// 	"239S",
// }

// func main() {
// 	m := tui.NewModel()
// 	p := tea.NewProgram(&m, tea.WithAltScreen())
// 	if _, err := p.Run(); err != nil {
// 		log.Fatalf("Error running program:", err)
// 	}
// }

func main() {
	ctx := context.Background()
	log.Println("Process Started")

	database, queries := db.Init(ctx)
	log.Println("DB Initialized")

	schedule,_ := gtfs.GetSchedule()
	log.Println("Got Schedule")

	db.StoreSchedule(ctx, database, queries, schedule)
	log.Println("Stored Schedule")
}

// func main() {
// 	realtime, _ := gtfs.GetRealtime()
// 	schedule, _ := gtfs.GetSchedule()

// 	departures := gtfs.FindDepartures(stopIds, realtime, schedule)
// 	fmt.Println(departures)
// }
