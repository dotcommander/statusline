package main

import (
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/dotcommander/statusline/internal/config"
	"github.com/dotcommander/statusline/internal/gitutil"
	"github.com/mattn/go-runewidth"
)

// ─── Segment ───────────────────────────────────────────────────────────────

// Segment holds a single breadcrumb segment's content and foreground color.
type Segment struct {
	Content string
	FgAnsi  string
}

// slot holds a deferred segment builder with its target line for declarative layout.
type slot struct {
	line          int                       // which output line (1, 2, ...)
	width         int                       // plain-text width for collapse calc
	build         func(active bool) Segment // lazy builder
	noSep         bool                      // skip separator before this slot
	styleOverride string                    // pre-parsed ANSI from per-token style config
}

// ─── Segment Builders ──────────────────────────────────────────────────────

func dirSegment(dirName string, active bool) Segment {
	fg := pal.muted
	if active {
		fg = pal.active
	}
	return Segment{
		Content: pal.git + strings.ToLower(dirName),
		FgAnsi:  fg,
	}
}

func gitSegment(git *gitutil.StatusResult, active bool, maxLen int) Segment {
	fg := pal.muted
	if active {
		fg = pal.active
	}
	if git == nil {
		return Segment{}
	}
	isClean := git.Staged == 0 && git.Unstaged == 0 && git.Untracked == 0
	branchColor := pal.red
	if isClean {
		branchColor = pal.green
	}
	branchTextFg := pal.soft
	if active {
		branchTextFg = pal.active
	}
	branch := strings.ToLower(git.Branch)
	if maxLen > 0 {
		runes := []rune(branch)
		if len(runes) > maxLen {
			if maxLen > 3 {
				branch = string(runes[:maxLen-3]) + "..."
			} else {
				branch = string(runes[:maxLen])
			}
		}
	}
	return Segment{
		Content: branchColor + symBranch + " " + branchTextFg + branch,
		FgAnsi:  fg,
	}
}

func projectSegment(project ProjectInfo, active bool) Segment {
	fg := pal.soft
	if active {
		fg = pal.active
	}
	display := project.Badge
	if project.Version != "" {
		display = project.Badge + " " + project.Version
	}
	return Segment{
		Content: fg + strings.ToLower(display),
		FgAnsi:  fg,
	}
}

var (
	claudeModelRe = regexp.MustCompile(`(?i)claude|opus|sonnet|haiku`)
	opusRe        = regexp.MustCompile(`(?i)opus`)
)

func modelSegment(modelName string, active bool) Segment {
	fg := pal.muted
	if active {
		fg = pal.active
	}
	nameColor := fg
	if claudeModelRe.MatchString(modelName) {
		nameColor = pal.orange
		if opusRe.MatchString(modelName) {
			nameColor = pal.paleOrange
		}
	}
	return Segment{
		Content: nameColor + strings.ToLower(modelName),
		FgAnsi:  fg,
	}
}

func contextSegment(ctx *ContextWindowData, maxOutputTokens int, active bool, ctxCfg *config.ContextConfig) (*Segment, ContextHealth) {
	if ctx == nil || ctx.ContextWindowSize == nil || *ctx.ContextWindowSize == 0 {
		return nil, HealthNone
	}

	used, rawPct := getEffectiveTokens(ctx, maxOutputTokens)
	if used == 0 {
		return nil, HealthNone
	}

	// Clamp to 0–100
	remainingPct := max(min(rawPct, 100), 0)
	health := contextHealth(remainingPct, ctxCfg)

	isCritical := remainingPct <= ctxCfg.CriticalPct
	isWarning := remainingPct <= ctxCfg.WarningPct

	color := pal.green
	if isCritical {
		color = pal.red
	} else if isWarning {
		color = pal.yellow
	}

	fg := pal.muted
	if active {
		fg = pal.active
	}

	var statusText string
	if remainingPct == 0 {
		statusText = fg + "ctx:" + ansiBold + pal.red + formatTokens(used) + " !!!" + ansiReset
	} else {
		statusText = fg + "ctx:" + color + formatTokens(used)
	}

	return &Segment{
		Content: statusText,
		FgAnsi:  fg,
	}, health
}

func labelSegment(version string, active bool) Segment {
	fg := pal.muted
	if active {
		fg = pal.active
	}
	content := pal.paleOrange + ansiBold + "cc"
	if version != "" {
		content += " " + pal.muted + version
	}
	content += ansiReset
	return Segment{Content: content, FgAnsi: fg}
}

