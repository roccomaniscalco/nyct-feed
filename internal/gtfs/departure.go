package gtfs

import (
	"nyct-feed/internal/pb"
	"slices"
)

type Departure struct {
	Route         Route
	FinalStopId   string
	FinalStopName string
	Times         []int64
}

func FindDepartures(stopIds []string, feeds []*pb.FeedMessage, schedule Schedule) []Departure {
	stopIdToName := schedule.GetStopIdToName()
	routeIdToRoute := schedule.GetRouteIdToRoute()

	tripToTimes := map[[2]string][]int64{}

	for _, stopId := range stopIds {
		for _, feed := range feeds {
			for _, feedEntity := range feed.GetEntity() {
				tripUpdate := feedEntity.GetTripUpdate()
				stopTimes := tripUpdate.GetStopTimeUpdate()
				routeId := tripUpdate.GetTrip().GetRouteId()
				for _, stopTime := range stopTimes {
					finalStopId := stopTimes[len(stopTimes)-1].GetStopId()
					tripKey := [2]string{routeId, finalStopId}
					// Exclude trips terminating at the target stop
					if stopTime.GetStopId() == stopId && finalStopId != stopId {
						tripToTimes[tripKey] = append(tripToTimes[tripKey], stopTime.GetDeparture().GetTime())
					}
				}
			}
		}
	}

	departures := []Departure{}
	for tripKey, times := range tripToTimes {
		routeId, finalStopId := tripKey[0], tripKey[1]
		slices.Sort(times)
		departures = append(departures, Departure{
			Route:         routeIdToRoute[routeId],
			FinalStopId:   finalStopId,
			FinalStopName: stopIdToName[finalStopId],
			Times:         times,
		})
	}

	return departures
}
