package legalparty

import (
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-codec"
)

func TestFoldLifecycle(t *testing.T) {
	state := LegalPartyState{}

	opened, err := DecideOpenLegalParty(testStream, gmbhProfile(), testUserID, testTime)(state, event.Version(0))
	requireNoError(t, err)
	if len(opened) != 1 {
		t.Fatalf("open must emit one event, got %d", len(opened))
	}
	state, err = FoldLegalParty(state, opened[0])
	requireNoError(t, err)
	if !state.IsOpen() || !state.IsValid() {
		t.Fatalf("state after open: open=%v valid=%v", state.IsOpen(), state.IsValid())
	}

	updatedProfile := gmbhProfile()
	updatedProfile.BillingEmail = "billing@andhoff.example"
	facts := &RegistryFacts{
		Source:    RegistrySourceGLEIF,
		LEI:       LEI(leiLiveDE),
		LegalName: "ANDHOFF Immobilien GmbH",
		Status:    "ACTIVE",
		FetchedAt: testTime,
	}
	updated, err := DecideUpdateLegalPartyProfile(testStream, updatedProfile, facts, testUserID, testTime)(
		state, event.Version(1))
	requireNoError(t, err)
	state, err = FoldLegalParty(state, updated[0])
	requireNoError(t, err)
	if state.Profile.BillingEmail != "billing@andhoff.example" || state.LastRegistryFacts == nil {
		t.Fatalf("update did not fold: email=%q facts=%v", state.Profile.BillingEmail, state.LastRegistryFacts)
	}

	closed, err := DecideCloseLegalParty(testStream, "ceased trading", testUserID, testTime)(
		state, event.Version(2))
	requireNoError(t, err)
	state, err = FoldLegalParty(state, closed[0])
	requireNoError(t, err)

	if !state.Closed || state.CloseReason != "ceased trading" {
		t.Fatalf("closure did not fold: %+v", state)
	}
	if !state.IsValid() {
		t.Fatal("closed state must stay valid")
	}

	// Retention: closure never erases the identity behind an issued
	// invoice — profile and provenance survive in the folded state.
	if state.Profile.LegalName != "ANDHOFF Immobilien GmbH" || state.LastRegistryFacts == nil {
		t.Fatalf("retention violated: profile=%+v facts=%v", state.Profile, state.LastRegistryFacts)
	}
}

func TestPayloadRoundTrip_BothEncodings(t *testing.T) {
	buildEvent := func(opts ...event.Option) event.Event {
		evt, err := event.New(
			EventLegalPartyOpened, testStream, AggregateTypeLegalParty, event.Version(0).Increment(),
			LegalPartyOpenedPayload{
				SchemaVersion: CurrentSchemaVersion,
				Profile:       gmbhProfile(),
				OpenedAt:      testTime,
			},
			opts...)
		requireNoError(t, err)

		return evt
	}

	deciders := []struct {
		name string
		opts []event.Option
	}{
		{"json", []event.Option{event.WithCodec(codec.JSONCodec{})}},
		{"cbor", []event.Option{}},
	}

	for _, d := range deciders {
		t.Run(d.name, func(t *testing.T) {
			evt := buildEvent(d.opts...)
			got, err := UnmarshalPayload[LegalPartyOpenedPayload](evt)
			requireNoError(t, err)
			if got.SchemaVersion != CurrentSchemaVersion {
				t.Fatalf("schema version: got %d, want %d", got.SchemaVersion, CurrentSchemaVersion)
			}
			if got.Profile.LegalName != "ANDHOFF Immobilien GmbH" || got.Profile.Register == nil {
				t.Fatalf("profile did not round-trip: %+v", got.Profile)
			}
			if !got.OpenedAt.Equal(testTime) {
				t.Fatalf("timestamp did not round-trip: %v", got.OpenedAt)
			}
		})
	}
}

