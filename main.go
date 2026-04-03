package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dotcommander/statusline/internal/gitutil"
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
	anthropicRe = regexp.MustCompile(`(?i)claude|opus|sonnet|haiku`)
	opusRe      = regexp.MustCompile(`(?i)opus`)
)

func modelSegment(modelName string, active bool) Segment {
	fg := pal.muted
	if active {
		fg = pal.active
	}
	nameColor := fg
	if anthropicRe.MatchString(modelName) {
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

func contextSegment(ctx *ContextWindowData, maxOutputTokens int, active bool, ctxCfg *ContextConfig) (*Segment, ContextHealth) {
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

// plainLen returns the visible character count of a string (excluding ANSI escapes).
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
		_, size := utf8.DecodeRuneInString(s[i:])
		n++
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

// truncateVisible truncates s so its visible width (excluding ANSI escapes)
// does not exceed maxWidth. Preserves ANSI codes that precede kept characters.
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
		if visible >= maxWidth {
			break
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		b.WriteString(s[i : i+size])
		visible++
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
			group = collapseBreadcrumbs(group, termWidth, sepPlainWidth, dotWidth)
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

		line := lb.String()
		// Safety: truncate any line that would wrap the terminal
		if plainLen(line) > termWidth {
			line = truncateVisible(line, termWidth)
		}
		lines = append(lines, line)
	}

	var b strings.Builder
	for i, line := range lines {
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
	defer func() {
		if r := recover(); r != nil {
			fmt.Print(pal.active + "●" + ansiReset + "\n" + pal.muted + ansiBold + "cc" + ansiReset)
		}
	}()

	pal = normalPalette()
	cfg := loadConfig()
	data := readInput()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Directory name from cwd
	dirName := filepath.Base(cwd)
	if dirName == "." || dirName == "/" {
		dirName = "unknown"
	}

	// Parse env vars
	maxOutputTokens := 0
	if s := os.Getenv("CLAUDE_CODE_MAX_OUTPUT_TOKENS"); s != "" {
		if v, err := strconv.Atoi(s); err == nil {
			maxOutputTokens = v
		}
	}
	enablePrompts := os.Getenv("CLAUDE_STATUSLINE_ENABLE_PROMPTS") != "0"

	// Select palette based on context window health
	if data.ContextWindow != nil && data.ContextWindow.ContextWindowSize != nil && *data.ContextWindow.ContextWindowSize > 0 {
		_, remainingPct := getEffectiveTokens(data.ContextWindow, maxOutputTokens)
		if remainingPct <= cfg.Context.AlertPct {
			pal = alertPalette()
		}
	}

	// Gather data
	var git *gitutil.StatusResult
	if s, err := gitutil.GetStatus(cwd); err == nil {
		git = &s
	}
	project := detectProject(cwd)

	var prompts []string
	if enablePrompts {
		prompts = fetchPromptsWithCache(data.TranscriptPath, cfg.Prompts.CacheTTL)
	}

	modelName := "claude"
	if data.Model != nil && data.Model.DisplayName != "" {
		modelName = data.Model.DisplayName
	}

	// Compute context health for dot color
	var health ContextHealth = HealthNone

	// Build plain widths for collapse calculation
	dirPlain := utf8.RuneCountInString(strings.ToLower(dirName))

	projectPlain := utf8.RuneCountInString(strings.ToLower(project.Badge))
	if project.Version != "" {
		projectPlain += 1 + utf8.RuneCountInString(project.Version)
	}

	modelPlain := utf8.RuneCountInString(strings.ToLower(modelName))

	maxBranchLen := 0
	if tc := cfg.Tokens["git"]; tc != nil {
		maxBranchLen = tc.MaxLength
	}

	gitPlain := 0
	if git != nil {
		branchLen := utf8.RuneCountInString(git.Branch)
		if maxBranchLen > 0 && branchLen > maxBranchLen {
			branchLen = maxBranchLen
		}
		gitPlain = utf8.RuneCountInString(symBranch) + 1 + branchLen
	}

	dcVersion := detectDCVersion()
	dcPlain := plainLen("dc")
	if dcVersion != "" {
		dcPlain = plainLen("dc:" + dcVersion)
	}

	// Context health (needed for dot color regardless of config)
	ctxSeg, ctxHealth := contextSegment(data.ContextWindow, maxOutputTokens, false, cfg.Context)
	health = ctxHealth

	labelPlain := plainLen("cc")
	if data.Version != "" {
		labelPlain = plainLen("cc " + data.Version)
	}

	emitSlots := func(token string, line int) []slot {
		switch token {
		case "dir":
			return []slot{{line: line, width: dirPlain,
				build: func(active bool) Segment { return dirSegment(dirName, active) }}}

		case "prompts":
			if !enablePrompts || len(prompts) == 0 {
				return nil
			}
			var out []slot
			shown := 0
			for i := len(prompts) - 1; i >= 0 && shown < cfg.Prompts.Max; i-- {
				prompt := prompts[i]
				icon := getPromptIcon(prompt)
				wordCount := cfg.Prompts.OlderWords
				color := pal.gray
				style := ansiItalic
				prefix := " "
				if shown == 0 {
					wordCount = cfg.Prompts.NewestWords
					color = pal.prompt
					style = ""
					prefix = "  "
				}
				content := prefix + color + style + icon + " " + strings.ToLower(truncateWords(prompt, wordCount))
				seg := Segment{Content: content, FgAnsi: pal.prompt}
				captured := seg
				out = append(out, slot{
					line: line, width: plainLen(content), noSep: true,
					build: func(_ bool) Segment { return captured },
				})
				shown++
			}
			return out

		case "label":
			return []slot{{line: line, width: labelPlain,
				build: func(active bool) Segment { return labelSegment(data.Version, active) }}}

		case "model":
			return []slot{{line: line, width: modelPlain,
				build: func(active bool) Segment { return modelSegment(modelName, active) }}}

		case "ctx":
			if ctxSeg == nil {
				return nil
			}
			ctxPlain := plainLen(ctxSeg.Content)
			return []slot{{line: line, width: ctxPlain,
				build: func(active bool) Segment {
					seg, _ := contextSegment(data.ContextWindow, maxOutputTokens, active, cfg.Context)
					return *seg
				}}}

		case "project":
			return []slot{{line: line, width: projectPlain,
				build: func(active bool) Segment { return projectSegment(project, active) }}}

		case "git":
			if git == nil {
				return nil
			}
			return []slot{{line: line, width: gitPlain,
				build: func(active bool) Segment { return gitSegment(git, active, maxBranchLen) }}}

		case "dc":
			if dcVersion == "" {
				return nil
			}
			return []slot{{line: line, width: dcPlain,
				build: func(active bool) Segment { return dcSegment(dcVersion, active) }}}
		}
		return nil
	}

	var slots []slot
	appendTokenSlots := func(token string, line int) {
		ss := emitSlots(token, line)
		if tc := cfg.Tokens[token]; tc != nil && tc.Style != "" {
			override := parseStyle(tc.Style)
			for i := range ss {
				ss[i].styleOverride = override
			}
		}
		slots = append(slots, ss...)
	}
	for _, token := range parseTokens(cfg.Line1) {
		appendTokenSlots(token, 1)
	}
	for _, token := range parseTokens(cfg.Line2) {
		appendTokenSlots(token, 2)
	}

	// Render all lines
	termWidth := getTerminalWidth()
	dot := dotColor(health) + cfg.Dot + ansiReset
	fmt.Print(renderLines(dot, slots, termWidth, cfg.Separator))
}
