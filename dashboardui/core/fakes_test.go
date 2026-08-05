package core

import (
	"context"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-cqrs-lite/listing/v4"
	"github.com/larsartmann/go-cqrs-lite/projectionhost/v4"
)

// --- Test fakes ---

type fakeSeekableJournal struct {
	events  []event.Event
	readErr error
	allErr  error
}

func (f *fakeSeekableJournal) ReadAll(_ context.Context) ([]event.Event, error) {
	if f.allErr != nil {
		return nil, f.allErr
	}

	return f.events, nil
}

func (f *fakeSeekableJournal) ReadFrom(_ context.Context, _ id.EventID, limit int) ([]event.Event, error) {
	if f.readErr != nil {
		return nil, f.readErr
	}

	if limit > 0 && limit < len(f.events) {
		return f.events[:limit], nil
	}

	return f.events, nil
}

type fakeEventByIDLoader struct {
	evt event.Event
	err error
}

func (f *fakeEventByIDLoader) LoadByEventID(_ context.Context, _ id.EventID) (event.Event, error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.evt, nil
}

type fakeStreamReader struct {
	page *listing.Page[listing.StreamListing]
	err  error
}

func (f *fakeStreamReader) List(
	_ context.Context,
	_ listing.ListOptions,
) (*listing.Page[listing.StreamListing], error) {
	if f.err != nil {
		return nil, f.err
	}

	return f.page, nil
}

func (f *fakeStreamReader) ListWithStatus(_ context.Context, _ listing.ListOptions) (*listing.Page[listing.StreamStatus], error) {
	return nil, nil
}

type fakeDeadLetterStore struct {
	entries []projectionhost.DeadLetterEntry
}

func (s *fakeDeadLetterStore) Store(_ context.Context, _ projectionhost.DeadLetterEntry) error {
	return nil
}

func (s *fakeDeadLetterStore) List(_ context.Context, _ string) ([]projectionhost.DeadLetterEntry, error) {
	return s.entries, nil
}

func (s *fakeDeadLetterStore) Delete(_ context.Context, _, _ string) error { return nil }
func (s *fakeDeadLetterStore) Purge(_ context.Context, _ string) error     { return nil }
