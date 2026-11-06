package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// lifecycleEvent builds one event on a CommandLifecycle stream, the shape
// commandlifecycle.Recorder appends (stream keyed by command ID).
func lifecycleEvent(
	t *testing.T,
	eventType event.Type,
	commandID id.StreamID,
	version int,
	payload any,
) event.Event {
	t.Helper()

	evt, err := event.New(
		eventType,
		commandID,
		commandlifecycle.StreamTypeCommandLifecycle,
		event.Version(version),
		payload,
	)
	if err != nil {
		t.Fatalf("event.New: %v", err)
	}

	return evt
}

// commandFatesFixture is the full fate vocabulary on three command streams:
// cmd-1 received+completed (one fate row), cmd-2 received+failed+retried+
// failed+dead-lettered (three fate rows), cmd-3 rejected (one row). Journal
// order is oldest-first; the newest-first expectation below is reversed.
//
// The streams are derived with id.StreamIDFrom, exactly as
// commandlifecycle.LifecycleStreamRef does, so the fixture has the real
// production shape: the stream's bare value is the command's display identity.
func commandFatesFixture(t *testing.T, base time.Time) ([]event.Event, map[string]id.CommandID) {
	t.Helper()

	cmd1, cmd2, cmd3 := id.NewCommandID(), id.NewCommandID(), id.NewCommandID()
	stream1, stream2, stream3 := id.StreamIDFrom(cmd1), id.StreamIDFrom(cmd2), id.StreamIDFrom(cmd3)

	payloads := []struct {
		eventType event.Type
		commandID id.StreamID
		version   int
		payload   any
	}{
		{commandlifecycle.TypeReceived, stream1, 1, commandlifecycle.ReceivedPayload{
			CommandID: commandlifecycle.CommandKey(cmd1.String()), CommandType: "RegisterUser",
			ReceivedAt: base,
		}},
		{commandlifecycle.TypeCompleted, stream1, 2, commandlifecycle.CompletedPayload{
			CommandID: commandlifecycle.CommandKey(cmd1.String()), CommandType: "RegisterUser",
			CompletedAt: base.Add(1 * time.Minute),
		}},
		{commandlifecycle.TypeReceived, stream2, 1, commandlifecycle.ReceivedPayload{
			CommandID: commandlifecycle.CommandKey(cmd2.String()), CommandType: "SendEmail",
			ReceivedAt: base.Add(2 * time.Minute),
		}},
		{commandlifecycle.TypeFailed, stream2, 2, commandlifecycle.FailedPayload{
			CommandID: commandlifecycle.CommandKey(cmd2.String()), CommandType: "SendEmail",
			Error: "smtp flap", Attempt: 1, FailedAt: base.Add(3 * time.Minute),
		}},
		{commandlifecycle.TypeRetried, stream2, 3, commandlifecycle.RetriedPayload{
			CommandType: "SendEmail", Attempt: 1, RetriedAt: base.Add(4 * time.Minute),
		}},
		{commandlifecycle.TypeDeadLettered, stream2, 4, commandlifecycle.DeadLetteredPayload{
			CommandType: "SendEmail", Error: "smtp flap", Attempts: 2,
			DeadLetteredAt: base.Add(5 * time.Minute),
		}},
		{commandlifecycle.TypeRejected, stream3, 1, commandlifecycle.RejectedPayload{
			CommandID: commandlifecycle.CommandKey(cmd3.String()), CommandType: "InviteMember",
			Error: "email already invited", Family: "rejection", ErrorCode: "member.duplicate",
			Attempt: 1, RejectedAt: base.Add(6 * time.Minute),
		}},
	}

	events := make([]event.Event, 0, len(payloads))
	for _, p := range payloads {
		events = append(events, lifecycleEvent(t, p.eventType, p.commandID, p.version, p.payload))
	}

	return events, map[string]id.CommandID{"cmd1": cmd1, "cmd2": cmd2, "cmd3": cmd3}
}

