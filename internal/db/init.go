package db

import (
	"context"
	"database/sql"
	_ "embed"
	"log"

	db "nyct-feed/internal/db/gen"
	"nyct-feed/internal/gtfs"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

func Init(ctx context.Context) (*sql.DB, *db.Queries) {
	database, err := sql.Open("sqlite", "./nyct.db")
	if err != nil {
		log.Fatal(err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=OFF",
		"PRAGMA synchronous=OFF",
		// "PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := database.ExecContext(ctx, pragma); err != nil {
			log.Fatal(err)
		}
	}

	// create tables
	if _, err := database.ExecContext(ctx, schema); err != nil {
		log.Fatal(err)
	}

	queries := db.New(database)

	return database, queries
}

func StoreSchedule(ctx context.Context, database *sql.DB, queries *db.Queries, s *gtfs.Schedule) error {
	// Start transaction
	tx, err := database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	qtx := queries.WithTx(tx)

	// Clear all tables first (in reverse dependency order)
	if err := qtx.DeleteCalendarDates(ctx); err != nil {
		return err
	}
	if err := qtx.DeleteStopTimes(ctx); err != nil {
		return err
	}
	if err := qtx.DeleteTrips(ctx); err != nil {
		return err
	}
	if err := qtx.DeleteCalendars(ctx); err != nil {
		return err
	}
	if err := qtx.DeleteRoutes(ctx); err != nil {
		return err
	}
	if err := qtx.DeleteStops(ctx); err != nil {
		return err
	}

	// Insert stops
	for _, stop := range s.Stops {
		err = qtx.InsertStop(ctx, db.InsertStopParams{
			StopID:        stop.StopId,
			StopName:      stop.StopName,
			StopLat:       stop.StopLat,
			StopLon:       stop.StopLon,
			LocationType:  stop.LocationType,
			ParentStation: sql.NullString{String: stop.ParentStation, Valid: stop.ParentStation != ""},
		})
		if err != nil {
			return err
		}
	}

	// Insert routes
	for _, route := range s.Routes {
		err = qtx.InsertRoute(ctx, db.InsertRouteParams{
			RouteID:        route.RouteId,
			AgencyID:       route.AgencyId,
			RouteShortName: route.RouteShortName,
			RouteLongName:  route.RouteLongName,
			RouteDesc:      route.RouteDesc,
			RouteType:      route.RouteType,
			RouteUrl:       route.RouteUrl,
			RouteColor:     route.RouteColor,
			RouteTextColor: route.RouteTextColor,
			RouteSortOrder: route.RouteSortOrder,
		})
		if err != nil {
			return err
		}
	}

	// Insert calendars
	for _, calendar := range s.Calendars {

		err = qtx.InsertCalendar(ctx, db.InsertCalendarParams{
			ServiceID: calendar.ServiceId,
			Monday:    calendar.Monday,
			Tuesday:   calendar.Tuesday,
			Wednesday: calendar.Wednesday,
			Thursday:  calendar.Thursday,
			Friday:    calendar.Friday,
			Saturday:  calendar.Saturday,
			Sunday:    calendar.Sunday,
			StartDate: calendar.StartDate,
			EndDate:   calendar.EndDate,
		})
		if err != nil {
			return err
		}
	}

	// Insert trips
	for _, trip := range s.Trips {

		err = qtx.InsertTrip(ctx, db.InsertTripParams{
			TripID:       trip.TripId,
			RouteID:      trip.RouteId,
			ServiceID:    trip.ServiceId,
			TripHeadsign: trip.TripHeadsign,
			DirectionID:  trip.DirectionId,
			ShapeID:      trip.ShapeId,
		})
		if err != nil {
			return err
		}
	}

	// Insert stop times
	for _, stopTime := range s.StopTimes {
		err = qtx.InsertStopTime(ctx, db.InsertStopTimeParams{
			TripID:        stopTime.TripId,
			StopID:        stopTime.StopId,
			ArrivalTime:   stopTime.ArrivalTime,
			DepartureTime: stopTime.DepartureTime,
			StopSequence:  stopTime.StopSequence,
		})
		if err != nil {
			return err
		}
	}

	// Insert calendar dates
	for _, calendarDate := range s.CalendarDates {
		err = qtx.InsertCalendarDate(ctx, db.InsertCalendarDateParams{
			ServiceID:     calendarDate.ServiceId,
			Date:          calendarDate.Date,
			ExceptionType: calendarDate.ExceptionType,
		})
		if err != nil {
			return err
		}
	}

	// Commit transaction
	return tx.Commit()
}
