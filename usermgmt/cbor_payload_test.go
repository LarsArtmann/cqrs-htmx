package usermgmt

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// ADR 0059: the five binary-heavy payloads are emitted CBOR while every other
// event stays JSON. These tests prove the mixed journal decodes through the
// encoding stamps on every shipped read path (fold + read model) and that the
// size win is real.

func cborTestUserStream(t *testing.T) (id.StreamID, []event.Event, []byte, []byte) {
	t.Helper()

	aggID := id.NewStreamID()
	credID := bytes.Repeat([]byte{0xA1}, 32)
	publicKey := bytes.Repeat([]byte{0xC0}, 77)
	secret := bytes.Repeat([]byte{0x5E}, 32)

	steps := []struct {
		name  string
		decid func(UserState, event.Version) ([]event.Event, error)
	}{
		{"register", decideRegisterUser(aggID, "cbor@example.com", "Cbor", []Role{RoleUser})},
		{"add credential", decideAddCredential(aggID, WebAuthnCredential{
			ID:              credID,
			PublicKey:       publicKey,
			AttestationType: "none",
			CreatedAt:       time.Now(),
		})},
		{"enable totp", decideEnableTOTP(aggID, secret)},
		{"record use", decideRecordCredentialUse(aggID, CredentialUse{
			CredentialID: credID,
			SignCount:    1,
			UserPresent:  true,
			UserVerified: true,
		})},
	}

	var (
		state  UserState
		events []event.Event
	)
	for i, step := range steps {
		evts, err := step.decid(state, event.Version(i))
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", step.name, err)
		}
		for _, evt := range evts {
			state, err = foldUser(state, evt)
			if err != nil {
				t.Fatalf("%s: fold: %v", step.name, err)
			}
			events = append(events, evt)
		}
	}

	return aggID, events, credID, secret
}

func TestBinaryPayloadEvents_StampedCBOR(t *testing.T) {
	_, events, _, _ := cborTestUserStream(t)

	wantEncodings := []string{
		"json", // UserRegistered
		"cbor", // CredentialAdded  (ADR 0059)
		"cbor", // TOTPEnabled      (ADR 0059)
		"cbor", // CredentialUsed   (ADR 0059)
	}
	if len(events) != len(wantEncodings) {
		t.Fatalf("got %d events, want %d", len(events), len(wantEncodings))
	}
	for i, want := range wantEncodings {
		if got := string(events[i].Encoding()); got != want {
			t.Errorf("event %d (%s): encoding = %q, want %q", i, events[i].Type(), got, want)
		}
	}

	botID := id.NewStreamID()
	botEvents, err := decideRegisterBot(
		botID,
		"bot",
		NewUserID("owner-1"),
		bytes.Repeat([]byte{0xB7}, 32),
		[]string{"read"},
	)(
		BotState{},
		0,
	)
	if err != nil {
		t.Fatalf("register bot: %v", err)
	}
	if got := string(botEvents[0].Encoding()); got != "cbor" {
		t.Errorf("BotRegistered: encoding = %q, want cbor", got)
	}

	rm := NewUserReadModel()
	if err := rm.handleEvent(rm, botEvents[0]); err != nil {
		t.Fatalf("read model over cbor BotRegistered: %v", err)
	}
}

// TestMixedJournal_DecodesViaEncodingStamps proves the ADR's central safety
// claim: a stream mixing JSON and CBOR events folds and projects correctly,
// with binary fields surviving byte-exact.
func TestMixedJournal_DecodesViaEncodingStamps(t *testing.T) {
	aggID, events, credID, secret := cborTestUserStream(t)

	var state UserState
	for _, evt := range events {
		var err error
		state, err = foldUser(state, evt)
		if err != nil {
			t.Fatalf("fold %s: %v", evt.Type(), err)
		}
	}

	if state.Email != "cbor@example.com" {
		t.Errorf("Email = %q", state.Email)
	}
	if len(state.Credentials) != 1 || !bytes.Equal(state.Credentials[0].ID, credID) {
		t.Errorf("credential ID did not round-trip byte-exact through CBOR")
	}
	if !bytes.Equal(state.TOTPSecret, secret) {
		t.Errorf("TOTP secret did not round-trip byte-exact through CBOR")
	}

	rm := NewUserReadModel()
	for _, evt := range events {
		if err := rm.Handle(context.Background(), evt); err != nil {
			t.Fatalf("read model %s: %v", evt.Type(), err)
		}
	}
	u, ok := rm.users[aggID]
	if !ok {
		t.Fatal("read model has no user after mixed stream")
	}
	if !u.TOTPEnabled || !bytes.Equal(u.TOTPSecret, secret) {
		t.Errorf(
			"read model TOTP did not round-trip: enabled=%v secretMatch=%v",
			u.TOTPEnabled,
			bytes.Equal(u.TOTPSecret, secret),
		)
	}
	if len(u.Credentials) != 1 || !bytes.Equal(u.Credentials[0].ID, credID) {
		t.Errorf("read model credential ID did not round-trip byte-exact through CBOR")
	}

	// The removal leg closes the loop: a CBOR CredentialRemoved must fold too.
	removed, err := decideRemoveCredential(aggID, credID)(state, event.Version(len(events)))
	if err != nil {
		t.Fatalf("remove credential: %v", err)
	}
	state, err = foldUser(state, removed[0])
	if err != nil {
		t.Fatalf("fold CredentialRemoved: %v", err)
	}
	if len(state.Credentials) != 0 {
		t.Errorf("credential still present after CBOR removal: %d", len(state.Credentials))
	}
}

// TestCBORPayloadSizeWin_TOTPEnabled pins the ADR's measurable claim: CBOR is
// materially smaller than base64-inflated JSON for a realistic secret.
func TestCBORPayloadSizeWin_TOTPEnabled(t *testing.T) {
	payload := TOTPEnabledPayload{
		SchemaVersion: currentSchemaVersion,
		Secret:        bytes.Repeat([]byte{0x5E}, 32),
	}

	jsonBytes, err := codec.JSONCodec{}.Encode(payload)
	if err != nil {
		t.Fatalf("json encode: %v", err)
	}
	cborBytes, err := codec.CBORCodec{}.Encode(payload)
	if err != nil {
		t.Fatalf("cbor encode: %v", err)
	}

	t.Logf("TOTPEnabled payload: json=%d bytes, cbor=%d bytes (%.0f%% smaller)",
		len(jsonBytes), len(cborBytes), 100*(1-float64(len(cborBytes))/float64(len(jsonBytes))))

	if len(cborBytes) >= len(jsonBytes) {
		t.Errorf(
			"cbor (%d bytes) not smaller than json (%d bytes) — the ADR 0059 claim failed",
			len(cborBytes),
			len(jsonBytes),
		)
	}

	var decoded TOTPEnabledPayload
	if err := (codec.CBORCodec{}).Decode(cborBytes, &decoded); err != nil {
		t.Fatalf("cbor decode: %v", err)
	}
	if !bytes.Equal(decoded.Secret, payload.Secret) || decoded.SchemaVersion != payload.SchemaVersion {
		t.Errorf("cbor round-trip mismatch")
	}
}
