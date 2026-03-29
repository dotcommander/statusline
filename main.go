package main

import (
	"fmt"
	"os"
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
	line  int                       // which output line (1, 2, ...)
	width int                       // plain-text width for collapse calc
	build func(active bool) Segment // lazy builder
	noSep bool                      // skip separator before this slot
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

func gitSegment(git *gitutil.StatusResult, active bool) Segment {
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
	return Segment{
		Content: branchColor + symBranch + " " + branchTextFg + strings.ToLower(git.Branch),
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

func contextSegment(ctx *ContextWindowData, maxOutputTokens int, active bool) (*Segment, ContextHealth) {
	if ctx == nil || ctx.ContextWindowSize == nil || *ctx.ContextWindowSize == 0 {
		return nil, HealthNone
	}

	used, rawPct := getEffectiveTokens(ctx, maxOutputTokens)
	if used == 0 {
		return nil, HealthNone
	}

	// Clamp to 0–100
	remainingPct := max(min(rawPct, 100), 0)
	health := contextHealth(remainingPct)

	isCritical := remainingPct <= contextCriticalPct
	isWarning := remainingPct <= contextWarningPct

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

func promptSegment(prompts []string) *Segment {
	if len(prompts) == 0 {
		return nil
	}

	const (
		maxPromptsToShow     = 3
		mostRecentWordCount  = 4
		olderPromptWordCount = 3
	)

	var parts []string
	shown := 0
	for i := len(prompts) - 1; i >= 0 && shown < maxPromptsToShow; i-- {
		prompt := prompts[i]
		icon := getPromptIcon(prompt)
		wordCount := olderPromptWordCount
		color := pal.gray
		style := ansiItalic
		if shown == 0 {
			wordCount = mostRecentWordCount
			color = pal.prompt
			style = ""
		}
		parts = append(parts, color+style+icon+" "+strings.ToLower(truncateWords(prompt, wordCount)))
		shown++
	}

	return &Segment{
		Content: "  " + strings.Join(parts, " "),
		FgAnsi:  pal.prompt,
	}
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

const separatorPlainWidth = 3 // ` › ` is 3 visible chars

// collapseBreadcrumbs drops slots from the right until the total
// width fits within termWidth. The first slot is always kept.
func collapseBreadcrumbs(slots []slot, termWidth int) []slot {
	if len(slots) == 0 {
		return slots
	}

	// dot (2 chars: "● ") + segments + separators between them
	const dotWidth = 2

	totalWidth := func(ss []slot) int {
		w := dotWidth
		for i, s := range ss {
			w += s.width
			if i > 0 {
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

// renderLines groups slots by line, collapses line 1, builds segments, and joins.
func renderLines(dot string, slots []slot, termWidth int) string {
	// Group slots by line, preserving order within each line.
	lineMap := map[int][]slot{}
	var lineNums []int
	for _, s := range slots {
		if _, exists := lineMap[s.line]; !exists {
			lineNums = append(lineNums, s.line)
		}
		lineMap[s.line] = append(lineMap[s.line], s)
	}

	var b strings.Builder
	for li, num := range lineNums {
		group := lineMap[num]

		// Line 1 only: collapse breadcrumbs and prepend dot.
		if num == 1 {
			group = collapseBreadcrumbs(group, termWidth)
			b.WriteString(dot)
			b.WriteString(" ")
		}

		// Build segments — last in line gets active=true.
		for i, s := range group {
			active := i == len(group)-1
			seg := s.build(active)
			b.WriteString(seg.FgAnsi)
			b.WriteString(seg.Content)
			if i+1 < len(group) && !group[i+1].noSep {
				b.WriteString(pal.separator)
				b.WriteString(symSeparator)
			}
		}

		if li+1 < len(lineNums) {
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
			fmt.Print(pal.active + symDot + ansiReset + "\n" + pal.muted + ansiBold + "cc" + ansiReset)
		}
	}()

	pal = normalPalette()
	data := readInput()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	// Directory name from cwd
	dirName := "unknown"
	if parts := strings.Split(cwd, "/"); len(parts) > 0 {
		if last := parts[len(parts)-1]; last != "" {
			dirName = last
		}
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
		if remainingPct <= contextAlertPct {
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
		prompts = fetchPrompts(data.TranscriptPath)
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

	gitPlain := 0
	if git != nil {
		gitPlain = utf8.RuneCountInString(symBranch) + 1 + utf8.RuneCountInString(git.Branch)
	}

	// Build declarative layout slots
	// Line 1: dir, prompts
	slots := []slot{
		{line: 1, width: dirPlain, build: func(active bool) Segment { return dirSegment(dirName, active) }},
	}

	// Conditional prompts on line 1
	if enablePrompts {
		promptSeg := promptSegment(prompts)
		if promptSeg != nil {
			promptPlain := plainLen(promptSeg.Content)
			captured := *promptSeg
			slots = append(slots, slot{
				line: 1, width: promptPlain, noSep: true,
				build: func(_ bool) Segment { return captured },
			})
		}
	}

	// Context health (needed for dot color)
	ctxSeg, ctxHealth := contextSegment(data.ContextWindow, maxOutputTokens, false)
	health = ctxHealth

	// Line 2: label, model, ctx, project, git
	labelPlain := plainLen("cc") // 2
	if data.Version != "" {
		labelPlain = plainLen("cc " + data.Version)
	}
	slots = append(slots, slot{
		line: 2, width: labelPlain,
		build: func(active bool) Segment { return labelSegment(data.Version, active) },
	})

	slots = append(slots, slot{
		line: 2, width: modelPlain,
		build: func(active bool) Segment { return modelSegment(modelName, active) },
	})

	if ctxSeg != nil {
		ctxPlain := plainLen(ctxSeg.Content)
		slots = append(slots, slot{
			line: 2, width: ctxPlain,
			build: func(active bool) Segment {
				seg, _ := contextSegment(data.ContextWindow, maxOutputTokens, active)
				return *seg
			},
		})
	}

	slots = append(slots, slot{
		line: 2, width: projectPlain,
		build: func(active bool) Segment { return projectSegment(project, active) },
	})

	if git != nil {
		slots = append(slots, slot{
			line: 2, width: gitPlain,
			build: func(active bool) Segment { return gitSegment(git, active) },
		})
	}

	// Render all lines
	termWidth := getTerminalWidth()
	dot := dotColor(health) + symDot + ansiReset
	fmt.Print(renderLines(dot, slots, termWidth))
}
