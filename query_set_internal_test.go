package cqrshtmx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	errorfamily "github.com/larsartmann/go-error-family"
)

// The apply adapter erases the loader's value type so one set can hold
// heterogeneous loaders. These white-box specs pin both of its contracts: the
// nil result an interface-typed loader can produce, and the type mismatch no
// constructor can.

// applyProbeProps records what the consumer's apply function received.
type applyProbeProps struct {
	count int
	calls int
}

type probeStringer struct{}

func (probeStringer) String() string { return "probe" }

// TestLoad_NilInterfaceResultAppliesTheZeroValue: a loader whose interface-typed
// V comes back nil applies V's zero value, exactly like a typed nil pointer,
// instead of skipping the apply function.
func TestLoad_NilInterfaceResultAppliesTheZeroValue(t *testing.T) {
	t.Parallel()

	var (
		got   fmt.Stringer = probeStringer{}
		calls int
	)

	loader := Load("maybe", func(context.Context) (fmt.Stringer, error) { return nil, nil },
		func(_ *applyProbeProps, value fmt.Stringer) {
			got = value
			calls++
		})

	value, err := loader.run(context.Background())
	if err != nil {
		t.Fatalf("run = %v, want nil", err)
	}

	if err := loader.apply(&applyProbeProps{}, value); err != nil {
		t.Fatalf("apply = %v, want nil", err)
	}

	if calls != 1 || got != nil {
		t.Fatalf("apply was not called with V's zero value: calls=%d value=%#v", calls, got)
	}
}

// TestLoad_TypedNilPointerIsPassedThrough: a typed nil pointer is a non-nil
// interface value and reaches apply unchanged.
func TestLoad_TypedNilPointerIsPassedThrough(t *testing.T) {
	t.Parallel()

	var (
		got   = &probeStringer{}
		calls int
	)

	loader := Load("maybe", func(context.Context) (*probeStringer, error) { return nil, nil },
		func(_ *applyProbeProps, value *probeStringer) {
			got = value
			calls++
		})

	value, err := loader.run(context.Background())
	if err != nil {
		t.Fatalf("run = %v, want nil", err)
	}

	if value == nil {
		t.Fatal("run returned a nil interface for a typed nil pointer; apply would see the zero-value path")
	}

	if err := loader.apply(&applyProbeProps{}, value); err != nil {
		t.Fatalf("apply = %v, want nil", err)
	}

	if calls != 1 || got != nil {
		t.Fatalf("typed nil did not reach apply: calls=%d value=%#v", calls, got)
	}
}

// TestQuerySet_ReportsAnApplyTypeMismatchInsteadOfSkipping: a slot value the
// apply function cannot accept is reported as a classified error naming the
// loader; it is never dropped silently.
func TestQuerySet_ReportsAnApplyTypeMismatchInsteadOfSkipping(t *testing.T) {
	t.Parallel()

	loader := Load("count", func(context.Context) (int, error) { return 7, nil },
		func(dst *applyProbeProps, value int) {
			dst.count = value
			dst.calls++
		})

	// A wiring fault no constructor can produce: force the type-erased slot to
	// carry another type.
	loader.run = func(context.Context) (any, error) { return "not an int", nil }

	set := QuerySet[applyProbeProps]{Loaders: []Loader[applyProbeProps]{loader}}

	props, _, err := set.load(context.Background(), true)
	if err == nil {
		t.Fatal("load returned nil for an apply type mismatch, want a classified error")
	}

	if code := errorfamily.Code(err); code != "cqrshtmx.queryset.apply_failed" {
		t.Errorf("code = %q, want %q", code, "cqrshtmx.queryset.apply_failed")
	}

	if !errors.Is(err, httperr.ErrDecodeFailed) {
		t.Errorf("errors.Is(err, ErrDecodeFailed) = false, want the decode cause on the chain: %v", err)
	}

	if !strings.Contains(err.Error(), "loader count") {
		t.Errorf("error does not name the loader: %v", err)
	}

	if props.calls != 0 {
		t.Error("apply ran for a value of the wrong type")
	}
}
