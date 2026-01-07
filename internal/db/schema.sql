-- Table for Stop struct
-- Maps to: Stop struct in internal/gtfs/schedule.go
CREATE TABLE IF NOT EXISTS stops (
    stop_id TEXT PRIMARY KEY,
    stop_name TEXT NOT NULL,
    stop_lat REAL NOT NULL,
    stop_lon REAL NOT NULL,
    location_type INTEGER NOT NULL DEFAULT 0 CHECK (location_type IN (0, 1)), -- 0 = Platform, 1 = Parent Station
    parent_station TEXT REFERENCES stops(stop_id) -- Null if stop is the parent station
);

-- Table for StopTime struct
-- Maps to: StopTime struct in internal/gtfs/schedule.go
CREATE TABLE IF NOT EXISTS stop_times (
    trip_id TEXT NOT NULL REFERENCES trips(trip_id),
    stop_id TEXT NOT NULL REFERENCES stops(stop_id),
    arrival_time TEXT NOT NULL,
    departure_time TEXT NOT NULL,
    stop_sequence INTEGER NOT NULL,
    PRIMARY KEY (trip_id, stop_sequence)
);

-- Table for Trip struct
-- Maps to: Trip struct in internal/gtfs/schedule.go
CREATE TABLE IF NOT EXISTS trips (
    trip_id TEXT PRIMARY KEY,
    route_id TEXT NOT NULL REFERENCES routes(route_id),
    service_id TEXT NOT NULL REFERENCES calendars(service_id),
    trip_headsign TEXT NOT NULL,
    direction_id INTEGER NOT NULL CHECK (direction_id IN (0, 1)),
    shape_id TEXT NOT NULL
);

-- Table for Route struct
-- Maps to: Route struct in internal/gtfs/schedule.go
CREATE TABLE IF NOT EXISTS routes (
    route_id TEXT PRIMARY KEY,
    agency_id TEXT NOT NULL,
    route_short_name TEXT NOT NULL,
    route_long_name TEXT NOT NULL,
    route_desc TEXT NOT NULL,
    route_type INTEGER NOT NULL,
    route_url TEXT NOT NULL,
    route_color TEXT NOT NULL,
    route_text_color TEXT NOT NULL,
    route_sort_order INTEGER NOT NULL
);

-- Table for Calendar struct
-- Maps to: Calendar struct in internal/gtfs/schedule.go
CREATE TABLE IF NOT EXISTS calendars (
    service_id TEXT PRIMARY KEY,
    monday INTEGER NOT NULL CHECK (monday IN (0, 1)),
    tuesday INTEGER NOT NULL CHECK (tuesday IN (0, 1)),
    wednesday INTEGER NOT NULL CHECK (wednesday IN (0, 1)),
    thursday INTEGER NOT NULL CHECK (thursday IN (0, 1)),
    friday INTEGER NOT NULL CHECK (friday IN (0, 1)),
    saturday INTEGER NOT NULL CHECK (saturday IN (0, 1)),
    sunday INTEGER NOT NULL CHECK (sunday IN (0, 1)),
    start_date TEXT NOT NULL,
    end_date TEXT NOT NULL
);

-- Table for CalendarDate struct
-- Maps to: CalendarDate struct in internal/gtfs/schedule.go
CREATE TABLE IF NOT EXISTS calendar_dates (
    service_id TEXT NOT NULL REFERENCES calendars(service_id),
    date TEXT NOT NULL,
    exception_type INTEGER NOT NULL CHECK (exception_type IN (1, 2)), -- 1 = Added, 2 = Cancelled
    PRIMARY KEY (service_id, date)
);

-- Create indexes for commonly queried fields
CREATE INDEX IF NOT EXISTS idx_stops_parent_station ON stops(parent_station);
CREATE INDEX IF NOT EXISTS idx_stops_location_type ON stops(location_type);
CREATE INDEX IF NOT EXISTS idx_stop_times_trip_id ON stop_times(trip_id);
CREATE INDEX IF NOT EXISTS idx_stop_times_stop_id ON stop_times(stop_id);
CREATE INDEX IF NOT EXISTS idx_trips_route_id ON trips(route_id);
CREATE INDEX IF NOT EXISTS idx_trips_service_id ON trips(service_id);
CREATE INDEX IF NOT EXISTS idx_calendar_dates_service_id ON calendar_dates(service_id);
CREATE INDEX IF NOT EXISTS idx_calendar_dates_date ON calendar_dates(date);
