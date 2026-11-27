package cqrshtmx

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/larsartmann/cqrs-htmx/v4/htmx"
	"github.com/larsartmann/cqrs-htmx/v4/httperr"
	errorfamily "github.com/larsartmann/go-error-family"
)

// viewKind classifies a [View]: the full page, a swappable region, or an
// out-of-band fragment.
type viewKind int

const (
	viewKindPage viewKind = iota
	viewKindRegion
	viewKindOOB
)

func (k viewKind) String() string {
	switch k {
	case viewKindPage:
		return "page"
	case viewKindRegion:
		return "region"
	case viewKindOOB:
		return "oob"
	default:
		return "unknown"
	}
}

// View is one renderable shape over a props struct T. Construct views with
// [Page], [Region] and [OOB]; the zero value is not usable.
//
// A view is pure presentation: it never dispatches a query and never loads
// data. That is the invariant the whole seam rests on — data arrives in the
// props struct, a view only decides what to draw.
type View[T any] struct {
	kind      viewKind
	selector  string
	isDefault bool
	id        string
	swap      string
	component func(T) TemplComponent
}

// Page declares the full-document view: the shape a direct browser visit, a
// boosted navigation, and a history restore all render.
func Page[T any](component func(T) TemplComponent) View[T] {
	return View[T]{kind: viewKindPage, component: component}
}

// Region declares a swappable fragment addressed by selector. The selector is
// matched against the request's HX-Target value; a leading "#" is optional on
// both sides, so Region("#rows", …) matches htmx's raw "rows" target id.
//
// A region is selected only for a targeted HTMX request. When the request is
// an HTMX partial whose target matches no region, a region marked [View.Default]
// catches it; without a default the request fails loudly (see [Views]).
func Region[T any](selector string, component func(T) TemplComponent) View[T] {
	return View[T]{kind: viewKindRegion, selector: selector, component: component}
}

// Default marks a region as the fallback for HTMX partial requests whose
// target matches no declared region. At most one region may be the default.
func (v View[T]) Default() View[T] {
	v.isDefault = true

	return v
}

// OOB declares an out-of-band swap fragment rendered alongside the primary
// view. The id is the element to update; the strategy defaults to innerHTML.
//
// Out-of-band fragments are emitted only on fragment responses (targeted HTMX
// requests); a full page never carries them, so a direct visit cannot leak a
// stray hx-swap-oob element into the document.
func OOB[T any](id string, component func(T) TemplComponent, swapStrategy ...SwapStrategy) View[T] {
	return View[T]{
		kind:      viewKindOOB,
		id:        id,
		swap:      oobSwapAttribute(swapStrategy),
		component: component,
	}
}

// Views is the render option that turns one props struct into the three view
// shapes an HTMX page actually needs. It replaces the per-handler
// `if htmx.RenderPartial(r)` branch with one declaration:
//
//	app.Query("ListTenants", decoder,
//	    cqrshtmx.Views(
//	        cqrshtmx.Region("#tenant-rows", tenantsRows),
//	        cqrshtmx.Page(tenantsPage),
//	        cqrshtmx.OOB("tenant-count", tenantCount),
//	    ))
//
// Selection is deterministic and request-derived:
//
//   - a boosted navigation or a history restore renders the Page;
//   - a targeted HTMX request renders the region whose selector matches
//     HX-Target (a leading "#" is optional on either side);
//   - a partial HTMX request that matches no region renders the region marked
//     Default, or fails with an "unroutable_view" rejection when none exists —
//     never a silent full-page fallback;
//   - every other request renders the Page.
//
// Exactly one Page is required and at most one region may be Default; both are
// validated when the option is built, so a misdeclaration panics at
// registration instead of surprising a caller per request.
func Views[T any](views ...View[T]) HandlerOption {
	set := newViewSet(views)

	return func(config *handlerConfig) {
		config.render = set.render
	}
}

// viewSet is the validated, partitioned form of a Views declaration.
type viewSet[T any] struct {
	page         *View[T]
	regions      []View[T]
	defaultIndex int
	oobs         []View[T]
}

func newViewSet[T any](views []View[T]) *viewSet[T] {
	set := &viewSet[T]{defaultIndex: -1}

	problems := declareViews(views, set)
	if len(problems) > 0 {
		//cqrs-lint:ignore(C009) programmer error: a misdeclared view set cannot be repaired per request
		panic("cqrs-htmx: Views declaration is invalid: " + strings.Join(problems, "; "))
	}

	return set
}

