package cqrshtmx

import (
	"github.com/larsartmann/cqrs-htmx/v4/htmx"
)

// The HTMX protocol vocabulary — request accessors, swap strategies, the
// script/extension serving handlers — lives in the htmx subpackage: it is a
// self-contained concern with no dependency on the CQRS handler machinery.
//
// Everything below is a re-export, so existing code keeps compiling unchanged:
// cqrshtmx.IsHTMXRequest and htmx.IsRequest are the same function value, and
// cqrshtmx.HTMXRequest is the identical type. New code may import the
// subpackage directly, where the names drop the redundant prefix
// (htmx.Request, htmx.Target, htmx.ScriptHandler).
//
// The root package keeps these aliases as its supported surface; they are not
// deprecated.

type (
	// HTMXRequest is the parsed set of HTMX request headers, stored in context
	// by HTMXMiddleware.
	HTMXRequest = htmx.Request

	// SwapStrategy defines how HTMX swaps content into the DOM.
	SwapStrategy = htmx.SwapStrategy
)

// HTMX swap strategy aliases.
const (
	// SwapInnerHTML replaces the target's inner HTML (the HTMX default).
	SwapInnerHTML = htmx.SwapInnerHTML
	// SwapOuterHTML replaces the target element itself.
	SwapOuterHTML = htmx.SwapOuterHTML
	// SwapBeforeBegin inserts content before the target element.
	SwapBeforeBegin = htmx.SwapBeforeBegin
	// SwapAfterBegin inserts content at the start of the target element.
	SwapAfterBegin = htmx.SwapAfterBegin
	// SwapBeforeEnd inserts content at the end of the target element.
	SwapBeforeEnd = htmx.SwapBeforeEnd
	// SwapAfterEnd inserts content after the target element.
	SwapAfterEnd = htmx.SwapAfterEnd
	// SwapDelete removes the target element.
	SwapDelete = htmx.SwapDelete
	// SwapNone leaves the DOM untouched.
	SwapNone = htmx.SwapNone

	// HeaderTrue is the HTMX header value for boolean true headers.
	HeaderTrue = htmx.HeaderTrue

	// HTMXCDNBaseURL is the default CDN base URL for loading HTMX.
	HTMXCDNBaseURL = htmx.CDNBaseURL

	// HTMXExtSSE is the htmx Server-Sent Events extension (htmx-ext-sse).
	HTMXExtSSE = htmx.ExtSSE
	// HTMXExtIdiomorph provides the morph swap strategy via idiomorph.
	HTMXExtIdiomorph = htmx.ExtIdiomorph
)

// Request accessors and the script/extension handlers. Each var holds the
// identical function value declared in the htmx subpackage.
var (
	// WithHTMX stores a parsed HTMXRequest in the context.
	WithHTMX = htmx.WithRequest

	// HTMXFromContext retrieves the parsed HTMXRequest from context. Returns nil
	// if HTMXMiddleware was not applied.
	HTMXFromContext = htmx.FromContext

	// IsHTMXRequest returns true if the request originates from HTMX. Checks the
	// HX-Request header directly — works with or without HTMXMiddleware.
	IsHTMXRequest = htmx.IsRequest

	// IsBoosted returns true if the request was sent via hx-boost.
	IsBoosted = htmx.IsBoosted

	// IsHistoryRestore returns true if the request is a history restoration.
	IsHistoryRestore = htmx.IsHistoryRestore

	// RenderPartial returns true when the request is from HTMX and is not a
	// history restore. Use this to decide whether to render a partial HTML
	// fragment or a full page.
	RenderPartial = htmx.RenderPartial

	// HTMXTarget returns the ID of the target element from the request.
	HTMXTarget = htmx.Target

	// HTMXTrigger returns the ID of the trigger element from the request.
	HTMXTrigger = htmx.Trigger

	// HTMXTriggerName returns the name of the trigger element from the request.
	HTMXTriggerName = htmx.TriggerName

	// HTMXPrompt returns the user's response from hx-prompt.
	HTMXPrompt = htmx.Prompt

	// HTMXCurrentURL returns the current URL from the request.
	HTMXCurrentURL = htmx.CurrentURL

	// HTMXVersion returns the embedded HTMX library version string.
	HTMXVersion = htmx.Version

	// HTMXScriptHandler serves the embedded HTMX JavaScript (minified) with
	// long-lived cache headers.
	HTMXScriptHandler = htmx.ScriptHandler

	// HTMXScriptHandlerWith serves the given HTMX JavaScript with long-lived
	// cache headers, keyed by the supplied version string for ETag revalidation.
	HTMXScriptHandlerWith = htmx.ScriptHandlerWith

	// HTMXScriptTag returns an HTML <script> tag pointing at the given path.
	HTMXScriptTag = htmx.ScriptTag

	// HTMXCDNScriptTag returns an HTML <script> tag that loads HTMX from the
	// unpkg CDN. Pass an empty version string to use the embedded library
	// version.
	HTMXCDNScriptTag = htmx.CDNScriptTag

	// HTMXExtensionHandler serves the embedded HTMX extension JavaScript for the
	// given extension name.
	HTMXExtensionHandler = htmx.ExtensionHandler

	// HTMXExtensionsHandler serves a concatenated bundle of the named embedded
	// HTMX extensions as a single asset.
	HTMXExtensionsHandler = htmx.ExtensionsHandler

	// HTMXExtensionVersion returns the version string of the embedded extension.
	HTMXExtensionVersion = htmx.ExtensionVersion

	// HTMXExtensionNames returns the names of all available embedded HTMX
	// extensions, sorted.
	HTMXExtensionNames = htmx.ExtensionNames

	// HTMXExtensionCDNScriptTag returns an HTML <script> tag that loads the given
	// HTMX extension from the unpkg CDN, using the embedded version.
	HTMXExtensionCDNScriptTag = htmx.ExtensionCDNScriptTag
)
