-- name: InsertStop :exec
INSERT INTO stops (stop_id, stop_name, stop_lat, stop_lon, location_type, parent_station)
VALUES (?, ?, ?, ?, ?, ?);

-- name: InsertStopTime :exec
INSERT INTO stop_times (trip_id, stop_id, arrival_time, departure_time, stop_sequence)
VALUES (?, ?, ?, ?, ?);

-- name: InsertTrip :exec
INSERT INTO trips (trip_id, route_id, service_id, trip_headsign, direction_id, shape_id)
VALUES (?, ?, ?, ?, ?, ?);

-- name: InsertRoute :exec
INSERT INTO routes (route_id, agency_id, route_short_name, route_long_name, route_desc, route_type, route_url, route_color, route_text_color, route_sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertCalendar :exec
INSERT INTO calendars (service_id, monday, tuesday, wednesday, thursday, friday, saturday, sunday, start_date, end_date)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertCalendarDate :exec
INSERT INTO calendar_dates (service_id, date, exception_type)
VALUES (?, ?, ?);

-- name: DeleteCalendarDates :exec
DELETE FROM calendar_dates;

-- name: DeleteStopTimes :exec
DELETE FROM stop_times;

-- name: DeleteTrips :exec
DELETE FROM trips;

-- name: DeleteCalendars :exec
DELETE FROM calendars;

-- name: DeleteRoutes :exec
DELETE FROM routes;

-- name: DeleteStops :exec
DELETE FROM stops;