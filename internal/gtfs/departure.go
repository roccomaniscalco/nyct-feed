package gtfs

import (
	"log"
	"nyct-feed/internal/pb"
	"slices"
	"strings"
	"time"
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

// Find all static departures for a given stop ID within 12hr
// Exclude cancelled trips, and include added trips
// 
// Ex. Find all departures going through Van Cortlandt Park-242 St on Mon Dec 29 starting at 8am
// Station ID: 101
// 
// StopTimes where stopId is 101N or 101S
// 
func FindScheduledDepartures(stationId string, schedule *Schedule) []Departure {
	now := time.Now().Format("20060102")
	stopIds := []string{stationId+"N", stationId+"S"}

	departures := []Departure{}

	serviceIds := []string{}
	for _, calendar := range schedule.Calendars {
		if calendar.StartDate <= now && calendar.EndDate >= now {
			serviceIds = append(serviceIds, calendar.ServiceId)
		}
	}

	stopTimes := []StopTime{}
	for _, stopId := range stopIds {
		for _, stopTime := range schedule.StopTimes {
			if stopId == stopTime.StopId {
				stopTimes = append(stopTimes, stopTime)
			}
		}
	}

	return departures
}