package integration_test

import (
	"bytes"
	"context"
	"slices"
	"testing"
	"time"

	"github.com/larsartmann/cqrs-htmx/usermgmt/v2"
	"github.com/larsartmann/go-cqrs-lite/encryption/v3"
	"github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/storage/memory/v3"
	"github.com/larsartmann/go-cqrs-lite/signing/v3"
)

// 32-byte keys — HMAC minimum is 32 bytes, AES-256-GCM requires exactly 32.
var (
	testSigningKey    = bytes.Repeat([]byte{0x5A}, 32)
	testEncryptionKey = bytes.Repeat([]byte{0xA5}, 32)
)

// encryptedStoreHooks returns SecurityHooks that wrap any event store with the
// given cipher, providing encryption-at-rest.
func encryptedStoreHooks(cipher encryption.EncrypterDecrypter) usermgmt.SecurityHooks {
	return usermgmt.SecurityHooks{
		StoreWrapper: func(s event.Store) (event.Store, error) {
			return encryption.NewEncryptedStore(s, cipher)
		},
	}
}

// seedTestUser registers a user named `name` (deriving id and email) and
// fails the test on error. Returns the new UserID and email for downstream
// assertions. Centralizes the test-user fixture for crypto/signing tests so
// they can't drift in how they seed users. Distinct from registerTestUser
// (which takes a root-module cqrshtmx.UserID for cross-module bridge tests).
func seedTestUser(t *testing.T, svc *usermgmt.Service, name string) (usermgmt.UserID, string) {
	t.Helper()
	id := usermgmt.NewUserID(name)
	email := name + "@example.com"
	if _, err := svc.Register(context.Background(), usermgmt.RegisterRequest{
		ID: id, Email: email,
	}); err != nil {
		t.Fatalf("Register %s: %v", name, err)
	}
	return id, email
}

