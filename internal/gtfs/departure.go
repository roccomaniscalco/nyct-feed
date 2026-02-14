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

// FindScheduledDepartures returns all scheduled departures for a given station ID.
func FindScheduledDepartures(stationId string, schedule *Schedule) []Departure {
	now := time.Now()
	nowDate := now.Format("20060102")
	nowWeekday := now.Weekday()

	departures := []Departure{}

	// 1. Get active service IDs
	serviceIds := map[string]struct{}{}

	// 1.1 Add service ID if date is in range and weekday receives service
	for _, calendar := range schedule.Calendars {
		isDateInService := calendar.StartDate <= nowDate && calendar.EndDate >= nowDate
		isWeekdayInService := calendar.GetIsWeekdayActive(nowWeekday)

		if isDateInService && isWeekdayInService {
			serviceIds[calendar.ServiceId] = struct{}{}
		}
	}
	log.Printf("Got calendar: %+v\n", len(schedule.Calendars))

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
	log.Printf("Got calendar dates: %+v\n", len(schedule.CalendarDates))

	stationIdToRouteIds := schedule.GetStationIdToRouteIds()
	routeIds := stationIdToRouteIds[stationId]
	log.Printf("Got route IDs for station %s: %+v\n", stationId, routeIds)

	// 2. Get trip IDs for active service IDs
	tripIds := map[string]struct{}{}
	for routeId := range routeIds {
		for serviceId := range serviceIds {
			for _, trip := range schedule.Trips {

				serviceIdMatch := trip.ServiceId == serviceId
				routeIdMatch := trip.RouteId == routeId

				if serviceIdMatch && routeIdMatch {
					tripIds[trip.TripId] = struct{}{}
				}
			}
		}
	}
	log.Printf("Got trips: %+v\n", len(tripIds))

	// 3. Get stop times for active trip IDs
	stopTimes := []StopTime{}
	for tripId := range tripIds {
		for _, stopTime := range schedule.StopTimes {
			// Shave off the "N" or "S" from StopId to get parent StopId
			parentStopId := stopTime.StopId[:3]

			isStationMatch := parentStopId == stationId
			isTripMatch := tripId == stopTime.TripId

			if isStationMatch && isTripMatch {
				stopTimes = append(stopTimes, stopTime)
			}
		}
	}
	log.Printf("Got stop times: %+v\n", len(stopTimes))

	return departures
}

func (c *Calendar) GetIsWeekdayActive(weekday time.Weekday) bool {
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
