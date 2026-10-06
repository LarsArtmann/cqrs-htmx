package dashboardui

import (
	"strings"
	"testing"

	memorystorage "github.com/larsartmann/go-cqrs-lite/storage/memory/v4"
)

// TestIsValidAccentColor_Valid pins the accepted grammar for M02: #hex
// (3/4/6/8 digits, any case), rgb()/rgba()/hsl()/hsla() with numeric
// components in comma or modern space/slash syntax, and named CSS colors
// (case-insensitive, including "transparent").
func TestIsValidAccentColor_Valid(t *testing.T) {
	valid := []string{
		defaultAccentColor,
		"#abc", "#abcd", "#a1b2c3", "#a1b2c3d4", "#ABCDEF", "#AbCdEf",
		"rgb(1, 2, 3)", "rgb(255 0 0)", "rgb(255 0 0 / 0.5)",
		"rgba(1,2,3,0.5)", "rgb(50%, 50%, 50%)",
		"hsl(120, 100%, 25%)", "hsl(120 100% 25% / 50%)", "hsla(300, 100%, 50%, 0.75)",
		"red", "RebeccaPurple", "LIGHTSKYBLUE", "lightgoldenrodyellow", "transparent",
	}
	for _, color := range valid {
		if !isValidAccentColor(color) {
			t.Errorf("isValidAccentColor(%q) = false, want true", color)
		}
	}
}

// TestIsValidAccentColor_RejectsBreakout pins M02/F07: values that would
// break out of the inline --accent declaration (or smuggle CSS constructs)
// must be rejected. The sink is a raw <style> block that HTML escaping cannot
// sanitize, so braces, semicolons, quotes, comments, url()/expression(),
// var() references, unit suffixes, and whitespace-padded junk are all
// invalid.
func TestIsValidAccentColor_RejectsBreakout(t *testing.T) {
	invalid := []string{
		"",
		"red;} body{display:none}",
		"#4f46e5;} *{display:none}",
		"#4f46e5; background:url(javascript:alert(1))",
		"url(javascript:alert(1))",
		"expression(alert(1))",
		"var(--brand)",
		"red /* comment */",
		"currentcolor",
		"oklch(0.5 0.1 20)",
		"color-mix(in srgb, red, blue)",
		"hsl(120deg, 100%, 25%)",
		"#12", "#12345", "#1234567", "#123456789", "#gggggg",
		"rgb(1,2,3", "rgb(1;2;3)", "rgb(255,0,0) extra",
		"\nred", "red\t", " red", "red blue",
		strings.Repeat("a", 33),
	}
	for _, color := range invalid {
		if isValidAccentColor(color) {
			t.Errorf("isValidAccentColor(%q) = true, want false", color)
		}
	}
}

// TestNewRejectsHostileAccentColor pins the M02 integration: New fails fast
// with a Rejection-family config error when AccentColor carries a CSS
// breakout, instead of splicing it into the rendered page.
func TestNewRejectsHostileAccentColor(t *testing.T) {
	store := memorystorage.NewMemoryStore()

	d, err := New(Config{
		EventSource: store,
		Journal:     store,
		AccentColor: "red;} body{display:none}",
	})
	if err == nil {
		t.Fatalf("New accepted a CSS-breakout AccentColor; dashboard = %+v", d)
	}

	if !strings.Contains(err.Error(), "AccentColor") {
		t.Errorf("error should name AccentColor, got: %v", err)
	}
}

// TestNewAcceptsLegitAccentColors pins M02/F08: representative valid accents
// (the default, hex8 with alpha, a named color) flow through New unchanged —
// strict validation must not narrow legitimate configuration.
func TestNewAcceptsLegitAccentColors(t *testing.T) {
	for _, color := range []string{defaultAccentColor, "#a1b2c3d4", "teal"} {
		store := memorystorage.NewMemoryStore()

		d, err := New(Config{
			EventSource: store,
			Journal:     store,
			AccentColor: color,
		})
		if err != nil {
			t.Errorf("New(AccentColor=%q) failed: %v", color, err)
		}

		if d.config.AccentColor != color {
			t.Errorf("AccentColor = %q, want the configured value %q", d.config.AccentColor, color)
		}
	}
}
