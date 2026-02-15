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
	Times         []time.Time
}

func FindDepartures(stopIds []string, realtime []*pb.FeedMessage, schedule *Schedule) []Departure {
	tripToTimes := map[[3]string][]time.Time{}
	for _, stopId := range stopIds {
		for _, feedMsg := range realtime {
			for _, feedEntity := range feedMsg.GetEntity() {
				tripUpdate := feedEntity.GetTripUpdate()
				routeId := tripUpdate.GetTrip().GetRouteId()
				stopTimes := tripUpdate.GetStopTimeUpdate()
				for _, stopTime := range stopTimes {
					finalStopId := stopTimes[len(stopTimes)-1].GetStopId()
					tripKey := [3]string{routeId, stopId, finalStopId}
					time := time.Unix(stopTime.GetDeparture().GetTime(), 0)
					// Exclude trips terminating at the target stop
					if stopTime.GetStopId() == stopId && finalStopId != stopId {
						tripToTimes[tripKey] = append(tripToTimes[tripKey], time)
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

// TODO: Need to find trips that are in service
func getActiveTrips(stationId string, schedule *Schedule) {
	now := time.Now()
	nowDate := now.Format("20060102")
	nowWeekday := now.Weekday()

	// 1. Get active service IDs
	serviceIds := map[string]struct{}{}
	// 1.1 Add service ID if date is in range and weekday receives service
	for _, calendar := range schedule.Calendars {
		isDateInService := calendar.StartDate <= nowDate && calendar.EndDate >= nowDate
		isWeekdayInService := calendar.IsWeekdayActive(nowWeekday)

		if isDateInService && isWeekdayInService {
			serviceIds[calendar.ServiceId] = struct{}{}
		}
	}
	// 1.2 Add or remove service IDs based on calendar date exceptions
	for _, calendarDate := range schedule.CalendarDates {
		if calendarDate.Date == nowDate {
			if calendarDate.ExceptionType == 1 { // Added service
				serviceIds[calendarDate.ServiceId] = struct{}{}
			} else if calendarDate.ExceptionType == 2 { // Cancelled service
				delete(serviceIds, calendarDate.ServiceId)
			} else {
				log.Printf("invalid exception type on calendar date: %v", calendarDate)
			}
		}
	}
	log.Printf("Got active service IDs: %+v\n", len(serviceIds))
}

// 1. Need day of the week to determine time.Time of departure
// 2. Need to apply calendar exceptions to determine active trips
func FindScheduleDepartures(stopIds []string, schedule *Schedule) []Departure {
	tripIdToStopTimes := make(map[string][]StopTime)
	for _, stopTime := range schedule.StopTimes {
		tripIdToStopTimes[stopTime.TripId] = append(tripIdToStopTimes[stopTime.TripId], stopTime)
	}

	tripToTimes := map[[3]string][]string{}
	for _, stopId := range stopIds {
		for _, trip := range schedule.Trips { // TODO: Replace schedule.Trips with active trips
			routeId := trip.RouteId
			stopTimes := tripIdToStopTimes[trip.TripId]
			for _, stopTime := range stopTimes {
				finalStopId := stopTimes[len(stopTimes)-1].StopId
				tripKey := [3]string{routeId, stopId, finalStopId}
				// Exclude trips terminating at the target stop
				if stopTime.StopId == stopId && finalStopId != stopId {
					tripToTimes[tripKey] = append(tripToTimes[tripKey], stopTime.DepartureTime)
				}
			}
		}
	}

	stopIdToName := schedule.GetStopIdToName()
	departures := []Departure{}
	for tripKey /*,times*/ := range tripToTimes {
		routeId, stopId, finalStopId := tripKey[0], tripKey[1], tripKey[2]
		departures = append(departures, Departure{
			RouteId:       routeId,
			StopId:        stopId,
			FinalStopId:   finalStopId,
			FinalStopName: stopIdToName[finalStopId],
			// Times:         times,
		})
	}

	// Sort departures by final stop name for consistent ordering
	slices.SortFunc(departures, func(a, b Departure) int {
		return strings.Compare(a.FinalStopName, b.FinalStopName)
	})

	return departures
}

func (c *Calendar) IsWeekdayActive(weekday time.Weekday) bool {
	switch weekday {
	case time.Monday:
		return c.Monday
	case time.Tuesday:
		return c.Tuesday
	case time.Wednesday:
		return c.Wednesday
	case time.Thursday:
		return c.Thursday
	case time.Friday:
		return c.Friday
	case time.Saturday:
		return c.Saturday
	case time.Sunday:
		return c.Sunday
	default:
		log.Panicln("unreachable code: invalid weekday")
		return false
	}
}