// TestLoadCommandFates_DecodesAllFates skips command.received, decodes the
// five fate payloads, counts fates only, and returns them newest-first.
func TestLoadCommandFates_DecodesAllFates(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 11, 6, 4, 0, 0, 0, time.UTC)
	events, cmds := commandFatesFixture(t, base)

	// One unrelated domain event must not leak into the view.
	domainEvt, _ := event.New("user.created", id.NewStreamID(), "User", event.Version(1), struct{}{})
	events = append(events, domainEvt)

	cfg := Config{SeekableJournal: &fakeSeekableJournal{events: events}}

	fates, total, err := LoadCommandFates(context.Background(), cfg, 50)
	if err != nil {
		t.Fatalf("LoadCommandFates: %v", err)
	}

	if total != 5 {
		t.Errorf("total = %d, want 5 (received and domain events are not fates)", total)
	}

	if len(fates) != 5 {
		t.Fatalf("len(fates) = %d, want 5", len(fates))
	}

	// Newest-first: rejected (cmd3) … completed (cmd1) last.
	wantFates := []struct {
		fate        string
		commandType string
		commandID   id.CommandID
		attempt     int
		errMsg      string
		family      string
		errorCode   string
		time        time.Time
	}{
		{
			FateRejected,
			"InviteMember",
			cmds["cmd3"],
			1,
			"email already invited",
			"rejection",
			"member.duplicate",
			base.Add(6 * time.Minute),
		},
		{FateDeadLettered, "SendEmail", cmds["cmd2"], 2, "smtp flap", "", "", base.Add(5 * time.Minute)},
		{FateRetried, "SendEmail", cmds["cmd2"], 1, "", "", "", base.Add(4 * time.Minute)},
		{FateFailed, "SendEmail", cmds["cmd2"], 1, "smtp flap", "", "", base.Add(3 * time.Minute)},
		{FateCompleted, "RegisterUser", cmds["cmd1"], 0, "", "", "", base.Add(1 * time.Minute)},
	}

	for i, want := range wantFates {
		got := fates[i]

		if got.Fate != want.fate {
			t.Errorf("fates[%d].Fate = %q, want %q", i, got.Fate, want.fate)
		}

		if got.CommandType != want.commandType {
			t.Errorf("fates[%d].CommandType = %q, want %q", i, got.CommandType, want.commandType)
		}

		if got.CommandID != want.commandID.String() {
			t.Errorf("fates[%d].CommandID = %q, want %q", i, got.CommandID, want.commandID.String())
		}

		if got.Attempt != want.attempt {
			t.Errorf("fates[%d].Attempt = %d, want %d", i, got.Attempt, want.attempt)
		}

		if got.Error != want.errMsg {
			t.Errorf("fates[%d].Error = %q, want %q", i, got.Error, want.errMsg)
		}

		if got.Family != want.family {
			t.Errorf("fates[%d].Family = %q, want %q", i, got.Family, want.family)
		}

		if got.ErrorCode != want.errorCode {
			t.Errorf("fates[%d].ErrorCode = %q, want %q", i, got.ErrorCode, want.errorCode)
		}

		if !got.Time.Equal(want.time) {
			t.Errorf("fates[%d].Time = %v, want %v", i, got.Time, want.time)
		}

		if got.EventID == (id.EventID{}) {
			t.Errorf("fates[%d].EventID must carry the lifecycle event ID for drill-down", i)
		}
	}
}

// TestLoadCommandFates_LimitKeepsNewest pins the rolling tail: a limit
// smaller than the fate count keeps the NEWEST fates, not the head.
func TestLoadCommandFates_LimitKeepsNewest(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 11, 6, 4, 0, 0, 0, time.UTC)
	events, _ := commandFatesFixture(t, base)

	fates, total, err := LoadCommandFates(
		context.Background(),
		Config{SeekableJournal: &fakeSeekableJournal{events: events}},
		2,
	)
	if err != nil {
		t.Fatalf("LoadCommandFates: %v", err)
	}

	if total != 5 {
		t.Errorf("total = %d, want 5 (the limit windows the view, not the count)", total)
	}

	if len(fates) != 2 {
		t.Fatalf("len(fates) = %d, want 2", len(fates))
	}

	if fates[0].Fate != FateRejected || fates[1].Fate != FateDeadLettered {
		t.Errorf("limit must keep the newest fates, got [%s, %s]", fates[0].Fate, fates[1].Fate)
	}
}