// TestSigningEncryption_StoreEncryptionAndBusSigning verifies the recommended opt-in
// pattern: transparent encryption-at-rest via StoreWrapper, plus bus-level
// event signing via PublishMiddleware and strict signature verification via
// HandlerMiddleware (RequireSignatureMiddleware rejects any unsigned event).
func TestSigningEncryption_StoreEncryptionAndBusSigning(t *testing.T) {
	signer, err := signing.NewHMAC(testSigningKey)
	if err != nil {
		t.Fatalf("NewHMAC: %v", err)
	}

	cipher, err := encryption.NewAES256GCM(testEncryptionKey)
	if err != nil {
		t.Fatalf("NewAES256GCM: %v", err)
	}

	innerStore := memory.NewMemoryStore()

	// Service A: write path — encrypts at rest, signs in transit.
	svcA, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore: innerStore,
		SecurityHooks: usermgmt.SecurityHooks{
			StoreWrapper: func(s event.Store) (event.Store, error) {
				return encryption.NewEncryptedStore(s, cipher)
			},
			PublishMiddleware: []event.PublishMiddleware{
				signing.SignMiddleware(signer),
			},
			// RequireSignatureMiddleware is strict: it rejects events lacking a
			// valid signature. If SignMiddleware failed to sign, this would error.
			HandlerMiddleware: []event.Middleware{
				signing.RequireSignatureMiddleware(signer),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewService A: %v", err)
	}

	const email = "alice@example.com"
	if _, err := svcA.Register(context.Background(), usermgmt.RegisterRequest{
		ID: usermgmt.NewUserID("alice"), Email: email,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Projections must still work through the crypto layer.
	if _, ok := svcA.ReadModel().FindByEmail(email); !ok {
		t.Fatal("expected alice in read model — projections must work with signing+encryption")
	}

	// Encryption-at-rest: the inner (raw) store must hold ciphertext, not plaintext.
	rawEvents, err := innerStore.ReadAll(context.Background())
	if err != nil {
		t.Fatalf("ReadAll raw store: %v", err)
	}
	assertEventsEncrypted(t, rawEvents, email)

	// Decrypt-on-load: a fresh Service reading from the same encrypted store
	// must reconstruct the user via journal replay (ReadAll decrypts).
	assertReloadsUser(t, innerStore, cipher, email)
}

// assertReloadsUser creates a fresh Service over an already-encrypted store and
// verifies the user reappears in the read model after decrypt-on-load replay.
func assertReloadsUser(
	t *testing.T,
	inner event.Store,
	cipher encryption.EncrypterDecrypter,
	email string,
) {
	t.Helper()
	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		EventStore:    inner,
		SecurityHooks: encryptedStoreHooks(cipher),
	})
	if err != nil {
		t.Fatalf("NewService reload: %v", err)
	}
	t.Cleanup(svc.Stop)
	if !waitForUser(svc, email) {
		t.Fatal("expected user to reappear in read model after decrypt-on-load replay")
	}
}

// assertEventsEncrypted verifies every event carries encryption metadata and
// that the plaintext email did not leak into the persisted (ciphertext) payload.
func assertEventsEncrypted(t *testing.T, events []event.Event, email string) {
	t.Helper()
	if len(events) == 0 {
		t.Fatal("expected events in raw store")
	}
	for i, evt := range events {
		if !encryption.HasEncryption(evt) {
			t.Errorf("event[%d] (%s): expected encryption metadata", i, evt.Type())
		}
		if bytes.Contains(evt.Payload(), []byte(email)) {
			t.Errorf("event[%d] (%s): plaintext email leaked into persisted payload",
				i, evt.Type())
		}
	}
}

// TestSigningEncryption_BusLevelCrypto verifies the alternative opt-in pattern:
// both signing and encryption applied at the bus level (encryption-in-transit),
// with no store wrapper. Projections receive decrypted+verified plaintext.
func TestSigningEncryption_BusLevelCrypto(t *testing.T) {
	signer, err := signing.NewHMAC(testSigningKey)
	if err != nil {
		t.Fatalf("NewHMAC: %v", err)
	}

	enc, err := encryption.NewXChaCha20Poly1305(testEncryptionKey)
	if err != nil {
		t.Fatalf("NewXChaCha20Poly1305: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		SecurityHooks: usermgmt.SecurityHooks{
			// Sign then encrypt on publish (sign the plaintext, then encrypt).
			PublishMiddleware: []event.PublishMiddleware{
				signing.SignMiddleware(signer),
				encryption.EncryptMiddleware(enc),
			},
			// Decrypt then verify on handle (decrypt to plaintext, then verify sig).
			HandlerMiddleware: []event.Middleware{
				encryption.DecryptMiddleware(enc),
				signing.RequireSignatureMiddleware(signer),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, email := seedTestUser(t, svc, "bob")

	// Projections received decrypted+verified plaintext → user must be present.
	if _, ok := svc.ReadModel().FindByEmail(email); !ok {
		t.Fatal("expected bob in read model — decrypt+verify must run before projections")
	}
}

// TestSigningEncryption_AuthzProjectionSurvivesCrypto verifies that the Casbin
// projection (which derives RBAC policies from events) still works when events
// are encrypted at rest — i.e. decrypted events reach the policy projection.
func TestSigningEncryption_AuthzProjectionSurvivesCrypto(t *testing.T) {
	cipher, err := encryption.NewAES256GCM(testEncryptionKey)
	if err != nil {
		t.Fatalf("NewAES256GCM: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		SecurityHooks: encryptedStoreHooks(cipher),
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	uid, _ := seedTestUser(t, svc, "carol")

	// The Casbin projection derives roles from events. If it received ciphertext
	// instead of plaintext, the decode would fail and no roles would be assigned.
	roles, err := svc.Authz().RolesForUser(uid, uid.Get())
	if err != nil {
		t.Fatalf("RolesForUser: %v", err)
	}
	if !slices.Contains(roles, usermgmt.RoleUser) {
		t.Errorf(
			"expected RoleUser among roles for carol, got %v — "+
				"casbin projection must work through encryption",
			roles,
		)
	}
}

// waitForUser polls the read model until the user appears or the context deadline
// expires. This is necessary because StartProjections runs the projection runner
// in a background goroutine — there is no synchronous "replay complete" signal.
// Uses context.WithTimeout for the deadline (not time.After+select) per the
// project's flaky-test anti-pattern guidance.
func waitForUser(svc *usermgmt.Service, email string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for {
		if _, ok := svc.ReadModel().FindByEmail(email); ok {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(20 * time.Millisecond):
		}
	}
}

// TestSigningEncryption_Ed25519AsymmetricSigning verifies that asymmetric
// Ed25519 signing works end-to-end: the service signs with the private key
// and verifies with the public key. This is the pattern for client-device
// signing scenarios where the verifying server doesn't hold the private key.
func TestSigningEncryption_Ed25519AsymmetricSigning(t *testing.T) {
	pubKey, privKey, err := signing.GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("GenerateEd25519KeyPair: %v", err)
	}

	signer, err := signing.NewEd25519(privKey)
	if err != nil {
		t.Fatalf("NewEd25519 signer: %v", err)
	}

	verifier, err := signing.NewEd25519Verifier(pubKey)
	if err != nil {
		t.Fatalf("NewEd25519Verifier: %v", err)
	}

	svc, err := usermgmt.NewService(usermgmt.ServiceConfig{
		SecurityHooks: usermgmt.SecurityHooks{
			PublishMiddleware: []event.PublishMiddleware{
				signing.SignMiddleware(signer),
			},
			HandlerMiddleware: []event.Middleware{
				signing.RequireSignatureMiddleware(verifier),
			},
		},
	})
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	_, email := seedTestUser(t, svc, "dave")

	// If Ed25519 sign+verify failed, RequireSignatureMiddleware would have
	// rejected the event and registration would have errored.
	if _, ok := svc.ReadModel().FindByEmail(email); !ok {
		t.Fatal("expected dave in read model — Ed25519 sign+verify must succeed")
	}
}