// ─── Collapse ──────────────────────────────────────────────────────────────

// plainLen returns the visible cell width of a string (excluding ANSI escapes).
// Uses go-runewidth so CJK and East Asian Ambiguous glyphs (●, ›, ⎇) are
// counted as the cells they actually occupy in the terminal.
func plainLen(s string) int {
	n := 0
	inEsc := false
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			inEsc = true
			i++
			continue
		}
		if inEsc {
			if s[i] == 'm' {
				inEsc = false
			}
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		n += runewidth.RuneWidth(r)
		i += size
	}
	return n
}

// collapseBreadcrumbs drops slots from the right until the total
// width fits within termWidth. The first slot is always kept.
func collapseBreadcrumbs(slots []slot, termWidth, separatorPlainWidth, dotWidth int) []slot {
	if len(slots) == 0 {
		return slots
	}

	totalWidth := func(ss []slot) int {
		w := dotWidth
		for i, s := range ss {
			w += s.width
			if i > 0 && !s.noSep {
				w += separatorPlainWidth
			}
		}
		return w
	}

	for len(slots) > 1 && totalWidth(slots) > termWidth {
		slots = slots[:len(slots)-1]
	}

	return slots
}

// ─── Renderer ──────────────────────────────────────────────────────────────

// truncateVisible truncates s so its visible cell width (excluding ANSI escapes)
// does not exceed maxWidth. Preserves ANSI codes that precede kept characters.
// A wide rune that would overflow is dropped entirely — no half-glyph output.
func truncateVisible(s string, maxWidth int) string {
	var b strings.Builder
	visible := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			// Copy the entire escape sequence unconditionally.
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++ // include 'm'
			}
			b.WriteString(s[i:j])
			i = j
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		w := runewidth.RuneWidth(r)
		if visible+w > maxWidth {
			break
		}
		b.WriteString(s[i : i+size])
		visible += w
		i += size
	}
	return b.String()
}

// renderLines groups slots by line, collapses line 1, builds segments, and joins.
func renderLines(dot string, slots []slot, termWidth int, separator string) string {
	sepPlainWidth := plainLen(separator)
	dotWidth := plainLen(dot) + 1 // +1 for the trailing space

	// Group slots by line, preserving order within each line.
	lineMap := map[int][]slot{}
	var lineNums []int
	for _, s := range slots {
		if _, exists := lineMap[s.line]; !exists {
			lineNums = append(lineNums, s.line)
		}
		lineMap[s.line] = append(lineMap[s.line], s)
	}

	var lines []string
	for _, num := range lineNums {
		group := lineMap[num]

		var lb strings.Builder

		// Line 1 only: collapse breadcrumbs and prepend dot.
		if num == 1 {
			group = collapseBreadcrumbs(group, termWidth-1, sepPlainWidth, dotWidth)
			lb.WriteString(dot)
			lb.WriteString(" ")
		}

		// Build segments — last in line gets active=true.
		for i, s := range group {
			active := i == len(group)-1
			seg := s.build(active)
			if s.styleOverride != "" {
				seg.Content = s.styleOverride + stripAnsi(seg.Content)
				seg.FgAnsi = s.styleOverride
			}
			lb.WriteString(seg.FgAnsi)
			lb.WriteString(seg.Content)
			if i+1 < len(group) && !group[i+1].noSep {
				lb.WriteString(pal.separator)
				lb.WriteString(separator)
			}
		}

		lines = append(lines, lb.String())
	}

	var b strings.Builder
	for i, line := range lines {
		isLast := i+1 == len(lines)
		// Non-last lines must stay below termWidth: when the cursor lands
		// in column termWidth the terminal's automatic-margin wraps it,
		// and the subsequent \n then consumes an extra row, pushing line 2
		// off the status area.  The last line needs no such reservation.
		limit := termWidth
		if !isLast {
			// Reserve 2 cells: one for the auto-margin column the terminal
			// refuses to print into without wrapping, plus one for residual
			// width-miscount slack (terminal-emulator ambiguous-width quirks).
			limit = termWidth - 2
		}
		if plainLen(line) > limit {
			line = truncateVisible(line, limit)
		}
		b.WriteString(line)
		if i+1 < len(lines) {
			b.WriteString(ansiReset)
			b.WriteString("\n")
		}
	}
	b.WriteString(ansiReset)
	return b.String()
}

// ─── Main ──────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) == 1 && stdinIsPiped() {
		Run()
		return
	}
	executeRoot()
}

func stdinIsPiped() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}
