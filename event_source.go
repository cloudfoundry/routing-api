package routing_api

import (
	"errors"
	"io"

	"github.com/vito/go-sse/sse"
)

//go:generate counterfeiter -o fake_receptor/fake_event_source.go . EventSource

// EventSource provides sequential access to a stream of events.
type EventSource interface {
	Next() (sse.Event, error)
	// Close() error
}

//go:generate counterfeiter -o fake_receptor/fake_raw_event_source.go . RawEventSource

type RawEventSource struct {
}

type eventSource struct {
	sseEventSource *sse.EventSource
}

func NewEventSource(sseEventSource *sse.EventSource) EventSource {
	return &eventSource{
		sseEventSource: sseEventSource,
	}
}

func (e *eventSource) Next() (sse.Event, error) {
	rawEvent, err := e.sseEventSource.Next()
	if err != nil {
		switch err {
		//make our own errors!
		case io.EOF:
			return sse.Event{}, err

		case sse.ErrSourceClosed:
			return sse.Event{}, errors.New("source closed")

		default:
			return sse.Event{}, errors.New("default")
		}
	}

	return rawEvent, nil
}

// func (e *eventSource) Close() error {
// 	err := e.Close()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }
