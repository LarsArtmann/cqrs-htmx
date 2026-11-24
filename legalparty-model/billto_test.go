package legalparty

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	identitymodel "github.com/larsartmann/cqrs-htmx/identity-model/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

func billToTestParty(n byte) LegalPartyID {
	return NewLegalPartyID("01MBILLTOPARTY0000000000000" + string(rune('0'+n)))
}

var billToTestActor = identitymodel.ActorIDFromUser(identitymodel.NewUserID("01MBILLTOUSER000000000000001")) //nolint:gochecknoglobals // fixed fixture

// TestLegalForm_RoundTrip pins the LegalFormClass vocabulary through JSON:
// every class survives an encode/decode cycle unchanged.
func TestLegalForm_RoundTrip(t *testing.T) {
	for _, class := range []LegalFormClass{
		LegalFormNaturalPerson, LegalFormLegalEntity, LegalFormPartnership,
		LegalFormPublicBody, LegalFormNonProfit,
	} {
		profile := validBillToProfile(t)
		profile.LegalForm = class

		raw, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("marshal %s: %v", class, err)
		}
		var back LegalPartyProfile
		if err := json.Unmarshal(raw, &back); err != nil {
			t.Fatalf("unmarshal %s: %v", class, err)
		}
		if back.LegalForm != class {
			t.Errorf("round-trip turned %s into %s", class, back.LegalForm)
		}
		if err := back.Validate(); err != nil {
			t.Errorf("round-tripped profile no longer validates (%s): %v", class, err)
		}
	}
}

// TestLegalForm_DecodeUnknownValueRejected pins that an unknown legal_form
// string decodes lossily but never validates: the class gate is the last
// line of defence against hand-written payloads.
func TestLegalForm_DecodeUnknownValueRejected(t *testing.T) {
	raw := `{
		"legal_name": "Muster GmbH",
		"legal_form": "corporation",
		"country": "DE",
		"registered_address": {"street": "Hauptstr. 1", "postal_code": "10115", "locality": "Berlin", "country": "DE"}
	}`
	var profile LegalPartyProfile
	if err := json.Unmarshal([]byte(raw), &profile); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if profile.LegalForm.IsValid() {
		t.Fatalf("unknown class %q validates — the gate is broken", profile.LegalForm)
	}
	err := profile.Validate()
	if err == nil || !strings.Contains(err.Error(), "legal_form_invalid") {
		t.Fatalf("Validate = %v, want the legal_form_invalid rejection", err)
	}
}

