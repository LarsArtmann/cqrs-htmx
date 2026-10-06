package dashboardui

import "strings"

// maxAccentColorLen bounds any accepted accent color literal. The longest
// valid form is a modern rgb()/hsla() component list; named colors cap at 20
// characters ("lightgoldenrodyellow").
const maxAccentColorLen = 32

// cssNamedColors is the closed set of named CSS color keywords accepted for
// Config.AccentColor, plus "transparent". "currentcolor" is deliberately
// absent: it is a computed reference, not a literal color.
var cssNamedColors = map[string]struct{}{
	"aliceblue": {}, "antiquewhite": {}, "aqua": {}, "aquamarine": {},
	"azure": {}, "beige": {}, "bisque": {}, "black": {}, "blanchedalmond": {},
	"blue": {}, "blueviolet": {}, "brown": {}, "burlywood": {}, "cadetblue": {},
	"chartreuse": {}, "chocolate": {}, "coral": {}, "cornflowerblue": {},
	"cornsilk": {}, "crimson": {}, "cyan": {}, "darkblue": {}, "darkcyan": {},
	"darkgoldenrod": {}, "darkgray": {}, "darkgreen": {}, "darkgrey": {},
	"darkkhaki": {}, "darkmagenta": {}, "darkolivegreen": {}, "darkorange": {},
	"darkorchid": {}, "darkred": {}, "darksalmon": {}, "darkseagreen": {},
	"darkslateblue": {}, "darkslategray": {}, "darkslategrey": {},
	"darkturquoise": {}, "darkviolet": {}, "deeppink": {}, "deepskyblue": {},
	"dimgray": {}, "dimgrey": {}, "dodgerblue": {}, "firebrick": {},
	"floralwhite": {}, "forestgreen": {}, "fuchsia": {}, "gainsboro": {},
	"ghostwhite": {}, "gold": {}, "goldenrod": {}, "gray": {}, "green": {},
	"greenyellow": {}, "grey": {}, "honeydew": {}, "hotpink": {},
	"indianred": {}, "indigo": {}, "ivory": {}, "khaki": {}, "lavender": {},
	"lavenderblush": {}, "lawngreen": {}, "lemonchiffon": {}, "lightblue": {},
	"lightcoral": {}, "lightcyan": {}, "lightgoldenrodyellow": {},
	"lightgray": {}, "lightgreen": {}, "lightgrey": {}, "lightpink": {},
	"lightsalmon": {}, "lightseagreen": {}, "lightskyblue": {},
	"lightslategray": {}, "lightslategrey": {}, "lightsteelblue": {},
	"lightyellow": {}, "lime": {}, "limegreen": {}, "linen": {}, "magenta": {},
	"maroon": {}, "mediumaquamarine": {}, "mediumblue": {},
	"mediumorchid": {}, "mediumpurple": {}, "mediumseagreen": {},
	"mediumslateblue": {}, "mediumspringgreen": {}, "mediumturquoise": {},
	"mediumvioletred": {}, "midnightblue": {}, "mintcream": {}, "mistyrose": {},
	"moccasin": {}, "navajowhite": {}, "navy": {}, "oldlace": {}, "olive": {},
	"olivedrab": {}, "orange": {}, "orangered": {}, "orchid": {},
	"palegoldenrod": {}, "palegreen": {}, "paleturquoise": {},
	"palevioletred": {}, "papayawhip": {}, "peachpuff": {}, "peru": {},
	"pink": {}, "plum": {}, "powderblue": {}, "purple": {},
	"rebeccapurple": {}, "red": {}, "rosybrown": {}, "royalblue": {},
	"saddlebrown": {}, "salmon": {}, "sandybrown": {}, "seagreen": {},
	"seashell": {}, "sienna": {}, "silver": {}, "skyblue": {}, "slateblue": {},
	"slategray": {}, "slategrey": {}, "snow": {}, "springgreen": {},
	"steelblue": {}, "tan": {}, "teal": {}, "thistle": {}, "tomato": {},
	"transparent": {}, "turquoise": {}, "violet": {}, "wheat": {}, "white": {},
	"whitesmoke": {}, "yellow": {}, "yellowgreen": {},
}

// isValidAccentColor reports whether color is a CSS color literal the
// dashboard is willing to interpolate into its inline :root style block:
// #hex (3/4/6/8 digits), rgb()/rgba()/hsl()/hsla() with numeric components,
// or a named CSS color (plus "transparent").
//
// The accent value is spliced into a raw <style> context, which the HTML
// escaper cannot sanitize: a value like `red;} body{...}` would break out of
// the --accent declaration and inject page rules. Rejecting everything
// outside this grammar makes that class unrepresentable in configuration.
// Function arguments accept digits, percentages, and separators only — no
// letters — so unit suffixes (deg, turn) are not accepted; use the
// comma/number forms or hex instead.
func isValidAccentColor(color string) bool {
	if color == "" || len(color) > maxAccentColorLen {
		return false
	}

	if strings.HasPrefix(color, "#") {
		digits := color[1:]

		switch len(digits) {
		case 3, 4, 6, 8:
			return isHexDigits(digits)
		default:
			return false
		}
	}

	lower := strings.ToLower(color)
	for _, fn := range []string{"rgb(", "rgba(", "hsl(", "hsla("} {
		if strings.HasPrefix(lower, fn) {
			if !strings.HasSuffix(lower, ")") {
				return false
			}

			return onlyNumericColorComponents(lower[len(fn) : len(lower)-1])
		}
	}

	_, ok := cssNamedColors[lower]

	return ok
}

// isHexDigits reports whether s is non-empty and entirely ASCII hex digits.
func isHexDigits(s string) bool {
	if s == "" {
		return false
	}

	for _, r := range s {
		if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
			return false
		}
	}

	return true
}

// onlyNumericColorComponents reports whether s contains only the characters
// that can appear between the parentheses of a numeric rgb()/hsl() color:
// digits, dot, percent, sign, comma, space, and the modern-syntax slash.
// Letters, quotes, braces, and every other byte are rejected.
func onlyNumericColorComponents(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}

	for _, r := range s {
		if !strings.ContainsRune("0123456789.,%+-/ \t", r) {
			return false
		}
	}

	return true
}
