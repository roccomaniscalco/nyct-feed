package gtfs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"nyct-feed/internal/csvutil"
	"os"
)

const scheduleUrl = "https://rrgtfsfeeds.s3.amazonaws.com/gtfs_supplemented.zip"

type Schedule struct {
	Stops         []Stop         `file:"stops.txt"`
	StopTimes     []StopTime     `file:"stop_times.txt"`
	Trips         []Trip         `file:"trips.txt"`
	Routes        []Route        `file:"routes.txt"`
	Calendars     []Calendar     `file:"calendar.txt"`
	CalendarDates []CalendarDate `file:"calendar_dates.txt"`
	cache         scheduleCache
}

// Cached values derived from schedule
type scheduleCache struct {
	stopIdToName map[string]string
	stations     []Station
}

type Station struct {
	Stop
	Routes []Route
}

type Stop struct {
	StopId        string  `csv:"stop_id"`
	StopName      string  `csv:"stop_name"`
	StopLat       float64 `csv:"stop_lat"`
	StopLon       float64 `csv:"stop_lon"`
	LocationType  int     `csv:"location_type"` // 0 = Platform, 1 = Station
	ParentStation string  `csv:"parent_station"`
}

type StopTime struct {
	TripId        string `csv:"trip_id"`
	StopId        string `csv:"stop_id"`
	ArrivalTime   string `csv:"arrival_time"`
	DepartureTime string `csv:"departure_time"`
	StopSequence  int    `csv:"stop_sequence"`
}

type Trip struct {
	RouteId      string `csv:"route_id"`
	TripId       string `csv:"trip_id"`
	ServiceId    string `csv:"service_id"`
	TripHeadsign string `csv:"trip_headsign"`
	DirectionId  int    `csv:"direction_id"`
	ShapeId      string `csv:"shape_id"`
}

type Route struct {
	RouteId        string `csv:"route_id"`
	AgencyId       string `csv:"agency_id"`
	RouteShortName string `csv:"route_short_name"`
	RouteLongName  string `csv:"route_long_name"`
	RouteDesc      string `csv:"route_desc"`
	RouteType      int    `csv:"route_type"`
	RouteUrl       string `csv:"route_url"`
	RouteColor     string `csv:"route_color"`
	RouteTextColor string `csv:"route_text_color"`
	RouteSortOrder int    `csv:"route_sort_order"`
}

type Calendar struct {
	ServiceId string `csv:"service_id"`
	Monday    bool   `csv:"monday"`
	Tuesday   bool   `csv:"tuesday"`
	Wednesday bool   `csv:"wednesday"`
	Thursday  bool   `csv:"thursday"`
	Friday    bool   `csv:"friday"`
	Saturday  bool   `csv:"saturday"`
	Sunday    bool   `csv:"sunday"`
	StartDate string `csv:"start_date"`
	EndDate   string `csv:"end_date"`
}

type CalendarDate struct {
	ServiceId     string `csv:"service_id"`
	Date          string `csv:"date"`
	ExceptionType int    `csv:"exception_type"` // 1 = Added, 2 = Cancelled
}

// GetStations returns a subset of Stops that are considered to be stations.
// Each Stop returned includes a set of RouteIds that pass through the station.
func (s *Schedule) GetStations() []Station {
	if s.cache.stations != nil {
		return s.cache.stations
	}

	// Build trip ID to route ID map
	tripIdToRouteId := make(map[string]string, len(s.Trips))
	for _, trip := range s.Trips {
		tripIdToRouteId[trip.TripId] = trip.RouteId
	}

	// Build station ID to route IDs map directly
	stationIdToRouteIds := make(map[string]map[string]struct{})
	for _, stopTime := range s.StopTimes {
		// Shave off the "N" or "S" from StopId to get parent StopId
		parentStopId := stopTime.StopId[:3]
		if routeId, exists := tripIdToRouteId[stopTime.TripId]; exists {
			if stationIdToRouteIds[parentStopId] == nil {
				stationIdToRouteIds[parentStopId] = make(map[string]struct{})
			}
			stationIdToRouteIds[parentStopId][routeId] = struct{}{}
		}
	}

	// Filter and populate stations
	var stations []Station
	for _, stop := range s.Stops {
		if stop.LocationType == 1 {
			routeIds := stationIdToRouteIds[stop.StopId]
			routes := []Route{}
			// Iterating instead of using map to preserve route sort order
			for _, route := range s.Routes {
				if _, exists := routeIds[route.RouteId]; exists {
					routes = append(routes, route)
				}
			}
			stations = append(stations, Station{Stop: stop, Routes: routes})
		}
	}

	s.cache.stations = stations
	return stations
}