// TestLoadCommandFates_CommandIDFallsBackToStream pins the pre-v4.4 shape:
// FailedPayload without a CommandID still identifies the command via the
// lifecycle stream ID (streams are keyed by command ID).
//
// The fallback must print the SAME string a payload-carrying event prints:
// LifecycleStreamRef derives the stream from the command ID, so its bare
// value is the command's display identity. A StreamID display form here would
// re-brand the same command twice in one view (gotcha 25).
func TestLoadCommandFates_CommandIDFallsBackToStream(t *testing.T) {
	t.Parallel()

	cmdID := id.NewCommandID()
	stream := id.StreamIDFrom(cmdID)

	events := []event.Event{
		lifecycleEvent(t, commandlifecycle.TypeFailed, stream, 1, commandlifecycle.FailedPayload{
			CommandType: "LegacyCommand", Error: "boom", Attempt: 3,
			FailedAt: time.Date(2026, 11, 6, 4, 0, 0, 0, time.UTC),
		}),
	}

	fates, _, err := LoadCommandFates(
		context.Background(),
		Config{Journal: &fakeSeekableJournal{events: events}},
		10,
	)
	if err != nil {
		t.Fatalf("LoadCommandFates: %v", err)
	}

	if len(fates) != 1 {
		t.Fatalf("len(fates) = %d, want 1", len(fates))
	}

	if fates[0].CommandID != cmdID.String() {
		t.Errorf("CommandID = %q, want the command identity %q", fates[0].CommandID, cmdID.String())
	}

	if fates[0].CommandID != stream.Get() {
		t.Errorf("CommandID = %q, want the stream's bare value %q", fates[0].CommandID, stream.Get())
	}

	if fates[0].Attempt != 3 {
		t.Errorf("Attempt = %d, want 3", fates[0].Attempt)
	}
}

// TestLoadCommandFates_UnknownTypeStaysVisible pins honesty under schema
// drift: a lifecycle event type this build does not know renders with the
// raw type as its fate instead of silently disappearing.
func TestLoadCommandFates_UnknownTypeStaysVisible(t *testing.T) {
	t.Parallel()

	events := []event.Event{
		lifecycleEvent(t, event.Type("command.escalated"), id.NewStreamID(), 1, struct{}{}),
	}

	fates, total, err := LoadCommandFates(
		context.Background(),
		Config{Journal: &fakeSeekableJournal{events: events}},
		10,
	)
	if err != nil {
		t.Fatalf("LoadCommandFates: %v", err)
	}

	if total != 1 || len(fates) != 1 {
		t.Fatalf("unknown lifecycle types must stay visible, got total=%d rows=%d", total, len(fates))
	}

	if fates[0].Fate != "command.escalated" {
		t.Errorf("Fate = %q, want the raw event type", fates[0].Fate)
	}
}

// TestLoadCommandFates_NoJournalReturnsZero and read errors surface as
// infrastructure failures — the panel's two degenerate inputs.
func TestLoadCommandFates_NoJournalReturnsZero(t *testing.T) {
	t.Parallel()

	fates, total, err := LoadCommandFates(context.Background(), Config{}, 10)
	if err != nil {
		t.Fatalf("LoadCommandFates with no journal: %v", err)
	}

	if fates != nil || total != 0 {
		t.Errorf("no journal must yield no rows and no error, got %d rows total=%d", len(fates), total)
	}
}

func TestLoadCommandFates_ReadErrorFails(t *testing.T) {
	t.Parallel()

	readErr := errors.New("boom")

	if _, _, err := LoadCommandFates(
		context.Background(),
		Config{SeekableJournal: &fakeSeekableJournal{readErr: readErr}},
		10,
	); err == nil {
		t.Fatal("seekable read failure must surface as an error")
	}

	if _, _, err := LoadCommandFates(
		context.Background(),
		Config{Journal: &fakeSeekableJournal{allErr: readErr}},
		10,
	); err == nil {
		t.Fatal("ReadAll failure must surface as an error")
	}
}
