package legalparty

import (
	"time"

	"github.com/larsartmann/go-codec"
	"github.com/larsartmann/go-cqrs-lite/event/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// LegalPartyOpenedPayload records a legal party entering the billing
// system's view with its initial profile.
type LegalPartyOpenedPayload struct {
	SchemaVersion int               `json:"schema_version"`
	Profile       LegalPartyProfile `json:"profile"`
	OpenedAt      time.Time         `json:"opened_at"`
}

// LegalPartyProfileUpdatedPayload replaces the profile wholesale, optionally
// carrying the provenance-stamped registry facts the update was built from.
type LegalPartyProfileUpdatedPayload struct {
	SchemaVersion int                `json:"schema_version"`
	Profile       LegalPartyProfile  `json:"profile"`
	RegistryFacts *RegistryFacts     `json:"registry_facts,omitempty"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

// LegalPartyClosedPayload records terminal closure. Closure is never a
// deletion: the profile stays in the journal and in the folded state for
// tax retention.
type LegalPartyClosedPayload struct {
	SchemaVersion int       `json:"schema_version"`
	Reason        string    `json:"reason"`
	ClosedAt      time.Time `json:"closed_at"`
}

// UnmarshalPayload decodes an event's payload into T, resolving the codec
// from the event's own encoding (events default to CBOR since go-cqrs-lite
// v4.15.2 — never assume JSON on the decode side).
func UnmarshalPayload[T any](evt event.Event) (T, error) {
	c, err := codec.ForEncoding(evt.Encoding())
	if err != nil {
		return *new(T), errorfamily.WrapCorruption(err,
			"legalparty.payload_decode_failed",
			"resolve codec for encoding "+string(evt.Encoding())+
				" (event "+string(evt.Type())+")")
	}

	var target T
	if err := c.Decode(evt.Payload(), &target); err != nil {
		return target, errorfamily.WrapCorruption(err,
			"legalparty.payload_decode_failed",
			"decode payload for event "+string(evt.Type()))
	}

	return target, nil
}
