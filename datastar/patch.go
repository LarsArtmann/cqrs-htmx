// Package datastar provides a DataStar adapter for cqrs-htmx applications.
//
// This module wraps [go-datastar] and [go-sse] to provide:
//   - Broadcaster: fan-out SSE patches using sse.Broadcaster[sse.Event]
//   - EventBridge: declarative CQRS event → Patch mapping
//   - Re-exports of common go-datastar types for single-import convenience
//
// Patches are first-class values implementing [godatastar.Patch] (Event() sse.Event).
// This means they can be stored, filtered, replayed, and broadcast using go-sse's
// Broadcaster, EventStore, SubscribeFilter, and Shutdown infrastructure.
//
// [go-datastar]: https://github.com/LarsArtmann/go-datastar
// [go-sse]: https://github.com/LarsArtmann/go-sse
package datastar

import (
	"net/http"

	godatastar "github.com/larsartmann/go-datastar"
	"github.com/larsartmann/go-sse"
)

// Patch is a DataStar SSE instruction that produces an [sse.Event].
// It is an alias for [godatastar.Patch].
type Patch = godatastar.Patch

// --- Type re-exports for single-import convenience ---

type (
	// ElementPatchOption configures element patching behavior.
	ElementPatchOption = godatastar.ElementPatchOption
	// SignalsPatchOption configures signal patching behavior.
	SignalsPatchOption = godatastar.SignalsPatchOption
	// ScriptPatchOption configures script execution behavior.
	ScriptPatchOption = godatastar.ScriptPatchOption
	// DispatchCustomEventOption configures custom event dispatch.
	DispatchCustomEventOption = godatastar.DispatchCustomEventOption
	// ElementPatchMode controls how patched elements merge into the DOM.
	ElementPatchMode = godatastar.ElementPatchMode
	// Namespace is the XML namespace for patched elements.
	Namespace = godatastar.Namespace
	// EventType is the DataStar SSE event type string.
//cqrs-lint:ignore(A008) re-export of go-datastar SDK type, not a go-cqrs-lite duplicate
	EventType = godatastar.EventType
	// TemplComponent is the interface satisfied by templ-generated components.
	TemplComponent = godatastar.TemplComponent
	// Response is the fluent SSE response builder from go-datastar.
	Response = godatastar.Response
)

// --- Option re-exports ---

var (
	// Element patch options
	WithSelector               = godatastar.WithSelector
	WithSelectorf              = godatastar.WithSelectorf
	WithSelectorID             = godatastar.WithSelectorID
	WithMode                   = godatastar.WithMode
	WithModeOuter              = godatastar.WithModeOuter
	WithModeInner              = godatastar.WithModeInner
	WithModeRemove             = godatastar.WithModeRemove
	WithModePrepend            = godatastar.WithModePrepend
	WithModeAppend             = godatastar.WithModeAppend
	WithModeBefore             = godatastar.WithModeBefore
	WithModeAfter              = godatastar.WithModeAfter
	WithModeReplace            = godatastar.WithModeReplace
	WithNamespace              = godatastar.WithNamespace
	WithNamespaceHTML          = godatastar.WithNamespaceHTML
	WithNamespaceSVG           = godatastar.WithNamespaceSVG
	WithNamespaceMathML        = godatastar.WithNamespaceMathML
	WithViewTransitions        = godatastar.WithViewTransitions
	WithViewTransitionsEnabled = godatastar.WithViewTransitionsEnabled
	WithoutViewTransitions     = godatastar.WithoutViewTransitions
	WithViewTransitionSelector = godatastar.WithViewTransitionSelector
	WithElementsEventID        = godatastar.WithElementsEventID
	WithElementsRetryDuration  = godatastar.WithElementsRetryDuration

	// Signal patch options
	WithOnlyIfMissing        = godatastar.WithOnlyIfMissing
	WithSignalsEventID       = godatastar.WithSignalsEventID
	WithSignalsRetryDuration = godatastar.WithSignalsRetryDuration

	// Script patch options
	WithScriptAutoRemove    = godatastar.WithScriptAutoRemove
	WithScriptAttributes    = godatastar.WithScriptAttributes
	WithScriptAttributeKVs  = godatastar.WithScriptAttributeKVs
	WithScriptEventID       = godatastar.WithScriptEventID
	WithScriptRetryDuration = godatastar.WithScriptRetryDuration

	// Custom event options
	WithCustomEventSelector   = godatastar.WithCustomEventSelector
	WithCustomEventBubbles    = godatastar.WithCustomEventBubbles
	WithCustomEventCancelable = godatastar.WithCustomEventCancelable
	WithCustomEventComposed   = godatastar.WithCustomEventComposed
	WithCustomEventEventID    = godatastar.WithCustomEventEventID

	// HTTP verb helpers
	GetSSE    = godatastar.GetSSE
	PostSSE   = godatastar.PostSSE
	PutSSE    = godatastar.PutSSE
	PatchSSE  = godatastar.PatchSSE
	DeleteSSE = godatastar.DeleteSSE
)