func TestDecideOpenLegalParty_Guards(t *testing.T) {
	t.Run("happy path by user", func(t *testing.T) {
		events, err := decideOpen(t, gmbhProfile(), testUserID)
		requireNoError(t, err)
		if events[0].Type() != EventLegalPartyOpened {
			t.Fatalf("event type: got %q", events[0].Type())
		}
		if events[0].StreamID() != testStream {
			t.Fatal("event must carry the stream id")
		}
	})

	t.Run("system actor may seed parties", func(t *testing.T) {
		_, err := decideOpen(t, gmbhProfile(), testSysID)
		requireNoError(t, err)
	})

	t.Run("bot actor rejected", func(t *testing.T) {
		_, err := decideOpen(t, gmbhProfile(), testBotID)
		if !errors.Is(err, ErrBotActorForbidden) {
			t.Fatalf("bot actor must hit ErrBotActorForbidden, got %v", err)
		}
	})

	t.Run("unattributed actor rejected", func(t *testing.T) {
		_, err := decideOpen(t, gmbhProfile(), testAnonID)
		if !errors.Is(err, ErrBotActorForbidden) {
			t.Fatalf("zero actor must hit ErrBotActorForbidden, got %v", err)
		}
	})

	t.Run("double open conflicts", func(t *testing.T) {
		_, err := DecideOpenLegalParty(testStream, gmbhProfile(), testUserID, testTime)(
			openState(gmbhProfile()), event.Version(1))
		requireErrorCode(t, err, "legalparty.legal_party.already_exists")
	})

	t.Run("invalid profile rejected", func(t *testing.T) {
		anchored := gmbhProfile()
		anchored.LEI = ""
		anchored.Register = nil
		_, err := decideOpen(t, anchored, testUserID)
		requireErrorCode(t, err, "legalparty.profile.registry_anchor_required")
	})
}

func TestDecideUpdateLegalPartyProfile_Guards(t *testing.T) {
	t.Run("unknown party", func(t *testing.T) {
		_, err := DecideUpdateLegalPartyProfile(testStream, gmbhProfile(), nil, testUserID, testTime)(
			LegalPartyState{}, event.Version(0))
		requireErrorCode(t, err, "legalparty.legal_party.not_found")
	})

	t.Run("closed party", func(t *testing.T) {
		closed := openState(gmbhProfile())
		closed.Closed = true
		closed.CloseReason = "ceased trading"
		closed.ClosedAt = testTime
		_, err := DecideUpdateLegalPartyProfile(testStream, gmbhProfile(), nil, testUserID, testTime)(
			closed, event.Version(1))
		if !errors.Is(err, ErrLegalPartyClosed) {
			t.Fatalf("closed party must hit ErrLegalPartyClosed, got %v", err)
		}
	})

	t.Run("bot actor rejected", func(t *testing.T) {
		_, err := DecideUpdateLegalPartyProfile(testStream, gmbhProfile(), nil, testBotID, testTime)(
			openState(gmbhProfile()), event.Version(1))
		if !errors.Is(err, ErrBotActorForbidden) {
			t.Fatalf("bot actor must hit ErrBotActorForbidden, got %v", err)
		}
	})

	t.Run("invalid facts rejected", func(t *testing.T) {
		bad := &RegistryFacts{Source: "csv-import", LegalName: "x", FetchedAt: testTime}
		_, err := DecideUpdateLegalPartyProfile(testStream, gmbhProfile(), bad, testUserID, testTime)(
			openState(gmbhProfile()), event.Version(1))
		requireErrorCode(t, err, "legalparty.registry_facts.source_invalid")
	})

	t.Run("valid facts ride the event", func(t *testing.T) {
		facts := &RegistryFacts{
			Source:    RegistrySourceOpenCorporates,
			LegalName: "ANDHOFF Immobilien GmbH",
			Status:    "Currently Registered",
			FetchedAt: testTime,
		}
		events, err := DecideUpdateLegalPartyProfile(testStream, gmbhProfile(), facts, testUserID, testTime)(
			openState(gmbhProfile()), event.Version(1))
		requireNoError(t, err)

		p, err := UnmarshalPayload[LegalPartyProfileUpdatedPayload](events[0])
		requireNoError(t, err)
		if p.RegistryFacts == nil || p.RegistryFacts.Source != RegistrySourceOpenCorporates {
			t.Fatalf("facts did not ride the event: %+v", p.RegistryFacts)
		}
	})
}

