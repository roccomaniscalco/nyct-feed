package main

// import (
// 	"fmt"
// 	"nyct-feed/internal/gtfs"
// 	"nyct-feed/internal/pb"
// 	"nyct-feed/internal/query"
// 	"time"
// )

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

// func main() {
// 	feedMessageQueryChannel := make(chan query.Query[[]*pb.FeedMessage])
// 	query.CreateQuery[[]*pb.FeedMessage](query.QueryOptions[[]*pb.FeedMessage]{
// 		QueryFn:         gtfs.FetchFeeds,
// 		QueryChannel:    feedMessageQueryChannel,
// 		RefetchInterval: time.Second * 15,
// 	})

// 	for feedMessageQuery := range feedMessageQueryChannel {
// 		fmt.Printf("Status: %v\n", feedMessageQuery.Status)
// 		fmt.Printf("Fetch Status: %v\n", feedMessageQuery.FetchStatus)
// 		fmt.Printf("Data Updated At: %v\n", feedMessageQuery.DataUpdatedAt)
// 		fmt.Printf("Data Length: %v\n", len(feedMessageQuery.Data))
// 	}
// }
