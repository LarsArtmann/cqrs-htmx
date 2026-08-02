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
	// ElementPatchMode controls how patched elements merge into the DOM.
	ElementPatchMode = sdk.ElementPatchMode
	// TemplComponent is the interface satisfied by templ-generated components.
	TemplComponent = sdk.TemplComponent
	// ServerSentEventGenerator is the core Datastar SSE writer type.
	ServerSentEventGenerator = sdk.ServerSentEventGenerator
)

// Option function re-exports for single-import convenience.
// These mirror the Datastar SDK functional options exactly.

var (
	// --- Element patch options ---

	WithSelector      = sdk.WithSelector
	WithSelectorf     = sdk.WithSelectorf
	WithSelectorID    = sdk.WithSelectorID
	WithMode          = sdk.WithMode
	WithModeOuter     = sdk.WithModeOuter
	WithModeInner     = sdk.WithModeInner
	WithModeRemove    = sdk.WithModeRemove
	WithModePrepend   = sdk.WithModePrepend
	WithModeAppend    = sdk.WithModeAppend
	WithModeBefore    = sdk.WithModeBefore
	WithModeAfter     = sdk.WithModeAfter
	WithModeReplace   = sdk.WithModeReplace
	WithNamespace     = sdk.WithNamespace
	WithNamespaceHTML = sdk.WithNamespaceHTML
	WithNamespaceSVG  = sdk.WithNamespaceSVG

	// --- Signal patch options ---

	WithOnlyIfMissing = sdk.WithOnlyIfMissing

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
