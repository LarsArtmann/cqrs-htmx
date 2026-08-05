package core

import (
	"testing"
)

func TestDetectCapabilities_AllNil(t *testing.T) {
	t.Parallel()

	caps := DetectCapabilities(Config{})

	checks := map[string]bool{
		"EventSource":     caps.EventSource,
		"EventByIDLoader": caps.EventByIDLoader,
		"Journal":         caps.Journal,
		"SeekableJournal": caps.SeekableJournal,
		"StreamReader":    caps.StreamReader,
		"ProjectionHost":  caps.ProjectionHost,
		"DeadLetterStore": caps.DeadLetterStore,
		"CommandJournal":  caps.CommandJournal,
		"QueryJournal":    caps.QueryJournal,
		"SnapshotStore":   caps.SnapshotStore,
		"EventBus":        caps.EventBus,
	}

	for name, val := range checks {
		if val {
			t.Errorf("%s should be false for zero-value Config", name)
		}
	}
}

func TestDetectCapabilities_AllSet(t *testing.T) {
	t.Parallel()

	journal := &fakeSeekableJournal{}
	cfg := Config{
		EventByIDLoader: &fakeEventByIDLoader{},
		Journal:         journal,
		SeekableJournal: journal,
		StreamReader:    &fakeStreamReader{},
		DeadLetterStore: &fakeDeadLetterStore{},
	}

	caps := DetectCapabilities(cfg)

	if !caps.EventByIDLoader {
		t.Error("EventByIDLoader should be true when set")
	}

	if !caps.Journal {
		t.Error("Journal should be true when set")
	}

	if !caps.SeekableJournal {
		t.Error("SeekableJournal should be true when set")
	}

	if !caps.StreamReader {
		t.Error("StreamReader should be true when set")
	}

	if !caps.DeadLetterStore {
		t.Error("DeadLetterStore should be true when set")
	}

	// Fields we didn't set should still be false
	if caps.EventSource {
		t.Error("EventSource should be false when not set")
	}

	if caps.ProjectionHost {
		t.Error("ProjectionHost should be false when not set")
	}
}

func TestHasEventRead(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		caps Capabilities
		want bool
	}{
		{"empty", Capabilities{}, false},
		{"journal only", Capabilities{Journal: true}, true},
		{"seekable only", Capabilities{SeekableJournal: true}, true},
		{"both", Capabilities{Journal: true, SeekableJournal: true}, true},
		{"neither but has other", Capabilities{EventSource: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.caps.HasEventRead(); got != tt.want {
				t.Errorf("HasEventRead() = %v, want %v", got, tt.want)
			}
		})
	}
}
