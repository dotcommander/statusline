package main

import "strings"

// parseStyle converts a user-friendly style string like "bold italic #ff5370"
// into concatenated ANSI escape sequences.
//
// Supported keywords: bold, italic, underline, dim, strikethrough.
// Supports #RRGGBB hex colors for foreground.
func parseStyle(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, token := range strings.Fields(s) {
		switch strings.ToLower(token) {
		case "bold":
			b.WriteString(ansiBold)
		case "italic":
			b.WriteString(ansiItalic)
		case "underline":
			b.WriteString("\x1b[4m")
		case "dim":
			b.WriteString("\x1b[2m")
		case "strikethrough":
			b.WriteString("\x1b[9m")
		default:
			if strings.HasPrefix(token, "#") && len(token) == 7 {
				b.WriteString(hexToAnsi(token, false))
			}
		}
	}
	return b.String()
}

// stripAnsi removes all ANSI escape sequences from a string,
// returning only the visible characters.
func stripAnsi(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			i++
			for i < len(s) && s[i] != 'm' {
				i++
			}
			if i < len(s) {
				i++ // skip 'm'
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
