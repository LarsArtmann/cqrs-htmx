package dashboardui

import (
	"context"
	"strings"

	"github.com/a-h/templ"
	"github.com/larsartmann/templ-components/display"
	"github.com/larsartmann/templ-components/utils"
)

// buttonLink renders a library Button as an anchor (navigation actions
// styled as buttons: View, Previous, Next, Back). A non-empty ariaLabel is
// attached; disabled renders the library's aria-disabled + tabindex=-1 +
// pointer-events-none treatment (replacing the old aria-disabled <span>).
func buttonLink(
	ctx context.Context,
	text, href, ariaLabel string,
	variant display.ButtonType,
	disabled bool,
) string {
	var b strings.Builder

	props := display.ButtonProps{
		BaseProps: utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: ariaLabel, Nonce: ""},
		Text:      text,
		Type:      display.ButtonHTMLButton,
		Href:      href,
		Variant:   variant,
		Size:      display.ButtonSizeMD,
		Disabled:  disabled,
		Icon:      nil,
		External:  false,
		Wire:      nil,
	}
	_ = display.Button(props).Render(ctx, &b)

	return b.String()
}

// buttonSubmit renders a library submit button for the data-confirm forms
// (Delete, Reset, Purge, Replay). The attrs carry the request-scoped extras
// (the payload-copy onclick handlers until the M13 CopyButton adoption).
func buttonSubmit(
	ctx context.Context,
	text, ariaLabel string,
	variant display.ButtonType,
	attrs templ.Attributes,
) string {
	var b strings.Builder

	props := display.ButtonProps{
		BaseProps: utils.BaseProps{
			ID:        "",
			Class:     "",
			Attrs:     attrs,
			AriaLabel: ariaLabel,
			Nonce:     "",
		},
		Text:     text,
		Type:     display.ButtonHTMLSubmit,
		Href:     "",
		Variant:  variant,
		Size:     display.ButtonSizeMD,
		Disabled: false,
		Icon:     nil,
		External: false,
		Wire:     nil,
	}
	_ = display.Button(props).Render(ctx, &b)

	return b.String()
}

// copyButtonHTML renders a library CopyButton (data-tc-copy + the library's
// singleton delegated clipboard script). The empty label falls back to the
// library's "Copy"/"Copied!" pair.
func copyButtonHTML(ctx context.Context, text, label string) string {
	var b strings.Builder

	props := display.CopyButtonProps{
		BaseProps:   utils.BaseProps{ID: "", Class: "", Attrs: nil, AriaLabel: "", Nonce: ""},
		Text:        text,
		Label:       label,
		CopiedLabel: "",
		Icon:        true,
		Href:        "",
	}
	_ = display.CopyButton(props).Render(ctx, &b)

	return b.String()
}
