package eventtest

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v3"
)

type FakeMetrics struct {
	mu        sync.Mutex
	Records   []string
	Durations []time.Duration
}

func (m *FakeMetrics) Observe(_ context.Context, name string, duration time.Duration, _ ...string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Records = append(m.Records, name)
	m.Durations = append(m.Durations, duration)
}

func AssertCallOrder(t *testing.T, callOrder, expected []string) {
	t.Helper()

	for i, v := range expected {
		if i >= len(callOrder) || callOrder[i] != v {
			t.Errorf("expected call order %v, got %v", expected, callOrder)

			break
		}
	}
}

func AssertMetricRecord(t *testing.T, m *FakeMetrics, wantName string) {
	t.Helper()

	if len(m.Records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(m.Records))
	}

	if m.Records[0] != wantName {
		t.Errorf("expected %s, got %s", wantName, m.Records[0])
	}
}

func assertSliceLen[T any](
	t *testing.T,
	name string,
	slice []T,
	want int,
	fail func(string, ...any),
) {
	t.Helper()

	if len(slice) != want {
		fail("%s = %d, want %d", name, len(slice), want)
	}
}

func AssertLen[T any](t *testing.T, name string, slice []T, want int) {
	t.Helper()
	assertSliceLen(t, name, slice, want, t.Errorf)
}

func AssertLenFatal[T any](t *testing.T, name string, slice []T, want int) {
	t.Helper()
	assertSliceLen(t, name, slice, want, t.Fatalf)
}

func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()

	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

func AssertError(t *testing.T, err error, msg string) {
	t.Helper()

	if err == nil {
		t.Error(msg)
	}
}

func AssertErrorWithResult(t *testing.T, result any, err error, substr string) {
	t.Helper()

	if result != nil && !reflect.ValueOf(result).IsNil() {
		t.Errorf("result = %v, want nil", result)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error = %q, want containing %q", err.Error(), substr)
	}
}

func AssertEqual[T comparable](t *testing.T, got, want T, msg string) {
	t.Helper()

	if got != want {
		t.Errorf("%s: got %v, want %v", msg, got, want)
	}
}

func AssertContains(t *testing.T, s, substr, msg string) {
	t.Helper()

	if !strings.Contains(s, substr) {
		t.Error(msg)
	}
}

func AssertNotContains(t *testing.T, s, substr, msg string) {
	t.Helper()

	if strings.Contains(s, substr) {
		t.Error(msg)
	}
}

func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()

	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("error = %q, want containing %q", err.Error(), substr)
	}
}

func AssertEventType(t *testing.T, events []event.Event, index int, want string) {
	t.Helper()

	if got := string(events[index].Type()); got != want {
		t.Errorf("events[%d].Type = %q, want %s", index, got, want)
	}
}

func AssertEventVersion(t *testing.T, events []event.Event, index, want int) {
	t.Helper()

	if got := int(events[index].Version()); got != want {
		t.Errorf("events[%d].Version = %d, want %d", index, got, want)
	}
}
