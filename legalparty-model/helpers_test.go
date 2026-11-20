package legalparty

import (
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/oklog/ulid/v2"
)

// shared fixtures
var (
	testTime  = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	testUserID = identitymodel.NewActorID(identitymodel.ActorUser, ulid.Make().String())
	testBotID  = identitymodel.NewActorID(identitymodel.ActorBot, ulid.Make().String())
	testSysID  = identitymodel.NewActorID(identitymodel.ActorSystem, "seed-import")
	testAnonID = identitymodel.ActorID{}
	testStream = id.NewStreamID()
)

var testAddress = mustAddress()

func mustAddress() RegisteredAddress {
	a, err := NewRegisteredAddress("Kurfürstendamm 11", "Berlin", "10719", "", "DE")
	if err != nil {
		panic(err)
	}

	return a
}

// gmbhProfile is a fully anchored German legal entity (the live GLEIF
// record's shape: GmbH with HRB entry and LEI).
func gmbhProfile() LegalPartyProfile {
	ref, _ := NewRegisterRef("de", "HRB 127525")

	return LegalPartyProfile{
		LegalName:      "ANDHOFF Immobilien GmbH",
		LegalForm:      LegalFormLegalEntity,
		LegalFormCode:  "2HBR",
		Country:        "DE",
		RegisteredAddr: testAddress,
		LEI:            LEI(leiLiveDE),
		Register:       &ref,
	}
}

func freelancerProfile() LegalPartyProfile {
	return LegalPartyProfile{
		LegalName:      "Ada Lovelace e.K.",
		LegalForm:      LegalFormNaturalPerson,
		Country:        "DE",
		RegisteredAddr: testAddress,
	}
}

func openState(profile LegalPartyProfile) LegalPartyState {
	return LegalPartyState{
		Profile:   profile,
		Opened:    true,
		OpenedAt:  testTime,
		LastUpdatedAt: testTime,
	}
}

func decideOpen(t *testing.T, profile LegalPartyProfile, actor identitymodel.ActorID) ([]event.Event, error) {
	t.Helper()

	return DecideOpenLegalParty(testStream, profile, actor, testTime)(LegalPartyState{}, event.Version(0))
}

func requireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func requireErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", want)
	}
	if got := errorCode(t, err); got != want {
		t.Fatalf("error code: got %q, want %q", got, want)
	}
}
