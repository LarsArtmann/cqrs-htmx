package dashboardui

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/utils"
)

// definitionListHTML renders the library DefinitionList into a string (hybrid
// adoption path). Terms/details are escaped by the library; use defItemRaw
// for pre-rendered detail markup and defItemCopy for copyable values.
func definitionListHTML(ctx context.Context, items []display.DefinitionItem) string {
	var b strings.Builder

	props := display.DefinitionListProps{
		BaseProps: utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Items:     items,
	}
	_ = display.DefinitionList(props).Render(ctx, &b)

	return b.String()
}

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
