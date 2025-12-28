package gtfs

import (
	"archive/zip"
	"bytes"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

const (
	scheduleUrl = "https://rrgtfsfeeds.s3.amazonaws.com/gtfs_subway.zip"
	scheduleTtl = 24 * time.Hour
)

type Schedule struct {
	Stops     []Stop     `file:"stops.txt"`
	StopTimes []StopTime `file:"stop_times.txt"`
	Trips     []Trip     `file:"trips.txt"`
	Routes    []Route    `file:"routes.txt"`
	Shapes    []Shape    `file:"shapes.txt"`
}

type Stop struct {
	StopId        string  `csv:"stop_id"`
	StopName      string  `csv:"stop_name"`
	StopLat       float64 `csv:"stop_lat"`
	StopLon       float64 `csv:"stop_lon"`
	LocationType  int     `csv:"location_type"` // 0 = Platform, 1 = Station
	ParentStation string  `csv:"parent_station"`
	RouteIds      map[string]bool
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

type Shape struct {
	ShapeId    string  `csv:"shape_id"`
	ShapePtSeq int     `csv:"shape_pt_sequence"`
	ShapePtLat float64 `csv:"shape_pt_lat"`
	ShapePtLon float64 `csv:"shape_pt_lon"`
}

// GetStations returns a subset of Stops that are considered to be stations.
// Each Stop returned includes a set of RouteIds that pass through the station.
func (s *Schedule) GetStations() []Stop {
	// Build trip ID to route ID map
	tripIdToRouteId := make(map[string]string)
	for _, trip := range s.Trips {
		tripIdToRouteId[trip.TripId] = trip.RouteId
	}

	// Build station ID to route IDs map directly
	stationIdToRouteIds := make(map[string]map[string]bool)
	for _, stopTime := range s.StopTimes {
		// Shave off the "N" or "S" from StopId to get parent StopId
		parentStopId := stopTime.StopId[:3]
		if routeId, exists := tripIdToRouteId[stopTime.TripId]; exists {
			if stationIdToRouteIds[parentStopId] == nil {
				stationIdToRouteIds[parentStopId] = make(map[string]bool)
			}
			stationIdToRouteIds[parentStopId][routeId] = true
		}
	}

	// Filter and populate stations
	var stations []Stop
	for _, stop := range s.Stops {
		if stop.LocationType == 1 {
			stop.RouteIds = stationIdToRouteIds[stop.StopId]
			stations = append(stations, stop)
		}
	}

	return stations
}

func (s *Schedule) GetStopIdToName() map[string]string {
	stopIdToName := make(map[string]string)
	for _, stop := range s.Stops {
		stopIdToName[stop.StopId] = stop.StopName
	}
	return stopIdToName
}

func (s *Schedule) GetRouteIdToRoute() map[string]Route {
	routeIdToRoute := make(map[string]Route)
	for _, route := range s.Routes {
		routeIdToRoute[route.RouteId] = route
	}
	return routeIdToRoute
}

// GetSchedule returns a GTFS schedule containing all schedule files.
// The schedule zip file is requested and stored when missing or stale.
// The schedule files are read and parsed from storage.
func GetSchedule() Schedule {
	var schedule Schedule
	scheduleType := reflect.TypeOf(schedule)

	currentTime := time.Now()
	isScheduleDirty := false

	// Schedule is dirty if any file is missing or expired
	for i := 0; i < scheduleType.NumField(); i++ {
		fileName := scheduleType.Field(i).Tag.Get("file")

		fileInfo, err := os.Stat(dataDir + fileName)
		if err != nil {
			isScheduleDirty = true
			break
		}
		if currentTime.Sub(fileInfo.ModTime()) > scheduleTtl {
			isScheduleDirty = true
			break
		}
	}

	if isScheduleDirty {
		fetchAndStoreSchedule()
	}

	// Parse and set each item on schedule
	for i := 0; i < scheduleType.NumField(); i++ {
		field := scheduleType.Field(i)
		fileRowType := field.Type.Elem()
		fileName := field.Tag.Get("file")
		filePath := dataDir + fileName

		bytes, err := os.ReadFile(filePath)
		if err != nil {
			log.Panicf("failed to read schedule file: %s", filePath)
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
		case reflect.TypeOf(Shape{}):
			schedule.Shapes = parseCSV(bytes, Shape{})
		}
	}

	return schedule
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

// fetchAndStoreSchedule requests a GTFS schedule ZIP folder and stores its files.
func fetchAndStoreSchedule() {
	// Download the ZIP folder
	resp, err := http.Get(scheduleUrl)
	if err != nil {
		log.Panicf("failed to download schedule from %s: %v", scheduleUrl, err)
	}
	defer resp.Body.Close()

	// Read the ZIP data into memory
	zipData, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panicf("failed to read ZIP data from response: %v", err)
	}

	// Create a ZIP reader
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		log.Panicf("failed to create ZIP reader: %v", err)
	}

	// Store each schedule file
	for _, file := range zipReader.File {
		storeScheduleFile(file)
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