// declareViews partitions the views and collects EVERY declaration problem, so
// a misdeclared set reports all of them at once instead of the first.
func declareViews[T any](views []View[T], set *viewSet[T]) []string {
	var problems []string

	pages := 0

	for _, view := range views {
		switch view.kind {
		case viewKindPage:
			pages++
			if pages == 1 {
				page := view
				set.page = &page
			}
		case viewKindRegion:
			if view.selector == "" {
				problems = append(problems, "Region requires a non-empty selector")
			}

			if view.isDefault {
				if set.defaultIndex >= 0 {
					problems = append(problems, "at most one Region may be Default")
				} else {
					set.defaultIndex = len(set.regions)
				}
			}

			set.regions = append(set.regions, view)
		case viewKindOOB:
			if view.id == "" {
				problems = append(problems, "OOB requires a non-empty element id")
			}

			set.oobs = append(set.oobs, view)
		default:
			problems = append(problems, "zero-value View (use Page, Region or OOB)")
		}
	}

	switch pages {
	case 0:
		problems = append(problems, "exactly one Page is required, found none")
	case 1:
	default:
		problems = append(problems, "exactly one Page is required, found more than one")
	}

	return problems
}

func (s *viewSet[T]) render(w http.ResponseWriter, r *http.Request, result any) error {
	props, ok := result.(T)
	if !ok {
		return errorfamily.NewRejection("unexpected_result_type",
			fmt.Sprintf("unexpected result type %T", result)).WithCause(httperr.ErrDecodeFailed)
	}

	primary, err := s.selectView(r)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", ContentTypeHTML)

	if err := primary.component(props).Render(r.Context(), w); err != nil {
		return err //nolint:wrapcheck // component error passthrough, matching RenderTempl
	}

	if LoadScopeFor(r) != ScopeFragment {
		return nil
	}

	for _, view := range s.oobs {
		if err := OOBTempl(view.id, view.component(props), SwapStrategy(view.swap)).
			Render(r.Context(), w); err != nil {
			return err //nolint:wrapcheck // component error passthrough
		}
	}

	return nil
}

func (s *viewSet[T]) selectView(r *http.Request) (View[T], error) {
	if LoadScopeFor(r) == ScopePage {
		return *s.page, nil
	}

	target := htmx.Target(r)
	if region, ok := s.regionFor(target); ok {
		return region, nil
	}

	if region, ok := s.defaultRegion(); ok {
		return region, nil
	}

	return View[T]{}, errorfamily.NewRejection("unroutable_view",
		fmt.Sprintf("no view matches target %q and no Default region is declared", target)).
		WithContext("target", target)
}

func (s *viewSet[T]) regionFor(target string) (View[T], bool) {
	for _, region := range s.regions {
		if selectorsMatch(region.selector, target) {
			return region, true
		}
	}

	return View[T]{}, false
}

func (s *viewSet[T]) defaultRegion() (View[T], bool) {
	if s.defaultIndex < 0 || s.defaultIndex >= len(s.regions) {
		return View[T]{}, false
	}

	return s.regions[s.defaultIndex], true
}

// selectorsMatch compares a declared selector with the request's HX-Target,
// tolerating one optional leading "#" on either side.
func selectorsMatch(selector, target string) bool {
	return strings.TrimPrefix(selector, "#") == strings.TrimPrefix(target, "#")
}

// LoadScope tells a [QuerySet] which loaders a request actually needs.
type LoadScope int

const (
	// ScopePage is a full-page render: every loader runs, including the
	// page-only ones.
	ScopePage LoadScope = iota
	// ScopeFragment is a region or out-of-band render: page-only loaders are
	// skipped.
	ScopeFragment
)

func (s LoadScope) String() string {
	if s == ScopeFragment {
		return "fragment"
	}

	return "page"
}

// LoadScopeFor derives the load scope from the request, using the same rules
// [Views] uses to pick a shape:
//
//   - a boosted navigation or a history restore is a page;
//   - a targeted HTMX request is a fragment;
//   - everything else is a page.
//
// It is exported so a consumer can reason about (and test) the cost model of
// page-only loaders without going through the handler.
func LoadScopeFor(r *http.Request) LoadScope {
	if htmx.IsBoosted(r) || htmx.IsHistoryRestore(r) {
		return ScopePage
	}

	if htmx.RenderPartial(r) {
		return ScopeFragment
	}

	return ScopePage
}
