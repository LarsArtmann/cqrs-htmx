package core

import (
	"context"
	"encoding/json"
	"time"

	"github.com/larsartmann/go-cqrs-lite/commandlifecycle/v4"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// Command fate labels — the terminal or attempt-level outcome of a dispatched
// command, derived from commandlifecycle event types.
const (
	// FateRejected: a business rule refused the command (no state changed,
	// no retry, no dead letter).
	FateRejected = "rejected"

	// FateFailed: one processing attempt failed with a retryable error.
	FateFailed = "failed"

	// FateRetried: another attempt was scheduled (Attempt is the 1-indexed
	// retry number).
	FateRetried = "retried"

	// FateDeadLettered: all retry attempts are exhausted.
	FateDeadLettered = "dead-lettered"

	// FateCompleted: the command was processed successfully.
	FateCompleted = "completed"
)

// CommandFate is one row of the command-fates view: what happened to a
// dispatched command, decoded from its command.rejected/failed/retried/
// dead-lettered/completed lifecycle event.
type CommandFate struct {
	// Time is when the fate was recorded (the payload's timestamp).
	Time time.Time

	// CommandType is the type of the command the fate belongs to.
	CommandType string

	// CommandID identifies the command. Payload value when the payload
	// carries one; otherwise the lifecycle stream ID (streams are keyed by
	// command ID, so the fallback is the same identity).
	CommandID string

	// Fate is one of the Fate* labels, or the raw event type for lifecycle
	// events this build does not know (they stay visible rather than
	// silently dropped).
	Fate string

	// Attempt is the 1-indexed attempt or retry number the fate describes
	// (0 when the payload carries none).
	Attempt int

	// Error is the recorded error message ("" for completed/retried).
	Error string

	// Family is the errorfamily classification stamped at rejection time
	// ("" for non-rejection fates).
	Family string

	// ErrorCode is the machine-readable error code when the rejecting error
	// chain declared one.
	ErrorCode string

	// EventID is the lifecycle event's ID, for drill-down into the event
	// browser.
	EventID id.EventID
}

// LoadCommandFates reads the newest command fates from the journal — every
// commandlifecycle event except command.received (arrival is not a fate; it
// would double every row with no new information). The scan pages the whole
// journal keeping a rolling tail (journals read oldest-first, so a zero-cursor
// window is the HEAD — the gotcha the events page already documents), then
// decodes payloads. Fates this build cannot decode (schema drift, or an event
// type newer than this dashboard) stay visible with the raw event type as
// their Fate. The second return value is the total number of fate events in
// the journal; the returned slice is newest-first.
func LoadCommandFates(
	ctx context.Context,
	cfg Config,
	limit int,
) ([]CommandFate, int, error) {
	var (
		fates []CommandFate
		total int
	)

	keep := func(batch []event.Event) error {
		for _, evt := range batch {
			if evt.StreamType() != commandlifecycle.StreamTypeCommandLifecycle {
				continue
			}

			fate, ok := decodeCommandFate(evt)
			if !ok {
				continue
			}

			total++

			fates = append(fates, fate)

			if len(fates) > limit {
				trimmed := make([]CommandFate, 0, limit)
				trimmed = append(trimmed, fates[len(fates)-limit:]...)
				fates = trimmed
			}
		}

		return nil
	}

	if cfg.SeekableJournal != nil {
		var after id.EventID

		for {
			batch, err := cfg.SeekableJournal.ReadFrom(ctx, after, tailScanBatch)
			if err != nil {
				return nil, 0, errorfamily.WrapInfrastructure(err,
					"dashboardui.command_fates.read_failed", "read journal for command fates").
					WithContextAny("limit", limit)
			}

			if err := keep(batch); err != nil {
				return nil, 0, err
			}

			if len(batch) < tailScanBatch {
				break
			}

			next := batch[len(batch)-1].ID()
			if next == after {
				return nil, 0, errorfamily.NewInfrastructure("dashboardui.command_fates.cursor_stalled",
					"journal cursor did not advance while reading command fates")
			}

			after = next
		}
	} else if cfg.Journal != nil {
		all, err := cfg.Journal.ReadAll(ctx)
		if err != nil {
			return nil, 0, errorfamily.WrapInfrastructure(err,
				"dashboardui.command_fates.read_all_failed", "read journal for command fates").
				WithContextAny("limit", limit)
		}

		if err := keep(all); err != nil {
			return nil, 0, err
		}
	}

	// Newest first for display: the scan is oldest-first.
	for i, j := 0, len(fates)-1; i < j; i, j = i+1, j-1 {
		fates[i], fates[j] = fates[j], fates[i]
	}

	return fates, total, nil
}

// decodeCommandFate maps one lifecycle event to a CommandFate row. ok is
// false only for command.received events, which are deliberately not fates.
// A payload that fails to decode still yields a row — with the raw event
// type as the Fate — so schema drift stays visible instead of hiding rows.
func decodeCommandFate(evt event.Event) (CommandFate, bool) {
	row := CommandFate{
		CommandID: evt.StreamID().String(),
		EventID:   evt.ID(),
	}

	switch evt.Type() {
	case commandlifecycle.TypeFailed:
		var p commandlifecycle.FailedPayload
		row.Fate, row.Attempt, row.Error, row.Time = FateFailed, p.Attempt, p.Error, p.FailedAt
		if err := json.Unmarshal(evt.Payload(), &p); err == nil {
			row.CommandType = p.CommandType
			row.Attempt = p.Attempt
			row.Error = p.Error
			row.Time = p.FailedAt

			if p.CommandID != "" {
				row.CommandID = string(p.CommandID)
			}
		}
	case commandlifecycle.TypeRejected:
		var p commandlifecycle.RejectedPayload
		row.Fate = FateRejected
		if err := json.Unmarshal(evt.Payload(), &p); err == nil {
			row.CommandType = p.CommandType
			row.Attempt = p.Attempt
			row.Error = p.Error
			row.Family = p.Family
			row.ErrorCode = p.ErrorCode
			row.Time = p.RejectedAt

			if p.CommandID != "" {
				row.CommandID = string(p.CommandID)
			}
		}
	case commandlifecycle.TypeRetried:
		var p commandlifecycle.RetriedPayload
		row.Fate = FateRetried
		if err := json.Unmarshal(evt.Payload(), &p); err == nil {
			row.CommandType = p.CommandType
			row.Attempt = p.Attempt
			row.Time = p.RetriedAt
		}
	case commandlifecycle.TypeDeadLettered:
		var p commandlifecycle.DeadLetteredPayload
		row.Fate = FateDeadLettered
		if err := json.Unmarshal(evt.Payload(), &p); err == nil {
			row.CommandType = p.CommandType
			row.Error = p.Error
			row.Attempt = p.Attempts
			row.Time = p.DeadLetteredAt
		}
	case commandlifecycle.TypeCompleted:
		var p commandlifecycle.CompletedPayload
		row.Fate = FateCompleted
		if err := json.Unmarshal(evt.Payload(), &p); err == nil {
			row.CommandType = p.CommandType
			row.Time = p.CompletedAt

			if p.CommandID != "" {
				row.CommandID = string(p.CommandID)
			}
		}
	case commandlifecycle.TypeReceived:
		return CommandFate{}, false
	default:
		// An event type newer than this build: keep it visible with the raw
		// type as the fate, so nothing on a CommandLifecycle stream silently
		// disappears from the view.
		row.Fate = string(evt.Type())
		row.CommandType = string(evt.Type())
		row.Time = evt.OccurredAt()
	}

	if row.Time.IsZero() {
		row.Time = evt.OccurredAt()
	}

	return row, true
}
