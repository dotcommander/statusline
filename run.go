package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/dotcommander/statusline/internal/gitutil"
)

// Run is the hot-path renderer. It reads JSON from stdin, gathers local state,
// and renders the two-line ANSI status bar to stdout.
func Run() {
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
	dirPlain := plainLen(strings.ToLower(dirName))

	projectPlain := plainLen(strings.ToLower(project.Badge))
	if project.Version != "" {
		projectPlain += 1 + utf8.RuneCountInString(project.Version)
	}

	modelPlain := plainLen(strings.ToLower(modelName))

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
		gitPlain = plainLen(symBranch) + 1 + branchLen
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
