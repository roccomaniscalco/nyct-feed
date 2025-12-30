package gtfs

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
)

const scheduleUrl = "https://rrgtfsfeeds.s3.amazonaws.com/gtfs_subway.zip"

type Schedule struct {
	Stops         []Stop         `file:"stops.txt"`
	StopTimes     []StopTime     `file:"stop_times.txt"`
	Trips         []Trip         `file:"trips.txt"`
	Routes        []Route        `file:"routes.txt"`
	Calendars     []Calendar     `file:"calendar.txt"`
	CalendarDates []CalendarDate `file:"calendar_dates.txt"`

	// Derived values
	RouteIdToRoute map[string]Route
	StopIdToName   map[string]string
	Stations       []Station
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
func (s *Schedule) CreateStations() {
	// Build trip ID to route ID map
	tripIdToRouteId := make(map[string]string)
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

	s.Stations = stations
}

func (s *Schedule) CreateStopIdToName() {
	stopIdToName := make(map[string]string)
	for _, stop := range s.Stops {
		stopIdToName[stop.StopId] = stop.StopName
	}
	s.StopIdToName = stopIdToName
}

func (s *Schedule) CreateRouteIdToRoute() {
	routeIdToRoute := make(map[string]Route)
	for _, route := range s.Routes {
		routeIdToRoute[route.RouteId] = route
	}
	s.RouteIdToRoute = routeIdToRoute
}

// GetSchedule fetches a GTFS schedule containing all schedule files.
func GetSchedule() (*Schedule, error) {
	schedule := Schedule{}
	scheduleType := reflect.TypeOf(schedule)

	scheduleFiles, err := fetchSchedule()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schedule: %v", err)
	}

	for _, file := range scheduleFiles {
		// Parse and set each item on schedule
		for i := 0; i < scheduleType.NumField(); i++ {
			field := scheduleType.Field(i)
			fileRowType := field.Type.Elem()
			fileName := field.Tag.Get("file")

			if fileName == file.Name {
				rc, err := file.Open()
				if err != nil {
					return nil, fmt.Errorf("failed to open zip file %s: %v", file.Name, err)
				}
				defer rc.Close()

				bytes, err := io.ReadAll(rc)
				if err != nil {
					return nil, fmt.Errorf("failed to read data from zip file %s: %v", file.Name, err)
				}

				switch fileRowType {
				case reflect.TypeOf(Stop{}):
					schedule.Stops = parseCSV(bytes, Stop{})
				case reflect.TypeOf(StopTime{}):
					schedule.StopTimes = parseCSV(bytes, StopTime{})
				case reflect.TypeOf(Trip{}):
					schedule.Trips = parseCSV(bytes, Trip{})
				case reflect.TypeOf(Route{}):
					schedule.Routes = parseCSV(bytes, Route{})
				case reflect.TypeOf(Calendar{}):
					schedule.Calendars = parseCSV(bytes, Calendar{})
				case reflect.TypeOf(CalendarDate{}):
					schedule.CalendarDates = parseCSV(bytes, CalendarDate{})
				}
			}
		}
	}

	// Create and store expensive derived values
	schedule.CreateRouteIdToRoute()
	schedule.CreateStopIdToName()
	schedule.CreateStations()

	return &schedule, nil
}

// parseCSV accepts a CSV as bytes and parses each row into a struct of type R.
// Each field in R to be parsed must specify a csv tag denoting its column header.
func parseCSV[R any](bytes []byte, row R) []R {
	r := reflect.TypeOf(row)
	rows := []R{}

	// Only accept structs
	if r.Kind() != reflect.Struct {
		log.Panicf("row must be of type struct: received %s", r.Kind())
	}

	lines := strings.Split(string(bytes), "\n")
	headers := parseCSVLine(lines[0])

	// Map header to column number
	headerToCol := make(map[string]int)
	for i, header := range headers {
		headerToCol[header] = i
	}

	// Map each CSV line to a new row struct
	for _, line := range lines[1:] {
		// Skip blank lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		cells := parseCSVLine(line)
		newRow := reflect.New(r).Elem()

		// Populate row values from CSV cells
		for i := 0; i < r.NumField(); i++ {
			fieldValue := newRow.Field(i)
			fieldType := newRow.Type().Field(i)

			// Get CSV header from tag or skip
			header := fieldType.Tag.Get("csv")
			if header == "" {
				continue
			}

			// Set row field to parsed cell value
			if col, exists := headerToCol[header]; exists && col < len(cells) {
				cell := cells[col]
				parseCSVCellValue(cell, fieldValue, fieldType)
			}
		}

		rows = append(rows, newRow.Interface().(R))
	}

	return rows
}

func parseCSVLine(line string) []string {
	cells := []string{}
	chars := strings.Split(line, "")
	cellStart := 0
	inQuotes := false

	for i, char := range chars {
		if char == "\"" {
			inQuotes = !inQuotes
		} else if !inQuotes && char == "," {
			field := strings.Join(chars[cellStart:i], "")
			cells = append(cells, field)
			cellStart = i + 1
		}
	}

	// Handle the last field (after final comma or whole line if no commas)
	if cellStart < len(chars) {
		field := strings.Join(chars[cellStart:], "")
		cells = append(cells, field)
	}

	return cells
}

// Parse cell to the corresponding field type and set field value
func parseCSVCellValue(cell string, fieldValue reflect.Value, fieldType reflect.StructField) {
	if fieldValue.CanSet() {
		switch fieldType.Type.Kind() {
		case reflect.String:
			val := strings.Trim(cell, "\"")
			fieldValue.SetString(val)
		case reflect.Int:
			val, err := strconv.Atoi(cell)
			if err != nil && cell != "" {
				log.Printf("warning: failed to parse field %s: %v", fieldType.Name, err)
			}
			fieldValue.SetInt(int64(val))
		case reflect.Bool:
			val, err := strconv.ParseBool(cell)
			if err != nil {
				log.Printf("warning: failed to parse field %s: %v", fieldType.Name, err)
			}
			fieldValue.SetBool(val)
		case reflect.Float64:
			val, err := strconv.ParseFloat(cell, 64)
			if err != nil {
				log.Printf("warning: failed to parse field %s: %v", fieldType.Name, err)
			}
			fieldValue.SetFloat(val)
		default:
			log.Panicf("unsupported CSV field type: %v for field %s", fieldType.Type.Kind(), fieldType.Name)
		}
	}
}

// fetchSchedule requests a GTFS schedule ZIP folder and returns its zip files.
func fetchSchedule() ([]*zip.File, error) {
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

	// Store each schedule file
	return zipReader.File, nil
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
