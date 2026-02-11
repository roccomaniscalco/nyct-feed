package gtfs

import (
	"nyct-feed/internal/pb"
	"slices"
	"strings"
)

type Departure struct {
	RouteId       string
	StopId        string
	FinalStopId   string
	FinalStopName string
	Times         []int64
}

func FindDepartures(stopIds []string, realtime []*pb.FeedMessage, schedule *Schedule) []Departure {
	tripToTimes := map[[3]string][]int64{}
	for _, stopId := range stopIds {
		for _, feedMsg := range realtime {
			for _, feedEntity := range feedMsg.GetEntity() {
				tripUpdate := feedEntity.GetTripUpdate()
				stopTimes := tripUpdate.GetStopTimeUpdate()
				routeId := tripUpdate.GetTrip().GetRouteId()
				for _, stopTime := range stopTimes {
					finalStopId := stopTimes[len(stopTimes)-1].GetStopId()
					tripKey := [3]string{routeId, stopId, finalStopId}
					// Exclude trips terminating at the target stop
					if stopTime.GetStopId() == stopId && finalStopId != stopId {
						tripToTimes[tripKey] = append(tripToTimes[tripKey], stopTime.GetDeparture().GetTime())
					}
				}
			}
		}
	}

	stopIdToName := schedule.GetStopIdToName()
	departures := []Departure{}
	for tripKey, times := range tripToTimes {
		routeId, stopId, finalStopId := tripKey[0], tripKey[1], tripKey[2]
		departures = append(departures, Departure{
			RouteId:       routeId,
			StopId:        stopId,
			FinalStopId:   finalStopId,
			FinalStopName: stopIdToName[finalStopId],
			Times:         times,
		})
	}

	// Sort departures by final stop name for consistent ordering
	slices.SortFunc(departures, func(a, b Departure) int {
		return strings.Compare(a.FinalStopName, b.FinalStopName)
	})

	return departures
}
