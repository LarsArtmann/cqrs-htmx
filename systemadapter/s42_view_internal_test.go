package systemadapter

import (
	"testing"
	"time"

	usermgmt "github.com/larsartmann/cqrs-htmx/usermgmt/v4"
	"github.com/larsartmann/go-cqrs-lite/record/v4"
)

// TestNewAuditEntryView_CarriesActor pins S42c at the fold's data source: the
// record's metadata actor lands in the view as the self-describing wire form,
// and a record without an actor yields an empty ActorID (never a fabricated one).
func TestNewAuditEntryView_CarriesActor(t *testing.T) {
	rec := record.Record{MetaData: record.CommonMetadata{
		Actor: record.Actor{Kind: record.ActorUser, Raw: "01HABCDEFG"},
	}}

	view := newAuditEntryView(rec, "evt", "agg", time.Unix(0, 0), usermgmt.AuditActionRegister)
	if view.ActorID != "user:01HABCDEFG" {
		t.Errorf("ActorID = %q, want %q", view.ActorID, "user:01HABCDEFG")
	}

	zero := newAuditEntryView(record.Record{}, "evt", "agg", time.Unix(0, 0), usermgmt.AuditActionRegister)
	if zero.ActorID != "" {
		t.Errorf("zero-actor record must yield empty ActorID, got %q", zero.ActorID)
	}
}
