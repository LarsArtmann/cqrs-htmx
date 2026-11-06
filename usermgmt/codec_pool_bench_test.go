package usermgmt

import (
	"testing"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// discardSink consumes encoded bytes immediately — the consumption shape
// EncodePooled exists for (the bytes are processed then dropped).
type discardSink struct{ n int }

func (s *discardSink) write(p []byte) error {
	s.n += len(p)

	return nil
}

// BenchmarkCBOREncodePlainVsPooled is the A20 codec-pooling spike: the exact
// seam upstream event.New uses (event_new.go: c.Encode(payload)) versus
// codec.EncodePooled with immediate consumption. It decides the fleet-audit
// "pooling adjunct" adoption question for cqrs-htmx — the verdict is
// recorded in the CHANGELOG entry; this bench stays as the receipt.
func BenchmarkCBOREncodePlainVsPooled(b *testing.B) {
	payload := CredentialAddedPayload{
		SchemaVersion:   currentSchemaVersion,
		ID:              []byte("credential-id-01JXAMPLE00EXAMPLE00"),
		PublicKey:       make([]byte, 128), // realistic WebAuthn COSE key length
		AttestationType: "none",
		Transports:      []string{"internal", "hybrid"},
		AAGUID:          make([]byte, 16),
		SignCount:       42,
		BackupEligible:  true,
		BackupState:     false,
		Name:            "Passkey",
	}

	c := codec.CBORCodec{}
	sink := &discardSink{}

	b.Run("Plain", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			data, err := c.Encode(payload)
			if err != nil {
				b.Fatalf("encode: %v", err)
			}

			if err := sink.write(data); err != nil {
				b.Fatalf("sink: %v", err)
			}
		}
	})

	b.Run("Pooled", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			if err := codec.EncodePooled(c, payload, sink.write); err != nil {
				b.Fatalf("encode pooled: %v", err)
			}
		}
	})
}

// BenchmarkEventNew_CredentialAdded measures the REAL encode hot path — the
// whole event.New call (which internally plain-Encodes the CBOR payload, the
// seam the pooled comparison above isolates). Numbers here contextualize the
// spike; the machine-pinned bench-spike gate does not track this bench.
func BenchmarkEventNew_CredentialAdded(b *testing.B) {
	aggID := id.NewStreamID()

	b.ReportAllocs()

	for b.Loop() {
		evt, err := event.New(
			eventCredentialAdded, aggID, aggregateTypeUser, 1,
			CredentialAddedPayload{
				SchemaVersion:   currentSchemaVersion,
				ID:              []byte("credential-id-01JXAMPLE00EXAMPLE00"),
				PublicKey:       make([]byte, 128),
				AttestationType: "none",
				Transports:      []string{"internal", "hybrid"},
				AAGUID:          make([]byte, 16),
				SignCount:       42,
				BackupEligible:  true,
				BackupState:     false,
				Name:            "Passkey",
			},
			event.WithCodec(codec.CBORCodec{}), event.WithSchemaVersion(1),
		)
		if err != nil {
			b.Fatalf("event.New: %v", err)
		}

		_ = evt
	}
}
