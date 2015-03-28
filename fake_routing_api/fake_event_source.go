package fake_routing_api

import "github.com/vito/go-sse/sse"

type FakeEventSource struct {
	events chan sse.Event
	errors chan error
}

func NewFakeEventSource() FakeEventSource {
	events := make(chan sse.Event, 10)
	errors := make(chan error, 10)
	return FakeEventSource{events: events, errors: errors}
}

func (fake *FakeEventSource) Next() (sse.Event, error) {
	select {
	case event := <-fake.events:
		return event, nil
	case err := <-fake.errors:
		return sse.Event{}, err
	}
}

func (fake *FakeEventSource) AddEvent(event sse.Event) {
	fake.events <- event
}

func (fake *FakeEventSource) AddError(err error) {
	fake.errors <- err
}

// func (fake *FakeEventSource) Close() error {
// }
