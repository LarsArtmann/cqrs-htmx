package datastar

import (
	sdk "github.com/starfederation/datastar-go/datastar"
)

// Patch represents a Datastar SSE instruction that can be applied to a
// connected client. Patches are the unit of communication for real-time
// updates: they describe how the server wants to change the client's DOM
// (patch-elements), reactive state (patch-signals), or execute actions
// (remove, script, redirect).
//
// Patches are created via the constructor functions and consumed by the
// Broadcaster, EventBridge, and Response types. The apply method is
// unexported so only this package can create Patch implementations — this
// keeps the wire-format contract controlled.
type Patch interface {
	apply(sse *sdk.ServerSentEventGenerator) error
}

// --- Elements Patch ---

type elementsPatch struct {
	html string
	opts []sdk.PatchElementOption
}

func (p elementsPatch) apply(sse *sdk.ServerSentEventGenerator) error {
	return sse.PatchElements(p.html, p.opts...)
}

// ElementsPatch creates a patch that morphs HTML elements into the DOM.
// By default, the elements replace existing elements with the same ID
// (outer mode). Use options to change the merge behavior:
//
//	ds.ElementsPatch(renderTodo(todo), ds.WithSelectorID("todo-list"), ds.WithModeAppend())
func ElementsPatch(html string, opts ...PatchElementOption) Patch {
	return elementsPatch{html: html, opts: opts}
}

// --- Elements Templ Patch ---

type elementsTemplPatch struct {
	component TemplComponent
	opts      []sdk.PatchElementOption
}

func (p elementsTemplPatch) apply(sse *sdk.ServerSentEventGenerator) error {
	return sse.PatchElementTempl(p.component, p.opts...)
}

// ElementsTemplPatch creates a patch that renders a templ component into
// the DOM. This is the idiomatic way to send server-rendered HTML in
// cqrs-htmx applications that use templ.
//
//	ds.ElementsTemplPatch(todoComponent(todo), ds.WithSelectorID("todo-"+todo.ID))
func ElementsTemplPatch(component TemplComponent, opts ...PatchElementOption) Patch {
	return elementsTemplPatch{component: component, opts: opts}
}

// --- Signals Patch ---

type signalsPatch struct {
	signals any
	opts    []sdk.PatchSignalsOption
}

func (p signalsPatch) apply(sse *sdk.ServerSentEventGenerator) error {
	return sse.MarshalAndPatchSignals(p.signals, p.opts...)
}

// SignalsPatch creates a patch that updates the client's reactive signals.
// The signals argument can be any JSON-marshallable value (typically a
// map[string]any or a struct).
//
//	ds.SignalsPatch(map[string]any{
//	    "notification": map[string]string{"level": "success", "message": "Created!"},
//	    "todoCount":    42,
//	})
func SignalsPatch(signals any, opts ...PatchSignalsOption) Patch {
	return signalsPatch{signals: signals, opts: opts}
}

// --- Signals If Missing Patch ---

type signalsIfMissingPatch struct {
	signals any
	opts    []sdk.PatchSignalsOption
}

func (p signalsIfMissingPatch) apply(sse *sdk.ServerSentEventGenerator) error {
	return sse.MarshalAndPatchSignalsIfMissing(p.signals, p.opts...)
}

// SignalsIfMissingPatch creates a patch that sets signals only if they
// don't already exist on the client. Useful for initializing default state
// on connect without overwriting user changes.
func SignalsIfMissingPatch(signals any, opts ...PatchSignalsOption) Patch {
	return signalsIfMissingPatch{signals: signals, opts: opts}
}

// --- Remove Patch ---

type removePatch struct {
	selector string
}

func (p removePatch) apply(sse *sdk.ServerSentEventGenerator) error {
	return sse.RemoveElement(p.selector)
}

// RemovePatch creates a patch that removes DOM elements matching the CSS
// selector from the client.
//
//	ds.RemovePatch("#todo-" + todoID)
func RemovePatch(selector string) Patch {
	return removePatch{selector: selector}
}

// --- Script Patch ---

type scriptPatch struct {
	script string
	opts   []sdk.ExecuteScriptOption
}

func (p scriptPatch) apply(sse *sdk.ServerSentEventGenerator) error {
	return sse.ExecuteScript(p.script, p.opts...)
}

// ScriptPatch creates a patch that executes JavaScript on the client.
// Use sparingly — prefer signal-driven reactivity over direct scripting.
//
//	ds.ScriptPatch("console.log('hello')")
func ScriptPatch(script string, opts ...ExecuteScriptOption) Patch {
	return scriptPatch{script: script, opts: opts}
}

// --- Redirect Patch ---

type redirectPatch struct {
	url string
}

func (p redirectPatch) apply(sse *sdk.ServerSentEventGenerator) error {
	return sse.Redirect(p.url)
}

// RedirectPatch creates a patch that navigates the client to a new URL.
// This sends a window.location.href assignment via an executed script.
//
//	ds.RedirectPatch("/dashboard")
func RedirectPatch(url string) Patch {
	return redirectPatch{url: url}
}

// applyAll applies a slice of patches to an SSE generator in order.
func applyAll(sse *sdk.ServerSentEventGenerator, patches []Patch) error {
	for _, p := range patches {
		if err := p.apply(sse); err != nil {
			return err
		}
	}

	return nil
}
