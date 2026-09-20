package transport

import (
	"bytes"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
	"github.com/larsartmann/go-sse"
)

// FuzzDomainEventToSSE proves the envelope builder is total: arbitrary event
// type strings and payload bytes must never panic, must always produce an
// SSE event whose ID matches the domain event (reconnect cursors depend on
// it), and must always encode to a well-formed wire frame.
func FuzzDomainEventToSSE(f *testing.F) {
	seeds := []struct{ eventType, payload string }{
		{"UserRegistered", `{"email":"alice@example.com"}`},
		{"", ""},
		{"weird\nevent\ntype", "multi\nline\npayload"},
		{"emoji-🎨-type", `{"name":"Jürgen","emoji":"🚀"}`},
		{"data-prefix", "data: injected\nretry: 1\nid: fake"},
		{string(bytes.Repeat([]byte("T"), 512)), string(bytes.Repeat([]byte("p"), 4096))},
	}

	for _, s := range seeds {
		f.Add(s.eventType, s.payload)
	}

	f.Fuzz(func(t *testing.T, eventType, payload string) {
		evt, err := event.New(
			event.Type(eventType),
			id.NewStreamID(),
			"fuzz",
			event.Version(1),
			payload,
		)
		if err != nil {
			t.Skip() // invalid constructed events are not the mapper's contract
		}

		mapped := DomainEventToSSE(evt)

		if mapped.ID.Get() != evt.ID().String() {
			t.Fatalf("envelope ID %q must equal the domain event ID %q (reconnect cursor contract)",
				mapped.ID.Get(), evt.ID().String())
		}

		var buf bytes.Buffer
		if err := sse.WriteEvent(&buf, mapped); err != nil {
			t.Fatalf("mapped event must always encode: %v", err)
		}

		if !bytes.Contains(buf.Bytes(), []byte("\n\n")) {
			t.Fatalf("encoded frame must terminate with a blank line:\n%q", buf.String())
		}
	})
}
