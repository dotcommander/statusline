package main

import (
	"strings"
	"testing"
)

func init() {
	pal = normalPalette()
}

func TestProgressivePromptCollapse(t *testing.T) {
	t.Parallel()
	dot := pal.green + defaultConfig.Dot + ansiReset
	sep := defaultConfig.Separator

	makePromptSlots := func(prompts []string) []slot {
		var slots []slot
		shown := 0
		for i := len(prompts) - 1; i >= 0 && shown < 3; i-- {
			prompt := prompts[i]
			icon := getPromptIcon(prompt)
			wordCount := 3
			color := pal.gray
			style := ansiItalic
			prefix := " "
			if shown == 0 {
				wordCount = 4
				color = pal.prompt
				style = ""
				prefix = "  "
			}
			content := prefix + color + style + icon + " " + strings.ToLower(truncateWords(prompt, wordCount))
			w := plainLen(content)
			seg := Segment{Content: content, FgAnsi: pal.prompt}
			captured := seg
			slots = append(slots, slot{
				line: 1, width: w, noSep: true,
				build: func(_ bool) Segment { return captured },
			})
			shown++
		}
		return slots
	}

	dirSlot := slot{
		line: 1, width: 10,
		build: func(active bool) Segment {
			return dirSegment("statusline", active)
		},
	}

	prompts := []string{
		"add authentication to the api endpoint",
		"fix the broken tests in auth module",
		"review work done today on the project",
	}

	t.Run("all prompts fit at wide terminal", func(t *testing.T) {
		pSlots := makePromptSlots(prompts)
		allSlots := append([]slot{dirSlot}, pSlots...)
		allSlots = append(allSlots, slot{
			line: 2, width: 5,
			build: func(_ bool) Segment { return Segment{Content: "line2", FgAnsi: ""} },
		})

		out := renderLines(dot, allSlots, 200, sep)
		lines := strings.Split(out, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %q", len(lines), out)
		}

		line1Width := plainLen(lines[0])
		if line1Width > 200 {
			t.Errorf("line 1 width %d exceeds termWidth 200", line1Width)
		}
	})

	t.Run("oldest prompt dropped at medium width", func(t *testing.T) {
		pSlots := makePromptSlots(prompts)
		allSlots := append([]slot{dirSlot}, pSlots...)
		allSlots = append(allSlots, slot{
			line: 2, width: 5,
			build: func(_ bool) Segment { return Segment{Content: "line2", FgAnsi: ""} },
		})

		// Compute the full width of line 1 with all prompts
		fullWidth := 2 // dotWidth
		for _, s := range allSlots {
			if s.line == 1 {
				fullWidth += s.width
			}
		}

		// Set termWidth so the oldest prompt must be dropped
		termWidth := fullWidth - 5
		out := renderLines(dot, allSlots, termWidth, sep)
		lines := strings.Split(out, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		line1Width := plainLen(lines[0])
		if line1Width > termWidth {
			t.Errorf("line 1 width %d exceeds termWidth %d", line1Width, termWidth)
		}

		// Should still contain the newest prompt text
		if !strings.Contains(lines[0], "review work done") {
			t.Error("newest prompt should be preserved")
		}
	})

	t.Run("all prompts dropped at narrow width", func(t *testing.T) {
		pSlots := makePromptSlots(prompts)
		allSlots := append([]slot{dirSlot}, pSlots...)
		allSlots = append(allSlots, slot{
			line: 2, width: 5,
			build: func(_ bool) Segment { return Segment{Content: "line2", FgAnsi: ""} },
		})

		out := renderLines(dot, allSlots, 20, sep)
		lines := strings.Split(out, "\n")

		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}

		line1Width := plainLen(lines[0])
		if line1Width > 20 {
			t.Errorf("line 1 width %d exceeds termWidth 20", line1Width)
		}
	})

	t.Run("line 2 always present", func(t *testing.T) {
		for _, tw := range []int{20, 40, 60, 80, 100, 120, 200} {
			pSlots := makePromptSlots(prompts)
			allSlots := append([]slot{dirSlot}, pSlots...)
			allSlots = append(allSlots, slot{
				line: 2, width: 5,
				build: func(_ bool) Segment { return Segment{Content: "line2", FgAnsi: ""} },
			})

			out := renderLines(dot, allSlots, tw, sep)
			lines := strings.Split(out, "\n")
			if len(lines) != 2 {
				t.Errorf("termWidth=%d: expected 2 lines, got %d", tw, len(lines))
			}
			if plainLen(lines[0]) > tw {
				t.Errorf("termWidth=%d: line 1 width %d exceeds termWidth", tw, plainLen(lines[0]))
			}
		}
	})
}

func TestPlainLen(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"\x1b[38;2;100;200;50mhello\x1b[0m", 5},
		{"\x1b[3m\x1b[38;2;86;95;137m› test\x1b[0m", 6},
		{"  " + "\x1b[38;2;86;95;137m" + "› hello", 9}, // "  › hello"
		{"●", 1},
	}
	for _, tt := range tests {
		got := plainLen(tt.input)
		if got != tt.want {
			t.Errorf("plainLen(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestPlainLenCellWidth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"ascii", "abc", 3},
		{"ansi-wrapped ascii", "\x1b[31mred\x1b[0m", 3},
		{"cjk wide chars", "日本", 4},
		{"cjk plus ascii", "abc日", 5},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := plainLen(tt.input)
			if got != tt.want {
				t.Errorf("plainLen(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncateVisibleCellWidth(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{"cjk truncated to width 4", "日本語", 4, "日本"},
		{"cjk truncated to width 3 drops wide rune", "日本語", 3, "日"},
		{"ansi preserved with truncation", "\x1b[31m日本語\x1b[0m", 2, "\x1b[31m日"},
		{"ascii unchanged when within width", "hello", 10, "hello"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := truncateVisible(tt.input, tt.maxWidth)
			if got != tt.want {
				t.Errorf("truncateVisible(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}
