package cqrshtmx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
	errorfamily "github.com/larsartmann/go-error-family"
)

// LoadFunc is the read half of a [Loader]: it produces one value for the props
// struct. A loader is a plain named function on purpose — it does not need a
// query dispatcher, so the same QuerySet serves consumers that dispatch queries
// and consumers that read a service directly. [LoadQuery] adapts a dispatched
// query into this shape when you want the per-query authz posture.
type LoadFunc[V any] func(ctx context.Context) (V, error)

// Loader describes one named read that contributes fields to a props struct T.
//
// Construct it with [Load] (or [LoadQuery]); the zero value is not usable.
type Loader[T any] struct {
	name     string
	optional bool
	pageOnly bool
	run      func(ctx context.Context) (any, error)
	apply    func(*T, any) error
}

// Load declares a loader named name whose function produces V and whose apply
// function writes that value into the props struct. The name is the identity
// used by [LoadReport] and by duplicate detection; it is not sent to any
// authorization engine (that is [LoadQuery]'s job).
//
//	cqrshtmx.Load("ListTenants", listVisibleTenants,
//	    func(props *tenantsProps, tenants []Tenant) { props.Rows = tenants })
func Load[T any, V any](name string, fn LoadFunc[V], apply func(*T, V)) Loader[T] {
	loader := Loader[T]{name: name}

	if fn != nil {
		loader.run = func(ctx context.Context) (any, error) {
			return fn(ctx)
		}
	}

	loader.apply = applier(apply)

	return loader
}

// Optional marks the loader as degradable: when it fails the request still
// renders, the props field stays at its zero value, and the failure is
// recorded in [LoadReport]. Without it a loader failure aborts the request
// with the loader's own error.
func (l Loader[T]) Optional() Loader[T] {
	l.optional = true

	return l
}

// PageOnly marks the loader as needed only by the full-page view: a targeted
// HTMX request (a region or out-of-band render) skips it entirely. See
// [LoadScopeFor] for the exact rule.
func (l Loader[T]) PageOnly() Loader[T] {
	l.pageOnly = true

	return l
}

// LoadQuery adapts a dispatched query into a loader. dispatch runs the query
// (wrap a *query.Dispatcher with [FromDispatcher]); build produces the query
// value for this request. The result is asserted to V, so a mismatched result
// type is a rejection rather than a silent zero field.
//
// This is the adapter that keeps the per-query posture: the dispatcher's gate
// still answers for `query/<Name>`, so a loader can never grant more than the
// query itself may read.
func LoadQuery[T any, V any](
	name string,
	dispatch DispatchFunc,
	build func(ctx context.Context) (query.Query, error),
	apply func(*T, V),
) Loader[T] {
	loader := Loader[T]{name: name}

	if dispatch != nil && build != nil {
		loader.run = func(ctx context.Context) (any, error) {
			qry, err := build(ctx)
			if err != nil {
				return nil, err
			}

			if qry == nil {
				return nil, errorfamily.NewRejection("query_build_returned_nil",
					"loader "+name+" built a nil query")
			}

			enrichQueryFromContext(ctx, qry)

			result, err := dispatch(ctx, qry)
			if err != nil {
				return nil, err
			}

			typed, ok := result.(V)
			if !ok {
				return nil, errorfamily.NewRejection("unexpected_result_type",
					fmt.Sprintf("loader %s: unexpected result type %T", name, result)).
					WithCause(httperr.ErrDecodeFailed)
			}

			return typed, nil
		}
	}

	loader.apply = applier(apply)

	return loader
}

// DispatchFunc runs one query and returns its result. *query.Dispatcher
// satisfies it through [FromDispatcher]; a consumer that owns its own dispatch
// path can pass any function with this shape.
type DispatchFunc func(ctx context.Context, qry query.Query) (any, error)

// FromDispatcher adapts a *query.Dispatcher into a [DispatchFunc]. A nil
// dispatcher yields nil, so LoadQuery's nil guard rejects the loader.
func FromDispatcher(dispatcher *query.Dispatcher) DispatchFunc {
	if dispatcher == nil {
		return nil
	}

	return func(ctx context.Context, qry query.Query) (any, error) {
		return dispatcher.Dispatch(ctx, qry)
	}
}

// applier adapts the consumer's typed apply function to the type-erased slot a
// heterogeneous loader set needs. A nil apply stays nil, so the load pass can
// skip it.
func applier[T, V any](apply func(*T, V)) func(*T, any) error {
	if apply == nil {
		return nil
	}

	return func(dst *T, value any) error {
		return applyLoaded(dst, value, apply)
	}
}

