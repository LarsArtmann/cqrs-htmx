package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cqrshtmx "github.com/larsartmann/cqrs-htmx/v4"
	errorfamily "github.com/larsartmann/go-error-family"
	gohealth "github.com/larsartmann/go-health"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
)

type fakeProvider struct {
	entries []cqrshtmx.ProjectionStatusEntry
}

func (f fakeProvider) ProjectionStatuses() []cqrshtmx.ProjectionStatusEntry {
	return f.entries
}

func TestCheckError_StatusSemantics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status     string
		wantErr    bool
		wantFamily errorfamily.Family
	}{
		{status: "live", wantErr: false},
		{status: "stopped", wantErr: false},
		{status: "idle", wantErr: true, wantFamily: errorfamily.Transient},
		{status: "running", wantErr: true, wantFamily: errorfamily.Transient},
		{status: "backoff", wantErr: true, wantFamily: errorfamily.Transient},
		{status: "draining", wantErr: true, wantFamily: errorfamily.Transient},
		{status: "failed", wantErr: true, wantFamily: errorfamily.Infrastructure},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()

			entry := cqrshtmx.ProjectionStatusEntry{Name: "user-read-model", Status: tt.status}
			err := checkError(entry)

			if !tt.wantErr {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)

			family := errorfamily.Classify(err)
			require.Equal(t, tt.wantFamily, family, "error: %v", err)
		})
	}
}

func TestCheckError_FailedCarriesLastError(t *testing.T) {
	t.Parallel()

	entry := cqrshtmx.ProjectionStatusEntry{
		Name:      "casbin-projection",
		Status:    "failed",
		LastError: "boom",
	}

	err := checkError(entry)
	require.Error(t, err)
	require.Contains(t, err.Error(), "casbin-projection")
	require.Contains(t, err.Error(), "boom")
}

func TestNewProbe_NilProvider(t *testing.T) {
	t.Parallel()

	probe, err := NewProbe(nil)
	require.Nil(t, probe)
	require.Error(t, err)

	family := errorfamily.Classify(err)
	require.Equal(t, errorfamily.Rejection, family)
}

func TestNewProbe_EvaluatesProjections(t *testing.T) {
	t.Parallel()

	provider := fakeProvider{entries: []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "live"},
		{Name: "casbin-projection", Status: "draining"},
		{Name: "tenant-read-model", Status: "failed", LastError: "disk on fire"},
	}}

	probe, err := NewProbe(provider)
	require.NoError(t, err)

	resp := probe.Evaluate(t.Context())

	require.Equal(t, gohealth.StatusPass, resp.Checks["user-read-model"].Status)
	require.Equal(t, gohealth.StatusWarn, resp.Checks["casbin-projection"].Status)
	require.Equal(t, gohealth.StatusWarn, resp.Checks["tenant-read-model"].Status)
	require.Contains(t, resp.Checks["tenant-read-model"].Error, "disk on fire")

	// Without critical services configured, failures degrade the roll-up
	// (and each check) to warn rather than fail.
	require.Equal(t, gohealth.StatusWarn, resp.Status)
}

func TestNewProbe_CriticalProjectionFailsOverall(t *testing.T) {
	t.Parallel()

	provider := fakeProvider{entries: []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "live"},
		{Name: "casbin-projection", Status: "failed", LastError: "boom"},
	}}

	probe, err := NewProbe(provider, gohealth.WithCriticalServices("casbin-projection"))
	require.NoError(t, err)

	resp := probe.Evaluate(t.Context())
	require.Equal(t, gohealth.StatusFail, resp.Checks["casbin-projection"].Status)
	require.Equal(t, gohealth.StatusFail, resp.Status)
}

func TestNewProbe_EmptyProviderPasses(t *testing.T) {
	t.Parallel()

	probe, err := NewProbe(fakeProvider{})
	require.NoError(t, err)

	resp := probe.Evaluate(t.Context())
	require.Equal(t, gohealth.StatusPass, resp.Status)
	require.Empty(t, resp.Checks)
}

type stubHealthchecker struct {
	err error
}

func (s stubHealthchecker) HealthCheck() error { return s.err }

type staticRecorder struct {
	results map[string]error
}

func (s staticRecorder) RecordHealthCheckWithContext(
	_ context.Context,
	_ do.Injector,
) map[string]error {
	return s.results
}

type slowProvider struct {
	entries []cqrshtmx.ProjectionStatusEntry
	delay   time.Duration
}

func (s slowProvider) ProjectionStatuses() []cqrshtmx.ProjectionStatusEntry {
	time.Sleep(s.delay)

	return s.entries
}

func TestRecorderChain_LaterRecorderWinsNameCollisions(t *testing.T) {
	t.Parallel()

	first := staticRecorder{results: map[string]error{
		"user-read-model": nil,
		"shared":          errors.New("first"),
	}}
	second := staticRecorder{results: map[string]error{
		"shared": errors.New("second"),
	}}

	chain := RecorderChain(first, second)
	results := chain.RecordHealthCheckWithContext(t.Context(), nil)

	require.Len(t, results, 2)
	require.NoError(t, results["user-read-model"])
	require.EqualError(t, results["shared"], "second")
}

func TestRecorderChain_SkipsNilRecorders(t *testing.T) {
	t.Parallel()

	recorder := staticRecorder{results: map[string]error{"user-read-model": nil}}

	chain := RecorderChain(nil, recorder, nil)
	results := chain.RecordHealthCheckWithContext(t.Context(), nil)

	require.Len(t, results, 1)
	require.Contains(t, results, "user-read-model")
}