// --- Mode and namespace constants ---

var (
	ElementPatchModeOuter   = godatastar.ElementPatchModeOuter
	ElementPatchModeInner   = godatastar.ElementPatchModeInner
	ElementPatchModeRemove  = godatastar.ElementPatchModeRemove
	ElementPatchModeReplace = godatastar.ElementPatchModeReplace
	ElementPatchModePrepend = godatastar.ElementPatchModePrepend
	ElementPatchModeAppend  = godatastar.ElementPatchModeAppend
	ElementPatchModeBefore  = godatastar.ElementPatchModeBefore
	ElementPatchModeAfter   = godatastar.ElementPatchModeAfter

	NamespaceHTML   = godatastar.NamespaceHTML
	NamespaceSVG    = godatastar.NamespaceSVG
	NamespaceMathML = godatastar.NamespaceMathML

	EventTypePatchElements = godatastar.EventTypePatchElements
	EventTypePatchSignals  = godatastar.EventTypePatchSignals
)

// --- Patch constructors (thin wrappers around go-datastar) ---

// ElementsPatch creates a patch that morphs HTML elements into the DOM.
func ElementsPatch(html string, opts ...ElementPatchOption) Patch {
	return godatastar.NewElementsPatch(html, opts...)
}

// ElementsTemplPatch renders a templ component and patches it into the DOM.
// Returns an error if rendering fails.
func ElementsTemplPatch(component TemplComponent, opts ...ElementPatchOption) (Patch, error) {
	return godatastar.ElementsFromTempl(component, opts...)
}

// SignalsPatch creates a patch that updates the client's reactive signals.
// Returns an error if marshaling the signals value to JSON fails.
func SignalsPatch(signals any, opts ...SignalsPatchOption) (Patch, error) {
	return godatastar.NewSignalsPatch(signals, opts...)
}

// SignalsIfMissingPatch creates a patch that sets signals only if they don't exist.
// Returns an error if marshaling the signals value to JSON fails.
func SignalsIfMissingPatch(signals any, opts ...SignalsPatchOption) (Patch, error) {
	return godatastar.NewSignalsIfMissingPatch(signals, opts...)
}

// RemovePatch creates a patch that removes DOM elements matching the selector.
func RemovePatch(selector string) Patch {
	return godatastar.NewRemovePatch(selector)
}

// RemoveByIDPatch creates a patch that removes an element by its ID.
func RemoveByIDPatch(id string) Patch {
	return godatastar.NewRemoveByIDPatch(id)
}

// ScriptPatch creates a patch that executes JavaScript on the client.
func ScriptPatch(script string, opts ...ScriptPatchOption) Patch {
	return godatastar.NewScriptPatch(script, opts...)
}

// RedirectPatch creates a patch that navigates the client to a new URL.
func RedirectPatch(url string) Patch {
	return godatastar.NewRedirectPatch(url)
}

// --- Inbound helpers ---

// ReadSignals extracts DataStar signals from an HTTP request.
func ReadSignals(r *http.Request, signals any) error {
	return godatastar.ReadSignals(r, signals)
}

// --- HTTP helpers ---

// NewResponse creates a fluent SSE response builder from an HTTP handler.
func NewResponse(w http.ResponseWriter, r *http.Request) *Response {
	return godatastar.NewResponse(sse.NewStream(w, r))
}

// ScriptHandler serves the embedded DataStar JavaScript client.
func ScriptHandler() http.Handler { return godatastar.ScriptHandler() }

// ScriptTag returns an HTML script tag for loading the DataStar client.
func ScriptTag(path string) string { return godatastar.ScriptTag(path) }

// Version returns the version of the embedded DataStar JavaScript client.
func Version() string { return godatastar.Version() }

// ErrorResponse sends a signals patch with error information.
func ErrorResponse(stream *sse.Stream, message string, code string) error {
	return godatastar.ErrorResponse(stream, message, code)
}