// applyLoaded hands one loader's result to the consumer's apply function.
//
// A nil result (a loader whose interface-typed V came back nil) applies as V's
// zero value, exactly like a typed nil pointer would, so a pointer-typed and an
// interface-typed loader behave the same. A value of any other type cannot be
// produced by [Load] or [LoadQuery] — both hand the function's own V through —
// so it is reported as a wiring fault instead of being skipped.
func applyLoaded[T, V any](dst *T, value any, apply func(*T, V)) error {
	if value == nil {
		var zero V

		apply(dst, zero)

		return nil
	}

	typed, ok := value.(V)
	if !ok {
		return errorfamily.NewRejection("unexpected_loader_value",
			fmt.Sprintf("loader produced %T, which its apply function does not accept", value)).
			WithCause(httperr.ErrDecodeFailed)
	}

	apply(dst, typed)

	return nil
}

// LoadReport is the honest account of one QuerySet run: which loaders ran,
// which were skipped by PageOnly, and which optional loaders failed. It
// carries no payloads, so it is safe to log and safe to hand to a UI.
type LoadReport struct {
	Ran      []string
	Skipped  []string
	Failures []LoadFailure
}

// LoadFailure names one optional loader that failed.
type LoadFailure struct {
	Name string
	Err  error
}

// HasFailures reports whether any optional loader failed.
func (r LoadReport) HasFailures() bool {
	return len(r.Failures) > 0
}

// QuerySet is one props struct built from a set of loaders. Loaders run in
// parallel, each writes its own value, and the props struct is assembled by a
// single goroutine after the join — so the props type never needs a mutex and
// two loaders can never race on a field.
//
// The loaders are exported so a set can be declared as a package-level value:
//
//	var tenants = cqrshtmx.QuerySet[tenantsProps]{
//	    Loaders: []cqrshtmx.Loader[tenantsProps]{
//	        cqrshtmx.Load("ListTenants", listTenants, applyRows),
//	        cqrshtmx.Load("TenantStatusCounts", countStatuses, applyCounts).PageOnly(),
//	    },
//	}
type QuerySet[T any] struct {
	Loaders []Loader[T]
}

// NewQuerySet is the variadic spelling of the struct literal.
func NewQuerySet[T any](loaders ...Loader[T]) QuerySet[T] {
	return QuerySet[T]{Loaders: loaders}
}

// validate rejects a misdeclared set at registration time: an empty set, an
// unnamed loader, a loader without a function, or two loaders sharing a name
// (the name is the report identity, so duplicates would be indistinguishable).
func (s QuerySet[T]) validate() {
	if len(s.Loaders) == 0 {
		//cqrs-lint:ignore(C009) programmer error: an empty QuerySet would render zero props silently
		panic("cqrs-htmx: QuerySet requires at least one Loader")
	}

	seen := make(map[string]struct{}, len(s.Loaders))

	for _, loader := range s.Loaders {
		if loader.name == "" {
			//cqrs-lint:ignore(C009) programmer error: unnamed loader cannot be reported
			panic("cqrs-htmx: QuerySet loader requires a non-empty name")
		}

		if loader.run == nil {
			//cqrs-lint:ignore(C009) programmer error: a loader without a function reads nothing
			panic("cqrs-htmx: QuerySet loader " + loader.name + " has no load function")
		}

		if _, duplicate := seen[loader.name]; duplicate {
			//cqrs-lint:ignore(C009) programmer error: duplicate loader names make the report ambiguous
			panic("cqrs-htmx: QuerySet loader name declared twice: " + loader.name)
		}

		seen[loader.name] = struct{}{}
	}
}

// load runs the loaders the scope needs and assembles the props struct.
//
// Error policy: a failure from a non-optional loader aborts the request. When
// several loaders fail at once the winner is the first failing loader in
// DECLARATION order, never completion order — a completion-order winner would
// make the reported error a flake. Optional failures are reported, not fatal.
func (s QuerySet[T]) load(
	ctx context.Context,
	includePageOnly bool,
) (T, LoadReport, error) {
	var props T

	runs := make([]loaderRun[T], 0, len(s.Loaders))
	for _, loader := range s.Loaders {
		runs = append(runs, loaderRun[T]{loader: loader, slot: slotState{}})
	}

	report := LoadReport{Ran: make([]string, 0, len(runs))}

	cause := runLoaders(ctx, runs, &report, includePageOnly)
	collectFailures(runs, &report)

	if fatal := firstFatalError(runs, cause); fatal != nil {
		return props, report, fatal
	}

	// Apply phase: one goroutine writes the props, so no field can race.
	for _, run := range runs {
		if !run.slot.ran || run.slot.err != nil || run.loader.apply == nil {
			continue
		}

		if err := run.loader.apply(&props, run.slot.value); err != nil {
			return props, report, errorfamily.WrapRejectionf(err,
				"cqrshtmx.queryset.apply_failed", "loader %s", run.loader.name)
		}
	}

	return props, report, nil
}

