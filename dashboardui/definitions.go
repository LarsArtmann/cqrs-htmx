package dashboardui

import (
	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/utils"
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
func defItemCopy(term, displayHTML, rawValue string) display.DefinitionItem {
	return display.DefinitionItem{
		Term:            term,
		Detail:          "",
		DetailComponent: templ.Join(rawComponent(displayHTML), copyButtonComponent(rawValue)),
	}
}

// rawComponent wraps pre-rendered HTML as a templ component.
func rawComponent(html string) templ.Component {
	return templ.Raw(html)
}

// copyButtonComponent builds an unrendered library CopyButton component for
// embedding inside other components (definition list details).
func copyButtonComponent(text string) templ.Component {
	return display.CopyButton(display.CopyButtonProps{
		BaseProps:   utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Text:        text,
		Label:       "",
		CopiedLabel: "",
		Icon:        true,
		Href:        "",
	})
}
