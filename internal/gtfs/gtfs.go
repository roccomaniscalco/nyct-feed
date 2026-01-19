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

type StoreScheduleParams struct {
	Stops         []db.InsertStopParams
	StopTimes     []db.InsertStopTimeParams
	Trips         []db.InsertTripParams
	Routes        []db.InsertRouteParams
	Calendars     []db.InsertCalendarParams
	CalendarDates []db.InsertCalendarDateParams
}

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
	ctx                 context.Context
	db                  *sql.DB
	ScheduleRefreshRate time.Duration
	RealtimeRefreshRate time.Duration
}

type ClientParams struct {
	Ctx                 context.Context
	Db                  *sql.DB
	ScheduleRefreshRate time.Duration
	RealtimeRefreshRate time.Duration
}

func NewClient(params ClientParams) *Client {
	return &Client{
		ctx:                 params.Ctx,
		db:                  params.Db,
		ScheduleRefreshRate: params.ScheduleRefreshRate,
		RealtimeRefreshRate: params.RealtimeRefreshRate,
	}
}

func (c *Client) SyncSchedule() error {
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
	if err := c.storeSchedule(schedule); err != nil {
		return fmt.Errorf("failed to store schedule: %v", err)
	}

	log.Println("Schedule Synced")
	return nil
}

// fetchSchedule requests a GTFS schedule ZIP folder and returns its zip files.
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

func parseSchedule(files []*zip.File) (*StoreScheduleParams, error) {
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

func (c *Client) storeSchedule(schedule *StoreScheduleParams) error {
	// Start transaction
	tx, err := c.db.BeginTx(c.ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare transaction queries
	txQueries, err := db.Prepare(c.ctx, tx)
	if err != nil {
		return err
	}
	defer txQueries.Close()

	// Clear all tables first (in reverse dependency order)
	if err := txQueries.DeleteCalendarDates(c.ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteStopTimes(c.ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteTrips(c.ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteCalendars(c.ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteRoutes(c.ctx); err != nil {
		return err
	}
	if err := txQueries.DeleteStops(c.ctx); err != nil {
		return err
	}

	// Insert in dependency order
	for _, stop := range schedule.Stops {
		err = txQueries.InsertStop(c.ctx, stop)
		if err != nil {
			return err
		}
	}
	for _, route := range schedule.Routes {
		err = txQueries.InsertRoute(c.ctx, route)
		if err != nil {
			return err
		}
	}
	for _, calendar := range schedule.Calendars {
		err = txQueries.InsertCalendar(c.ctx, calendar)
		if err != nil {
			return err
		}
	}
	for _, trip := range schedule.Trips {
		err = txQueries.InsertTrip(c.ctx, trip)
		if err != nil {
			return err
		}
	}
	for _, stopTime := range schedule.StopTimes {
		err = txQueries.InsertStopTime(c.ctx, stopTime)
		if err != nil {
			return err
		}
	}
	for _, calendarDate := range schedule.CalendarDates {
		err = txQueries.InsertCalendarDate(c.ctx, calendarDate)
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return tx.Commit()
}