// loaderRun pairs one loader with the slot its goroutine fills. Pointers into
// the slice are what the goroutines receive, so no loader ever needs to index
// a parallel array (the classic out-of-bounds shape).
type loaderRun[T any] struct {
	loader Loader[T]
	slot   slotState
}

// runLoaders starts every loader the scope needs, waits for the join, and
// returns the cancellation cause (non-nil when a fatal loader failed).
func runLoaders[T any](
	ctx context.Context,
	runs []loaderRun[T],
	report *LoadReport,
	includePageOnly bool,
) error {
	runCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	var wg sync.WaitGroup

	for index := range runs {
		run := &runs[index]

		if !includePageOnly && run.loader.pageOnly {
			report.Skipped = append(report.Skipped, run.loader.name)

			continue
		}

		run.slot.ran = true

		report.Ran = append(report.Ran, run.loader.name)

		wg.Add(1)

		go func(run *loaderRun[T]) {
			defer wg.Done()

			value, err := run.loader.run(runCtx)
			run.slot.value = value
			run.slot.err = err

			// Fail fast: the first non-optional failure cancels every sibling
			// still in flight. The reported error is chosen after the join, in
			// declaration order, so cancellation artifacts never win.
			if err != nil && !run.loader.optional {
				cancel(err)
			}
		}(run)
	}

	wg.Wait()

	if len(report.Skipped) == 0 {
		report.Skipped = nil
	}

	return context.Cause(runCtx) //nolint:wrapcheck // cause is the loader's own error, relayed verbatim
}

// collectFailures records the optional-loader failures; fatal ones are picked
// later by firstFatalError so the report never claims a request survived.
func collectFailures[T any](runs []loaderRun[T], report *LoadReport) {
	for _, run := range runs {
		if !run.loader.optional || !run.slot.ran || run.slot.err == nil {
			continue
		}

		report.Failures = append(report.Failures, LoadFailure{
			Name: run.loader.name,
			Err:  run.slot.err,
		})
	}
}

// firstFatalError picks the reported failure: the first non-optional loader
// failure in declaration order, skipping the context.Canceled artifacts that
// our own fail-fast cancellation produced in the siblings.
func firstFatalError[T any](runs []loaderRun[T], cause error) error {
	for _, run := range runs {
		if run.loader.optional || !run.slot.ran || run.slot.err == nil {
			continue
		}

		if isCancellationArtifact(run.slot.err, cause) {
			continue
		}

		return run.slot.err
	}

	return nil
}

// slotState is the per-loader outcome QuerySet.load collects.
type slotState struct {
	ran   bool
	value any
	err   error
}

// isCancellationArtifact reports whether err is the context.Canceled that our
// own fail-fast cancellation injected into a sibling loader. The loader whose
// error CAUSED the cancellation keeps its error: cause is that exact value —
// so a loader that returns context.Canceled on its own, with nothing else
// failing, is reported as the fatal error rather than swallowed.
func isCancellationArtifact(err, cause error) bool {
	if cause == nil || !errors.Is(err, context.Canceled) {
		return false
	}

	return !errors.Is(cause, err)
}

// OnLoadError installs the hook called when optional loaders failed. Without
// it the failures are logged at WARN with the loader names, so a degrading
// read never becomes a silent one.
func OnLoadError(fn func(r *http.Request, report LoadReport)) HandlerOption {
	return func(config *handlerConfig) {
		config.onLoadError = fn
	}
}

// OnLoad installs a hook that receives the account of every QuerySet load
// that did not abort — which loaders ran, which PageOnly skipped — so a
// consumer can export the cost model to metrics. It also fires on a degraded
// run (optional loaders failed); [OnLoadError] remains the failure-focused
// hook. A load that aborts never reaches it: a fatal loader error goes to the
// error path and to [OnError].
func OnLoad(fn func(r *http.Request, report LoadReport)) HandlerOption {
	return func(config *handlerConfig) {
		config.onLoad = fn
	}
}

// reportLoad hands the run's account to the consumer hooks: OnLoadError when
// optional loaders failed (logged at WARN instead when no hook is installed),
// then OnLoad on every non-fatal run.
func reportLoad(r *http.Request, config *handlerConfig, report LoadReport) {
	if report.HasFailures() {
		if config.onLoadError != nil {
			config.onLoadError(r, report)
		} else {
			logOptionalFailures(r, report)
		}
	}

	if config.onLoad != nil {
		config.onLoad(r, report)
	}
}

func logOptionalFailures(r *http.Request, report LoadReport) {
	names := make([]string, 0, len(report.Failures))

	for _, failure := range report.Failures {
		names = append(names, failure.Name)
		errorfamily.LogErrorContext(r.Context(), failure.Err,
			slog.Default().With(slog.String("path", r.URL.Path), slog.String("loader", failure.Name)))
	}

	slog.Default().Warn("cqrshtmx: optional loaders degraded",
		slog.String("path", r.URL.Path),
		slog.String("loaders", strings.Join(names, ",")))
}