func (s *Schedule) GetStopIdToName() map[string]string {
	if s.cache.stopIdToName != nil {
		return s.cache.stopIdToName
	}

	stopIdToName := make(map[string]string)
	for _, stop := range s.Stops {
		stopIdToName[stop.StopId] = stop.StopName
	}

	s.cache.stopIdToName = stopIdToName
	return stopIdToName
}

// GetSchedule fetches a GTFS schedule containing all schedule files.
func GetSchedule() (*Schedule, error) {
	scheduleFiles, err := fetchScheduleFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %v", err)
	}

	log.Println("Schedule files fetched")

	schedule := Schedule{}
	for _, file := range scheduleFiles {
		if err := parseScheduleFile(file, &schedule); err != nil {
			return nil, fmt.Errorf("failed to parse schedule file %s: %v", file.Name, err)
		}
	}

	log.Println("Schedule parsed")

	return &schedule, nil
}

// fetchScheduleFiles requests a GTFS schedule ZIP folder and returns its zip files.
func fetchScheduleFiles() ([]*zip.File, error) {
	// Download the ZIP folder
	resp, err := http.Get(scheduleUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to download schedule from %s: %v", scheduleUrl, err)
	}
	defer resp.Body.Close()

	// Read the ZIP data into memory
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ZIP data from response: %v", err)
	}

	// Create a ZIP reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("failed to create ZIP reader: %v", err)
	}

	return zipReader.File, nil
}

func parseScheduleFile(file *zip.File, schedule *Schedule) error {
	rc, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open zip file %s: %v", file.Name, err)
	}
	defer rc.Close()

	// Map filename to schedule field and type
	switch file.Name {
	case "stops.txt":
		var err error
		schedule.Stops, err = csvutil.ReadAllParsed(rc, Stop{})
		return err
	case "stop_times.txt":
		var err error
		schedule.StopTimes, err = csvutil.ReadAllParsed(rc, StopTime{})
		return err
	case "trips.txt":
		var err error
		schedule.Trips, err = csvutil.ReadAllParsed(rc, Trip{})
		return err
	case "routes.txt":
		var err error
		schedule.Routes, err = csvutil.ReadAllParsed(rc, Route{})
		return err
	case "calendar.txt":
		var err error
		schedule.Calendars, err = csvutil.ReadAllParsed(rc, Calendar{})
		return err
	case "calendar_dates.txt":
		var err error
		schedule.CalendarDates, err = csvutil.ReadAllParsed(rc, CalendarDate{})
		return err
	default:
		return nil // Skip unknown files
	}
}

func storeScheduleFile(file *zip.File) {
	rc, err := file.Open()
	if err != nil {
		log.Panicf("failed to open zip file %s: %v", file.Name, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		log.Panicf("failed to read data from zip file %s: %v", file.Name, err)
	}

	// Ensure the data directory exists
	if err := os.MkdirAll(dataDir, dirPerms); err != nil {
		log.Panicf("failed to create data directory %s: %v", dataDir, err)
	}

	err = os.WriteFile(dataDir+file.Name, data, filePerms)
	if err != nil {
		log.Panicf("failed to write file %s: %v", dataDir+file.Name, err)
	}
}
