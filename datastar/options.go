package datastar

import (
	"net/http"

	sdk "github.com/starfederation/datastar-go/datastar"
)

// Type aliases for Datastar SDK types. These allow consumers to use the adapter
// module as a single import without also importing the SDK directly.

type (
	// PatchElementOption configures element patching behavior (selector, mode, etc.).
	PatchElementOption = sdk.PatchElementOption
	// PatchSignalsOption configures signal patching behavior (only-if-missing, etc.).
	PatchSignalsOption = sdk.PatchSignalsOption
	// ExecuteScriptOption configures script execution behavior (auto-remove, attributes).
	ExecuteScriptOption = sdk.ExecuteScriptOption
	// DispatchCustomEventOption configures custom event dispatch behavior.
	DispatchCustomEventOption = sdk.DispatchCustomEventOption
	// SSEOption configures the SSE generator (context, compression, etc.).
	SSEOption = sdk.SSEOption
	// ElementPatchMode controls how patched elements merge into the DOM.
	ElementPatchMode = sdk.ElementPatchMode
	// Namespace is the XML namespace for patched elements (html, svg, mathml).
	Namespace = sdk.Namespace
	// EventType is the Datastar SSE event type string.
	//cqrs-lint:ignore(A008) re-export of Datastar SDK type, not a go-cqrs-lite duplicate
	EventType = sdk.EventType
	// TemplComponent is the interface satisfied by templ-generated components.
	TemplComponent = sdk.TemplComponent
	// ServerSentEventGenerator is the core Datastar SSE writer type.
	ServerSentEventGenerator = sdk.ServerSentEventGenerator
)

// Namespace constants (Go has no const alias, so these are vars).
var (
	NamespaceHTML   = sdk.NamespaceHTML
	NamespaceSVG    = sdk.NamespaceSVG
	NamespaceMathML = sdk.NamespaceMathML
)

// ElementPatchMode constants.
var (
	ElementPatchModeOuter   = sdk.ElementPatchModeOuter
	ElementPatchModeInner   = sdk.ElementPatchModeInner
	ElementPatchModeRemove  = sdk.ElementPatchModeRemove
	ElementPatchModeReplace = sdk.ElementPatchModeReplace
	ElementPatchModePrepend = sdk.ElementPatchModePrepend
	ElementPatchModeAppend  = sdk.ElementPatchModeAppend
	ElementPatchModeBefore  = sdk.ElementPatchModeBefore
	ElementPatchModeAfter   = sdk.ElementPatchModeAfter
)

// EventType constants.
var (
	EventTypePatchElements = sdk.EventTypePatchElements
	EventTypePatchSignals  = sdk.EventTypePatchSignals
)

// Option function re-exports for single-import convenience.
// These mirror the Datastar SDK functional options exactly.

var (
	// --- Element patch options ---

	WithSelector               = sdk.WithSelector
	WithSelectorf              = sdk.WithSelectorf
	WithSelectorID             = sdk.WithSelectorID
	WithMode                   = sdk.WithMode
	WithModeOuter              = sdk.WithModeOuter
	WithModeInner              = sdk.WithModeInner
	WithModeRemove             = sdk.WithModeRemove
	WithModePrepend            = sdk.WithModePrepend
	WithModeAppend             = sdk.WithModeAppend
	WithModeBefore             = sdk.WithModeBefore
	WithModeAfter              = sdk.WithModeAfter
	WithModeReplace            = sdk.WithModeReplace
	WithNamespace              = sdk.WithNamespace
	WithNamespaceHTML          = sdk.WithNamespaceHTML
	WithNamespaceSVG           = sdk.WithNamespaceSVG
	WithNamespaceMathML        = sdk.WithNamespaceMathML
	WithViewTransitions        = sdk.WithViewTransitions
	WithoutViewTransitions     = sdk.WithoutViewTransitions
	WithUseViewTransitions     = sdk.WithUseViewTransitions
	WithViewTransitionSelector = sdk.WithViewTransitionSelector
	WithPatchElementsEventID   = sdk.WithPatchElementsEventID
	WithRetryDuration          = sdk.WithRetryDuration

	// --- Signal patch options ---

	WithOnlyIfMissing             = sdk.WithOnlyIfMissing
	WithPatchSignalsEventID       = sdk.WithPatchSignalsEventID
	WithPatchSignalsRetryDuration = sdk.WithPatchSignalsRetryDuration

	// --- ExecuteScript options ---

	WithExecuteScriptEventID       = sdk.WithExecuteScriptEventID
	WithExecuteScriptRetryDuration = sdk.WithExecuteScriptRetryDuration
	WithExecuteScriptAutoRemove    = sdk.WithExecuteScriptAutoRemove
	WithExecuteScriptAttributes    = sdk.WithExecuteScriptAttributes
	WithExecuteScriptAttributeKVs  = sdk.WithExecuteScriptAttributeKVs

	// --- SSE options ---

	WithContext = sdk.WithContext

	// --- SSE attribute helpers (generate @get/@post/etc. strings) ---

	GetSSE    = sdk.GetSSE
	PostSSE   = sdk.PostSSE
	PutSSE    = sdk.PutSSE
	PatchSSE  = sdk.PatchSSE
	DeleteSSE = sdk.DeleteSSE
)

// NewSSE creates a Datastar Server-Sent Event generator from an HTTP response
// writer and request. This is a re-export of datastar.NewSSE for consumers who
// need direct access to the SDK's SSE type.
//
// Most consumers should use NewResponse or Broadcaster.ServeHTTP instead.
func NewSSE(w http.ResponseWriter, r *http.Request, opts ...sdk.SSEOption) *sdk.ServerSentEventGenerator {
	return sdk.NewSSE(w, r, opts...)
}