func TestDecideCloseLegalParty_Guards(t *testing.T) {
	t.Run("unknown party", func(t *testing.T) {
		_, err := DecideCloseLegalParty(testStream, "ceased trading", testUserID, testTime)(
			LegalPartyState{}, event.Version(0))
		requireErrorCode(t, err, "legalparty.legal_party.not_found")
	})

	t.Run("empty reason rejected", func(t *testing.T) {
		_, err := DecideCloseLegalParty(testStream, "  ", testUserID, testTime)(
			openState(gmbhProfile()), event.Version(1))
		requireErrorCode(t, err, "legalparty.legal_party.close_reason_required")
	})

	t.Run("double close conflicts", func(t *testing.T) {
		closed := openState(gmbhProfile())
		closed.Closed = true
		closed.CloseReason = "ceased trading"
		closed.ClosedAt = testTime
		_, err := DecideCloseLegalParty(testStream, "again", testUserID, testTime)(closed, event.Version(2))
		if !errors.Is(err, ErrLegalPartyClosed) {
			t.Fatalf("second close must hit ErrLegalPartyClosed, got %v", err)
		}
	})

	t.Run("bot actor rejected", func(t *testing.T) {
		_, err := DecideCloseLegalParty(testStream, "ceased trading", testBotID, testTime)(
			openState(gmbhProfile()), event.Version(1))
		if !errors.Is(err, ErrBotActorForbidden) {
			t.Fatalf("bot actor must hit ErrBotActorForbidden, got %v", err)
		}
	})

	t.Run("closed event normalizes to UTC", func(t *testing.T) {
		events, err := DecideCloseLegalParty(testStream, "ceased trading", testUserID, testTime)(
			openState(gmbhProfile()), event.Version(1))
		requireNoError(t, err)

		p, err := UnmarshalPayload[LegalPartyClosedPayload](events[0])
		requireNoError(t, err)
		if p.ClosedAt.Location() != time.UTC {
			t.Fatalf("closed at must normalize to UTC: %v", p.ClosedAt.Location())
		}
	})
}

func TestFoldLegalParty_UnknownEvent(t *testing.T) {
	evt, err := event.New(
		"BogusEvent", testStream, AggregateTypeLegalParty, event.Version(0).Increment(),
		struct{ X int `json:"x"` }{X: 1},
		event.WithCodec(codec.JSONCodec{}))
	requireNoError(t, err)

	_, err = FoldLegalParty(LegalPartyState{}, evt)
	requireErrorCode(t, err, "legalparty.legal_party.unknown_event")
}

func TestStateIsValid(t *testing.T) {
	tests := []struct {
		name  string
		state LegalPartyState
		want  bool
	}{
		{"fresh", LegalPartyState{}, true},
		{"open", openState(gmbhProfile()), true},
		{"closed with reason", func() LegalPartyState {
			s := openState(gmbhProfile())
			s.Closed = true
			s.CloseReason = "x"
			s.ClosedAt = testTime

			return s
		}(), true},
		{"closed without reason", func() LegalPartyState {
			s := openState(gmbhProfile())
			s.Closed = true

			return s
		}(), false},
		{"reason without close", func() LegalPartyState {
			s := openState(gmbhProfile())
			s.CloseReason = "x"

			return s
		}(), false},
		{"closed without timestamp", func() LegalPartyState {
			s := openState(gmbhProfile())
			s.Closed = true
			s.CloseReason = "x"

			return s
		}(), false},
		{"unopened with timestamp", LegalPartyState{OpenedAt: testTime}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.state.IsValid(); got != tt.want {
				t.Fatalf("IsValid = %v, want %v", got, tt.want)
			}
		})
	}
}
