package dashboardui

import (
	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
)

// defItem builds a definition item with plain-text detail.
func defItem(term, detail string) display.DefinitionItem {
	return display.DefinitionItem{Term: term, Detail: detail, DetailComponent: nil}
}

// defItemRaw builds a definition item whose detail is pre-rendered HTML
// (links, code, badges). Only pass trusted, already-escaped markup.
func defItemRaw(term, html string) display.DefinitionItem {
	return display.DefinitionItem{Term: term, Detail: "", DetailComponent: rawComponent(html)}
}

// defItemCopy builds a definition item whose detail is display markup plus a
// library CopyButton carrying the raw value.
func defItemCopy(term, displayHTML, rawValue, nonce string) display.DefinitionItem {
	return display.DefinitionItem{
		Term:            term,
		Detail:          "",
		DetailComponent: templ.Join(rawComponent(displayHTML), copyButtonComponent(rawValue, nonce)),
	}
}

// rawComponent wraps pre-rendered HTML as a templ component.
func rawComponent(html string) templ.Component {
	return templ.Raw(html)
}

// copyButtonComponent builds an unrendered library CopyButton component for
// embedding inside other components (definition list details). nonce threads
// the per-request CSP nonce into the component's clipboard singleton script
// ("" renders the script without a nonce attribute, which a nonce-locked CSP
// blocks).
func copyButtonComponent(text, nonce string) templ.Component {
	return display.CopyButton(display.CopyButtonProps{
		ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: nonce,
		Text:        text,
		Label:       "",
		CopiedLabel: "",
		Icon:        true,
		Href:        "",
		LabelClass:  "",
	})
}
