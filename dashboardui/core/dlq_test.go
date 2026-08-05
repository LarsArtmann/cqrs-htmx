package core

import (
	"context"
	"testing"
)

func TestDLQProjectionLinks_NilHost(t *testing.T) {
	t.Parallel()

	links := DLQProjectionLinks(context.Background(), Config{})
	if links != nil {
		t.Errorf("DLQProjectionLinks with nil host should return nil, got %d entries", len(links))
	}
}

func TestDLQProjectionLinks_NoHostButConfigured(t *testing.T) {
	t.Parallel()

	// With nil ProjectionHost but DeadLetterStore set, should still return nil
	// (the function checks ProjectionHost first)
	cfg := Config{
		DeadLetterStore: &fakeDeadLetterStore{},
	}

	links := DLQProjectionLinks(context.Background(), cfg)
	if links != nil {
		t.Errorf("should return nil when ProjectionHost is nil, got %d entries", len(links))
	}
}
