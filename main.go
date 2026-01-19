package main

import (
	"context"
	"log"
	"nyct-feed/internal/db"
	"nyct-feed/internal/gtfs"
	"time"
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

	database := db.Init(ctx)
	log.Println("DB Initialized")

	client := gtfs.NewClient(gtfs.ClientParams{
		Ctx:                 ctx,
		Db:                  database,
		ScheduleRefreshRate: time.Hour * 2,
		RealtimeRefreshRate: time.Second * 5,
	})

	err := client.SyncSchedule()
	if err != nil {
		log.Panicf("An error occurred while syncing schedule: %v", err)
	}
}

// func main() {
// 	realtime, _ := gtfs.GetRealtime()
// 	schedule, _ := gtfs.GetSchedule()

// 	departures := gtfs.FindDepartures(stopIds, realtime, schedule)
// 	fmt.Println(departures)
// }
