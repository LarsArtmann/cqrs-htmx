package cqrshtmx

import (
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// The HTMX protocol vocabulary — request accessors, swap strategies, the
// script/extension serving handlers — lives in the htmx subpackage: it is a
// self-contained concern with no dependency on the CQRS handler machinery.
//
// Every symbol in this file is a deprecated alias. Import the subpackage
// directly, where the names drop the redundant prefix:
//
//	import "github.com/larsartmann/cqrs-htmx/v4/htmx"
//
// then write htmx.Request, htmx.Target, htmx.ScriptHandler. These aliases are
// identical types and function values and are deleted in cqrs-htmx v5; see
// ADR-0062 and docs/migrations/v4-to-v5.md.

type (
	// HTMXRequest is the parsed set of HTMX request headers, stored in context
	// by HTMXMiddleware.
	//
	// Deprecated: use htmx.Request from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXRequest = htmx.Request

	// SwapStrategy defines how HTMX swaps content into the DOM.
	//
	// Deprecated: use htmx.SwapStrategy from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapStrategy = htmx.SwapStrategy
)

// HTMX swap strategy aliases.
const (
	// SwapInnerHTML replaces the target's inner HTML (the HTMX default).
	//
	// Deprecated: use htmx.SwapInnerHTML from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapInnerHTML = htmx.SwapInnerHTML
	// SwapOuterHTML replaces the target element itself.
	//
	// Deprecated: use htmx.SwapOuterHTML from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapOuterHTML = htmx.SwapOuterHTML
	// SwapBeforeBegin inserts content before the target element.
	//
	// Deprecated: use htmx.SwapBeforeBegin from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapBeforeBegin = htmx.SwapBeforeBegin
	// SwapAfterBegin inserts content at the start of the target element.
	//
	// Deprecated: use htmx.SwapAfterBegin from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapAfterBegin = htmx.SwapAfterBegin
	// SwapBeforeEnd inserts content at the end of the target element.
	//
	// Deprecated: use htmx.SwapBeforeEnd from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapBeforeEnd = htmx.SwapBeforeEnd
	// SwapAfterEnd inserts content after the target element.
	//
	// Deprecated: use htmx.SwapAfterEnd from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapAfterEnd = htmx.SwapAfterEnd
	// SwapDelete removes the target element.
	//
	// Deprecated: use htmx.SwapDelete from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapDelete = htmx.SwapDelete
	// SwapNone leaves the DOM untouched.
	//
	// Deprecated: use htmx.SwapNone from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	SwapNone = htmx.SwapNone

	// HeaderTrue is the HTMX header value for boolean true headers.
	//
	// Deprecated: use htmx.HeaderTrue from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HeaderTrue = htmx.HeaderTrue

	// HTMXCDNBaseURL is the default CDN base URL for loading HTMX.
	//
	// Deprecated: use htmx.CDNBaseURL from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXCDNBaseURL = htmx.CDNBaseURL

	// HTMXExtSSE is the htmx Server-Sent Events extension (htmx-ext-sse).
	//
	// Deprecated: use htmx.ExtSSE from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXExtSSE = htmx.ExtSSE
	// HTMXExtIdiomorph provides the morph swap strategy via idiomorph.
	//
	// Deprecated: use htmx.ExtIdiomorph from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXExtIdiomorph = htmx.ExtIdiomorph
)

// Request accessors and the script/extension handlers. Each var holds the
// identical function value declared in the htmx subpackage.
var (
	// WithHTMX stores a parsed HTMXRequest in the context.
	//
	// Deprecated: use htmx.WithRequest from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	WithHTMX = htmx.WithRequest

	// HTMXFromContext retrieves the parsed HTMXRequest from context. Returns nil
	// if HTMXMiddleware was not applied.
	//
	// Deprecated: use htmx.FromContext from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXFromContext = htmx.FromContext

	// IsHTMXRequest returns true if the request originates from HTMX. Checks the
	// HX-Request header directly — works with or without HTMXMiddleware.
	//
	// Deprecated: use htmx.IsRequest from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	IsHTMXRequest = htmx.IsRequest

	// IsBoosted returns true if the request was sent via hx-boost.
	//
	// Deprecated: use htmx.IsBoosted from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	IsBoosted = htmx.IsBoosted

	// IsHistoryRestore returns true if the request is a history restoration.
	//
	// Deprecated: use htmx.IsHistoryRestore from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	IsHistoryRestore = htmx.IsHistoryRestore

	// RenderPartial returns true when the request is from HTMX and is not a
	// history restore. Use this to decide whether to render a partial HTML
	// fragment or a full page.
	//
	// Deprecated: use htmx.RenderPartial from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	RenderPartial = htmx.RenderPartial

	// HTMXTarget returns the ID of the target element from the request.
	//
	// Deprecated: use htmx.Target from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXTarget = htmx.Target

	// HTMXTrigger returns the ID of the trigger element from the request.
	//
	// Deprecated: use htmx.Trigger from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXTrigger = htmx.Trigger

	// HTMXTriggerName returns the name of the trigger element from the request.
	//
	// Deprecated: use htmx.TriggerName from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXTriggerName = htmx.TriggerName

	// HTMXPrompt returns the user's response from hx-prompt.
	//
	// Deprecated: use htmx.Prompt from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXPrompt = htmx.Prompt

	// HTMXCurrentURL returns the current URL from the request.
	//
	// Deprecated: use htmx.CurrentURL from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXCurrentURL = htmx.CurrentURL

	// HTMXVersion returns the embedded HTMX library version string.
	//
	// Deprecated: use htmx.Version from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXVersion = htmx.Version

	// HTMXScriptHandler serves the embedded HTMX JavaScript (minified) with
	// long-lived cache headers.
	//
	// Deprecated: use htmx.ScriptHandler from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXScriptHandler = htmx.ScriptHandler

	// HTMXScriptHandlerWith serves the given HTMX JavaScript with long-lived
	// cache headers, keyed by the supplied version string for ETag revalidation.
	//
	// Deprecated: use htmx.ScriptHandlerWith from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXScriptHandlerWith = htmx.ScriptHandlerWith

	// HTMXScriptTag returns an HTML <script> tag pointing at the given path.
	//
	// Deprecated: use htmx.ScriptTag from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXScriptTag = htmx.ScriptTag

	// HTMXCDNScriptTag returns an HTML <script> tag that loads HTMX from the
	// unpkg CDN. Pass an empty version string to use the embedded library
	// version.
	//
	// Deprecated: use htmx.CDNScriptTag from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXCDNScriptTag = htmx.CDNScriptTag

	// HTMXExtensionHandler serves the embedded HTMX extension JavaScript for the
	// given extension name.
	//
	// Deprecated: use htmx.ExtensionHandler from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXExtensionHandler = htmx.ExtensionHandler

	// HTMXExtensionsHandler serves a concatenated bundle of the named embedded
	// HTMX extensions as a single asset.
	//
	// Deprecated: use htmx.ExtensionsHandler from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXExtensionsHandler = htmx.ExtensionsHandler

	// HTMXExtensionVersion returns the version string of the embedded extension.
	//
	// Deprecated: use htmx.ExtensionVersion from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXExtensionVersion = htmx.ExtensionVersion

	// HTMXExtensionNames returns the names of all available embedded HTMX
	// extensions, sorted.
	//
	// Deprecated: use htmx.ExtensionNames from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXExtensionNames = htmx.ExtensionNames

	// HTMXExtensionCDNScriptTag returns an HTML <script> tag that loads the given
	// HTMX extension from the unpkg CDN, using the embedded version.
	//
	// Deprecated: use htmx.ExtensionCDNScriptTag from github.com/larsartmann/cqrs-htmx/v4/htmx
	// directly. Removed in cqrs-htmx v5.
	HTMXExtensionCDNScriptTag = htmx.ExtensionCDNScriptTag
)
