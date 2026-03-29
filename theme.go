package main

import "fmt"

// ─── Symbols ───────────────────────────────────────────────────────────────

const (
	symSeparator = " \u203a " // ` › ` breadcrumb separator
	symDot       = "\u25cf"   // ● mode dot
	symBranch    = "\ue0a0"
)

// ─── Config ────────────────────────────────────────────────────────────────

const (
	contextWarningPct  = 25
	contextCriticalPct = 10
	contextAlertPct    = 30
)

// ─── ANSI codes ────────────────────────────────────────────────────────────

var (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiItalic = "\x1b[3m"

	// Base accent colors (used by palette constructors)
	accentGreen      = hexToAnsi("#9ece6a", false)
	accentRed        = hexToAnsi("#f7768e", false)
	accentYellow     = hexToAnsi("#e0af68", false)
	accentOrange     = hexToAnsi("#ff9e64", false)
	accentPaleOrange = hexToAnsi("#d4a373", false)
	accentGray       = hexToAnsi("#565f89", false)
)

// ─── Palette ───────────────────────────────────────────────────────────────

// palette holds all foreground colors used by segment builders.
// Switching palettes changes the entire status line appearance.
type palette struct {
	muted      string
	soft       string
	active     string
	separator  string
	git        string
	prompt     string
	green      string
	red        string
	yellow     string
	orange     string
	paleOrange string
	gray       string
}

// pal is the active color palette, set in main() based on context health.
var pal *palette

// normalPalette returns the standard Tokyo Night color palette.
func normalPalette() *palette {
	return &palette{
		muted:      hexToAnsi("#565f89", false),
		soft:       hexToAnsi("#a9b1d6", false),
		active:     hexToAnsi("#c0caf5", false),
		separator:  hexToAnsi("#3b4261", false),
		git:        hexToAnsi("#7aa2f7", false),
		prompt:     hexToAnsi("#565f89", false),
		green:      accentGreen,
		red:        accentRed,
		yellow:     accentYellow,
		orange:     accentOrange,
		paleOrange: accentPaleOrange,
		gray:       accentGray,
	}
}

// alertPalette returns a palette where all colors are red,
// signaling critical context window depletion.
func alertPalette() *palette {
	return &palette{
		muted:      accentRed,
		soft:       accentRed,
		active:     accentRed,
		separator:  accentRed,
		git:        accentRed,
		prompt:     accentRed,
		green:      accentRed,
		red:        accentRed,
		yellow:     accentRed,
		orange:     accentRed,
		paleOrange: accentRed,
		gray:       accentRed,
	}
}

// hexToAnsi converts a #RRGGBB hex color to an ANSI true-color escape code.
func hexToAnsi(hex string, isBg bool) string {
	if len(hex) != 7 || hex[0] != '#' {
		return ""
	}
	r := parseHexByte(hex[1:3])
	g := parseHexByte(hex[3:5])
	b := parseHexByte(hex[5:7])
	code := 38
	if isBg {
		code = 48
	}
	return fmt.Sprintf("\x1b[%d;2;%d;%d;%dm", code, r, g, b)
}

func parseHexByte(s string) int {
	val := 0
	for _, c := range s {
		val <<= 4
		switch {
		case c >= '0' && c <= '9':
			val += int(c - '0')
		case c >= 'a' && c <= 'f':
			val += int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			val += int(c-'A') + 10
		}
	}
	return val
}