// TestBillTo_OpenRedirectClose walks the aggregate's full lifecycle through
// the deciders and the fold.
func TestBillTo_OpenRedirectClose(t *testing.T) {
	aggID := DeriveBillToStreamID("tenant-1")
	at := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

	events, err := DecideOpenBillTo(aggID, billToTestParty(1), billToTestActor, at)(
		BillToState{}, 0)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(events) != 1 || events[0].Type() != EventBillToOpened {
		t.Fatalf("open events = %v", eventTypesOf(events))
	}
	state := foldBillToAll(t, BillToState{}, events)
	if !state.Opened || state.LegalPartyID != billToTestParty(1) || state.TenantID != "tenant-1" {
		t.Fatalf("state after open = %+v", state)
	}

	events, err = DecideRedirectBillTo(aggID, billToTestParty(2), "holding restructure", billToTestActor, at)(
		state, 1)
	if err != nil {
		t.Fatalf("redirect: %v", err)
	}
	if len(events) != 1 || events[0].Type() != EventBillToRedirected {
		t.Fatalf("redirect events = %v", eventTypesOf(events))
	}
	state = foldBillToAll(t, state, events)
	if state.LegalPartyID != billToTestParty(2) {
		t.Fatalf("party after redirect = %v", state.LegalPartyID)
	}

	// Redirecting to the party already named is the idempotent no-op.
	events, err = DecideRedirectBillTo(aggID, billToTestParty(2), "again", billToTestActor, at)(
		state, 2)
	if err != nil {
		t.Fatalf("same-party redirect: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("same-party redirect emitted %v", eventTypesOf(events))
	}

	events, err = DecideCloseBillTo(aggID, "tenant churned", billToTestActor, at)(state, 3)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if len(events) != 1 || events[0].Type() != EventBillToClosed {
		t.Fatalf("close events = %v", eventTypesOf(events))
	}
	state = foldBillToAll(t, state, events)
	if !state.Closed || state.CloseReason != "tenant churned" {
		t.Fatalf("state after close = %+v", state)
	}
}

// TestBillTo_Guards covers every refusal: duplicate open, unknown link,
// terminal closure, missing party, missing reasons, and the bot-actor guard.
func TestBillTo_Guards(t *testing.T) {
	aggID := DeriveBillToStreamID("tenant-g")
	at := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	open := DecideOpenBillTo(aggID, billToTestParty(1), billToTestActor, at)
	openEvents, openErr := DecideOpenBillTo(aggID, billToTestParty(1), billToTestActor, at)(BillToState{}, 0)
	opened := foldBillToAll(t, BillToState{}, mustBillToEvents(t, openEvents, openErr))
	closeEvents, closeErr := DecideCloseBillTo(aggID, "done", billToTestActor, at)(opened, 1)
	closed := foldBillToAll(t, opened, mustBillToEvents(t, closeEvents, closeErr))
	bot := identitymodel.ActorIDFromBot(identitymodel.BotID("01MBILLTOBOT0000000000000001"))

	if _, err := open(opened, 1); err == nil || !strings.Contains(err.Error(), "already_exists") {
		t.Errorf("duplicate open = %v, want already_exists", err)
	}
	if _, err := DecideRedirectBillTo(aggID, billToTestParty(2), "x", billToTestActor, at)(
		BillToState{}, 0); err == nil || !strings.Contains(err.Error(), "not_found") {
		t.Errorf("redirect on unopened = %v, want not_found", err)
	}
	if _, err := DecideRedirectBillTo(aggID, billToTestParty(2), "x", billToTestActor, at)(
		closed, 2); err == nil || !strings.Contains(err.Error(), "closed") {
		t.Errorf("redirect on closed = %v, want closed", err)
	}
	if _, err := DecideCloseBillTo(aggID, "again", billToTestActor, at)(closed, 3); err == nil ||
		!strings.Contains(err.Error(), "closed") {
		t.Errorf("double close = %v, want closed", err)
	}
	if _, err := DecideOpenBillTo(aggID, billToTestParty(1), bot, at)(BillToState{}, 0); err == nil ||
		!strings.Contains(err.Error(), "bot") {
		t.Errorf("bot open = %v, want the bot-actor refusal", err)
	}
	if _, err := DecideOpenBillTo(aggID, NewLegalPartyID(""), billToTestActor, at)(
		BillToState{}, 0); err == nil || !strings.Contains(err.Error(), "party_required") {
		t.Errorf("party-less open = %v, want party_required", err)
	}
	if _, err := DecideRedirectBillTo(aggID, billToTestParty(9), "", billToTestActor, at)(
		opened, 1); err == nil || !strings.Contains(err.Error(), "redirect_reason_required") {
		t.Errorf("reason-less redirect = %v, want redirect_reason_required", err)
	}
	if _, err := DecideCloseBillTo(aggID, "  ", billToTestActor, at)(opened, 1); err == nil ||
		!strings.Contains(err.Error(), "close_reason_required") {
		t.Errorf("blank close reason = %v, want close_reason_required", err)
	}
}

func mustBillToEvents(t *testing.T, events []event.Event, err error) []event.Event {
	t.Helper()
	if err != nil {
		t.Fatalf("decide: %v", err)
	}
	return events
}

func foldBillToAll(t *testing.T, start BillToState, events []event.Event) BillToState {
	t.Helper()
	state := start
	for _, evt := range events {
		next, err := FoldBillTo(state, evt)
		if err != nil {
			t.Fatalf("FoldBillTo(%s): %v", evt.Type(), err)
		}
		state = next
	}
	return state
}

func eventTypesOf(events []event.Event) []string {
	types := make([]string, len(events))
	for i, evt := range events {
		types[i] = string(evt.Type())
	}
	return types
}

// validBillToProfile builds the minimal validating profile the round-trip
// test mutates.
func validBillToProfile(t *testing.T) LegalPartyProfile {
	t.Helper()
	raw := `{
		"legal_name": "Muster GmbH",
		"legal_form": "legal_entity",
		"country": "DE",
		"registered_address": {"street": "Hauptstr. 1", "postal_code": "10115", "locality": "Berlin", "country": "DE"}
	}`
	var profile LegalPartyProfile
	if err := json.Unmarshal([]byte(raw), &profile); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}
	if err := profile.Validate(); err != nil {
		t.Fatalf("fixture does not validate: %v", err)
	}
	return profile
}
