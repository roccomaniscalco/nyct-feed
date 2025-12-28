package query

import (
	"time"
)

type Status int

const (
	Pending Status = iota
	Error
	Success
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "Pending"
	case Error:
		return "Error"
	case Success:
		return "Success"
	default:
		return "Unknown"
	}
}

type FetchStatus int

const (
	Fetching FetchStatus = iota
	Idle
)

func (fs FetchStatus) String() string {
	switch fs {
	case Fetching:
		return "Fetching"
	case Idle:
		return "Idle"
	default:
		return "Unknown"
	}
}

type Query[TData any] struct {
	Data          TData
	DataUpdatedAt time.Time
	Status        Status
	FetchStatus   FetchStatus
}

type QueryOptions[TData any] struct {
	QueryChannel    chan Query[TData]
	RefetchInterval time.Duration
	QueryFn[TData]
}

type QueryFn[TData any] func() (TData, error)

func CreateQuery[TData any](options QueryOptions[TData]) chan struct{} {
	var q = Query[TData]{}
	ticker := time.NewTicker(options.RefetchInterval)
	quit := make(chan struct{})

	go func() {
		defer ticker.Stop()

		// Execute immediately
		executeQuery(&q, options)

		for {
			select {
			case <-ticker.C:
				executeQuery(&q, options)
			case <-quit:
				return
			}
		}
	}()

	return quit // Return quit channel so caller can stop the query
}

func executeQuery[TData any](q *Query[TData], options QueryOptions[TData]) {
	// Send update before invoking queryFn
	q.FetchStatus = Fetching
	if q.DataUpdatedAt.IsZero() {
		q.Status = Pending
	}
	options.QueryChannel <- *q

	// Invoke queryFn
	data, err := options.QueryFn()

	// Send resulting update after invoking queryFn
	q.FetchStatus = Idle
	if err != nil {
		q.Status = Error
	} else {
		q.Status = Success
		q.DataUpdatedAt = time.Now()
		q.Data = data
	}
	options.QueryChannel <- *q
}
