package gtfs

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"nyct-feed/internal/csvutil"
	"nyct-feed/internal/db"
)

const scheduleUrl = "https://rrgtfsfeeds.s3.amazonaws.com/gtfs_supplemented.zip"

var realtimeUrls = [8]string{
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-ace",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-bdfm",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-g",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-jz",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-nqrw",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-l",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs",
	"https://api-endpoint.mta.info/Dataservice/mtagtfsfeeds/nyct%2Fgtfs-si",
}

type Client struct {
	Ctx                 context.Context
	DB                  *sql.DB
	ScheduleRefreshRate time.Duration
	RealtimeRefreshRate time.Duration
}

func RegisterClient(c Client) {
	c.syncSchedule()
}

type schedule struct {
	Stops         []db.InsertStopParams
	StopTimes     []db.InsertStopTimeParams
	Trips         []db.InsertTripParams
	Routes        []db.InsertRouteParams
	Calendars     []db.InsertCalendarParams
	CalendarDates []db.InsertCalendarDateParams
}

func (c *Client) syncSchedule() error {
	log.Println("Fetching Schedule...")
	files, err := fetchSchedule(scheduleUrl)
	if err != nil {
		return fmt.Errorf("failed to fetch schedule: %v", err)
	}

	log.Println("Parsing Schedule...")
	schedule, err := parseSchedule(files)
	if err != nil {
		return fmt.Errorf("failed to parse schedule: %v", err)
	}

	log.Println("Storing Schedule...")
	if err := storeSchedule(c.DB, c.Ctx, schedule); err != nil {
		return fmt.Errorf("failed to store schedule: %v", err)
	}

	log.Println("Schedule Synced")
	return nil
}

// fetchSchedule requests a GTFS schedule ZIP folder and returns its files.
func fetchSchedule(url string) ([]*zip.File, error) {
	// Download the ZIP folder
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download schedule from %s: %v", url, err)
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

// parseSchedule parses GTFS files into a struct of CSV data.
func parseSchedule(files []*zip.File) (*schedule, error) {
	schedule := StoreScheduleParams{}

	for _, file := range files {

		rc, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("failed to open zip file %s: %v", file.Name, err)
		}
		defer rc.Close()

		switch file.Name {
		case "stops.txt":
			schedule.Stops, err = csvutil.ReadAllParsed(rc, db.InsertStopParams{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse Stops: %v", err)
			}
		case "stop_times.txt":
			schedule.StopTimes, err = csvutil.ReadAllParsed(rc, db.InsertStopTimeParams{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse StopTimes: %v", err)
			}
		case "trips.txt":
			schedule.Trips, err = csvutil.ReadAllParsed(rc, db.InsertTripParams{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse Trips: %v", err)
			}
		case "routes.txt":
			schedule.Routes, err = csvutil.ReadAllParsed(rc, db.InsertRouteParams{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse Routes: %v", err)
			}
		case "calendar.txt":
			schedule.Calendars, err = csvutil.ReadAllParsed(rc, db.InsertCalendarParams{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse Calendars: %v", err)
			}
		case "calendar_dates.txt":
			schedule.CalendarDates, err = csvutil.ReadAllParsed(rc, db.InsertCalendarDateParams{})
			if err != nil {
				return nil, fmt.Errorf("failed to parse CalendarDates: %v", err)
			}
		}

	}

	return &schedule, nil
}

// storeSchedule wipes all schedule tables and batch uploads the new schedule in a transaction.
func storeSchedule(DB *sql.DB, ctx context.Context, s *schedule) error {
	// Start transaction
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare transaction queries
	txQueries, err := db.Prepare(ctx, tx)
	if err != nil {
		return err
	}
	defer txQueries.Close()

	// Clear all tables first (in reverse dependency order)
	if err := txQueries.DeleteCalendarDates(ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteStopTimes(ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteTrips(ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteCalendars(ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteRoutes(ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteStops(ctx); err != nil {
		return err
	}

	// Insert in dependency order
	for _, stop := range s.Stops {
		err = txQueries.InsertStop(ctx, stop)
		if err != nil {
			return err
		}
	}
	for _, route := range s.Routes {
		err = txQueries.InsertRoute(ctx, route)
		if err != nil {
			return err
		}
	}
	for _, calendar := range s.Calendars {
		err = txQueries.InsertCalendar(ctx, calendar)
		if err != nil {
			return err
		}
	}
	for _, trip := range s.Trips {
		err = txQueries.InsertTrip(ctx, trip)
		if err != nil {
			return err
		}
	}
	for _, stopTime := range s.StopTimes {
		err = txQueries.InsertStopTime(ctx, stopTime)
		if err != nil {
			return err
		}
	}
	for _, calendarDate := range s.CalendarDates {
		err = txQueries.InsertCalendarDate(ctx, calendarDate)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return tx.Commit()
}