func TestRecorderChain_EmptyChainReportsNoChecks(t *testing.T) {
	t.Parallel()

	chain := RecorderChain()
	require.Empty(t, chain.RecordHealthCheckWithContext(t.Context(), nil))

	allNil := RecorderChain(nil, nil)
	require.Empty(t, allNil.RecordHealthCheckWithContext(t.Context(), nil))
}

func TestNewProbe_CarriesDurationsOnTheWire(t *testing.T) {
	t.Parallel()

	provider := slowProvider{
		entries: []cqrshtmx.ProjectionStatusEntry{
			{Name: "user-read-model", Status: "live"},
			{Name: "casbin-projection", Status: "draining"},
		},
		delay: 2 * time.Millisecond,
	}

	probe, err := NewProbe(provider)
	require.NoError(t, err)

	resp := probe.Evaluate(t.Context())

	for name, check := range resp.Checks {
		require.Positive(t, check.DurationNanos, "check %q should carry a duration", name)
	}
}

func TestRecorder_DetailedDurationsFlowThroughWithHealthRecorder(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideValue(injector, stubHealthchecker{err: nil})

	provider := slowProvider{
		entries: []cqrshtmx.ProjectionStatusEntry{
			{Name: "user-read-model", Status: "live"},
		},
		delay: 2 * time.Millisecond,
	}

	probe := gohealth.New(injector, gohealth.WithHealthRecorder(Recorder(provider)))
	resp := probe.Evaluate(t.Context())

	require.Contains(t, resp.Checks, "user-read-model")
	require.Positive(t,
		resp.Checks["user-read-model"].DurationNanos,
		"detailed recorder should report the batch read time")
}

func TestRecorderChain_ComposesInOneProbe(t *testing.T) {
	t.Parallel()

	provider := fakeProvider{entries: []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "live"},
	}}
	plugin := staticRecorder{results: map[string]error{"audit-log": nil}}

	probe := gohealth.New(do.New(), gohealth.WithHealthRecorder(
		RecorderChain(Recorder(provider), plugin),
	))
	resp := probe.Evaluate(t.Context())

	require.Contains(t, resp.Checks, "user-read-model")
	require.Contains(t, resp.Checks, "audit-log")
	require.Equal(t, gohealth.StatusPass, resp.Status)
}

func TestRecorder_MergesInjectorChecks(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideValue(injector, stubHealthchecker{err: nil})

	provider := fakeProvider{entries: []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "live"},
	}}

	recorder := Recorder(provider)
	results := recorder.RecordHealthCheckWithContext(t.Context(), injector)

	require.Contains(t, results, "user-read-model")
	require.NoError(t, results["user-read-model"])

	// The injector's own healthchecker service is included in the batch.
	// do derives the check name from the service type; find it by elimination.
	var injectorChecks int

	for name, err := range results {
		if name == "user-read-model" {
			continue
		}

		injectorChecks++

		require.NoError(t, err)
	}

	require.Equal(t, 1, injectorChecks, "injector service check missing: %v", results)
}

func TestRecorder_FailingInjectorService(t *testing.T) {
	t.Parallel()

	injector := do.New()
	do.ProvideValue(injector, stubHealthchecker{
		err: errors.New("db unreachable"),
	})

	recorder := Recorder(fakeProvider{})
	results := recorder.RecordHealthCheckWithContext(t.Context(), injector)

	require.Len(t, results, 1)

	for _, err := range results {
		require.EqualError(t, err, "db unreachable")
	}
}

func TestRecorder_NilInjector(t *testing.T) {
	t.Parallel()

	recorder := Recorder(fakeProvider{entries: []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "live"},
	}})

	results := recorder.RecordHealthCheckWithContext(t.Context(), nil)
	require.Len(t, results, 1)
	require.NoError(t, results["user-read-model"])
}

func TestNewDashboard_ServesJSON(t *testing.T) {
	t.Parallel()

	provider := fakeProvider{entries: []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "live"},
	}}

	probe, err := NewProbe(provider)
	require.NoError(t, err)

	dash := NewDashboard(probe)
	require.NotNil(t, dash)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health/ui", nil)
	r.Header.Set("Accept", "application/json")
	dash.Handler()(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(t, w.Body.String(), `"status"`)
}

func TestNewDashboard_RendersHTML(t *testing.T) {
	t.Parallel()

	provider := fakeProvider{entries: []cqrshtmx.ProjectionStatusEntry{
		{Name: "user-read-model", Status: "live"},
	}}

	probe, err := NewProbe(provider, gohealth.WithRefreshInterval(50*time.Millisecond))
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	require.NoError(t, probe.Start(ctx))
	t.Cleanup(probe.Shutdown)

	dash := NewDashboard(probe)

	// The dashboard serves the probe's cached response, so wait for the
	// first refresh to populate it before asserting.
	deadline := time.Now().Add(5 * time.Second)

	for {
		cached := probe.CachedResponse()
		if _, ok := cached.Checks["user-read-model"]; ok || time.Now().After(deadline) {
			require.Contains(t, cached.Checks, "user-read-model")

			break
		}

		time.Sleep(10 * time.Millisecond)
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/health/ui", nil)
	dash.Handler()(w, r)

	require.Equal(t, http.StatusOK, w.Code, "body: %s", w.Body.String())
	require.Contains(t, w.Body.String(), "Health Dashboard")
	require.Contains(t, w.Body.String(), "user-read-model", "projection check should render")
}
