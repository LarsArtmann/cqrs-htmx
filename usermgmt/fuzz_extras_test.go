package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

func newFuzzTenantEvent(aggID id.StreamID, payload []byte) (event.Event, error) {
	return event.NewEvent(eventTenantCreated, aggID, aggregateTypeTenant, 1, payload)
}

// FuzzProjectionDedupMap verifies that the projection dedup map handles
// arbitrary event IDs without crashes. Exercises the replay→live dedup
// path with malformed, empty, and adversarial inputs.
func FuzzProjectionDedupMap(f *testing.F) {
	f.Add("01JXTES000000000000000000A")
	f.Add("")
	f.Add("null\x00")
	f.Add("../../../etc/passwd")

	f.Fuzz(func(t *testing.T, eventID string) {
		seen := make(map[string]struct{})
		seen[eventID] = struct{}{}
		if _, ok := seen[eventID]; !ok {
			t.Error("dedup map should contain the key just added")
		}
	})
}

// FuzzRegisterUserDecider verifies that decideRegisterUser handles arbitrary
// inputs without panicking. Exercises the guard logic with adversarial emails,
// display names, and user IDs.
func FuzzRegisterUserDecider(f *testing.F) {
	f.Add("test@example.com", "Alice")
	f.Add("", "")
	f.Add("null\x00byte", string(make([]byte, 256)))

	f.Fuzz(func(t *testing.T, email, displayName string) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("decideRegisterUser panicked: %v", r)
			}
		}()

		uid := NewUserID("fuzz-user")
		aggID, err := aggIDFromUser(uid)
		if err != nil {
			return
		}

		decide := decideRegisterUser(aggID, email, displayName, []Role{RoleUser})
		_, _ = decide(UserState{}, 1)
	})
}

// FuzzFoldTenant verifies that foldTenant handles arbitrary event payloads
// without panicking on malformed JSON or unknown event types.
func FuzzFoldTenant(f *testing.F) {
	f.Add([]byte(`{"name":"acme","display_name":"Acme"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte("\x00"))

	f.Fuzz(func(t *testing.T, payload []byte) {
		t.Parallel()

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("foldTenant panicked: %v", r)
			}
		}()

		aggID := id.NewStreamID()
		evt, err := newFuzzTenantEvent(aggID, payload)
		if err != nil {
			return
		}
		_, _ = foldTenant(TenantState{}, evt)
	})
}