// QuerySet returns an http.HandlerFunc that loads the set, then renders the
// props through the configured option — [Views] is the intended renderer, but
// any render option ([RenderTemplResult], [Render], …) works.
//
// The pipeline is the query pipeline minus the decode step: pre-dispatch checks
// (authorization, method guard, CSRF) run first, then the renderer's pre-load
// check (an unroutable [Views] request is refused here, before any read),
// loaders run under the handler timeout, the HTMX response options apply, and
// the renderer writes the response. Loaders that are PageOnly are skipped for
// targeted HTMX requests, per [LoadScopeFor].
//
// Two App.Query behaviours deliberately do not apply, because there is no
// request body to decode: no Accept-Query header is advertised, and
// [RequestGuard] never runs (it needs a decoded value — gate the reads
// instead; a [LoadQuery] loader keeps its `query/<Name>` posture through the
// dispatcher).
func (a *App) QuerySet[T any](set QuerySet[T], opts ...HandlerOption) http.HandlerFunc {
	set.validate()

	config := buildHandlerConfig(opts)
	if config.maxBodySize == 0 {
		config.maxBodySize = a.maxBodySize
	}

	a.collectOpenAPI("query", config)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w, r = a.applyServerTiming(w, r)
		r = a.enrichUserID(r)

		ctx, err := a.dispatchContext(w, r, config)
		if err != nil {
			return
		}

		if err := config.runBeforeLoad(r); err != nil {
			a.handleErr(w, r, ctx, config, err)

			return
		}

		a.loadAndRespond(w, r, ctx, config, set)
	})
}

// loadAndRespond runs the set's loaders under the handler timeout and writes
// the loaded props through the renderer.
func (a *App) loadAndRespond[T any](
	w http.ResponseWriter,
	r *http.Request,
	ctx context.Context,
	config *handlerConfig,
	set QuerySet[T],
) {
	//nolint:contextcheck // ctx is extracted from r inside dispatchContext
	loadCtx, cancel := a.timeoutCtx(ctx, config)
	defer cancel()

	//nolint:contextcheck // loadCtx derives from dispatchContext's ctx, which derives from r
	props, report, err := set.load(loadCtx, LoadScopeFor(r) == ScopePage)
	if err != nil {
		//nolint:contextcheck // ctx is extracted from r inside dispatchContext
		a.handleErr(w, r, ctx, config, err)

		return
	}

	reportLoad(r, config, report)

	//nolint:contextcheck // loadCtx derives from dispatchContext's ctx, which derives from r
	if renderErr := respondLoaded(w, r.WithContext(loadCtx), config, props); renderErr != nil {
		//nolint:contextcheck // ctx is extracted from r inside dispatchContext
		a.handleErr(w, r, ctx, config, renderErr)

		return
	}

	//nolint:contextcheck // loadCtx derives from dispatchContext's ctx, which derives from r
	a.afterDispatchHook(loadCtx, r, nil)
}

// QuerySetHandler is the App-less form of [App.QuerySet] for consumers that
// mount a bare mux: there is no dispatch pipeline and no authorization gate, so
// the consumer's own middleware owns both. Errors map through
// [httperr.DefaultErrorHandler] — the App-level Config.ErrorHandler and
// server-timing do not apply. The load and render rules are identical,
// including the HTMX response options.
func QuerySetHandler[T any](set QuerySet[T], opts ...HandlerOption) http.HandlerFunc {
	set.validate()

	config := buildHandlerConfig(opts)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := config.runBeforeLoad(r); err != nil {
			httperr.DefaultErrorHandler(w, r, err)

			return
		}

		props, report, err := set.load(r.Context(), LoadScopeFor(r) == ScopePage)
		if err != nil {
			httperr.DefaultErrorHandler(w, r, err)

			return
		}

		reportLoad(r, config, report)

		if renderErr := respondLoaded(w, r, config, props); renderErr != nil {
			httperr.DefaultErrorHandler(w, r, renderErr)
		}
	})
}

// respondLoaded writes a loaded props struct the way App.Query writes a query
// result: the HTMX response options first (a non-HTMX Redirect is terminal),
// then the renderer, then the default status.
func respondLoaded(w http.ResponseWriter, r *http.Request, config *handlerConfig, props any) error {
	if applyHTMXResponse(w, r, config) {
		return nil
	}

	if config.render != nil {
		if err := config.render(w, r, props); err != nil {
			return err //nolint:wrapcheck // the renderer's own error is relayed verbatim, matching App.Query
		}
	}

	writeDefaultStatus(w, config)

	return nil
}
