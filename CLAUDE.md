# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Two-line ANSI status bar renderer for Claude Code. Receives JSON on stdin (session ID, model, context window stats, transcript path), gathers local state (git, project stack), renders a responsive breadcrumb bar with Tokyo Night theming. Ships two binaries: `statusline` (renderer) and `slconfig` (interactive TUI config editor).

## Build & Test

```bash
go build .                        # statusline binary
go build ./cmd/slconfig           # slconfig TUI editor
go build ./cmd/setup              # setup command binary
go test ./...                     # all tests (3 packages)
go test -run TestPlainLen .       # single test
go test -v -race -count=1 ./...  # verbose with race detector
go vet ./...                      # static analysis
```

Install to PATH:

```bash
go build -o statusline . && ln -sf "$(pwd)/statusline" ~/go/bin/statusline
go build -o slconfig ./cmd/slconfig && ln -sf "$(pwd)/slconfig" ~/go/bin/slconfig
```

No Makefile, Taskfile, or linting config exists yet.

## Architecture

All rendering logic lives in the **root package** (not `internal/`). The pipeline:

1. **input.go** — reads JSON from stdin (Claude Code pipes session data)
2. **config.go** — loads `~/.config/statusline/config.yaml`, merges defaults
3. **project.go** — detects language stack from cwd markers (go.mod, Cargo.toml, package.json, etc.)
4. **internal/gitutil/** — runs `git status --porcelain=v2 --branch` with 1s timeout, parses branch + file counts
5. **tokens.go** — computes context window health (green/yellow/red) from token usage data
6. **prompts.go** — tails transcript JSONL, extracts recent user prompts, classifies with icons, optional file-based cache
7. **theme.go** — palette definitions (normal + alert mode when context is low), hex→true-color ANSI conversion
8. **style.go** — parses per-token style overrides ("bold italic #hex")
9. **main.go** — orchestrates everything: builds deferred `slot` builders per config token, calculates widths, collapses line 1 from right when exceeding terminal width, renders two ANSI lines

**types.go** defines `InputData` (stdin schema) and `StatusOutput` (rendered lines).

### Token System

Layout is configured as token templates: `"[dir] [prompts]"` for line 1, `"[label] [dc] [model] [ctx] [project] [git]"` for line 2. Each token is a named slot with a lazy builder, plain-text width, and optional style override. Line 1 collapses rightmost slots first when space is tight; line 2 does not collapse.

### Panic Recovery

`main()` defers a panic handler that outputs a minimal fallback status (dot + "cc") so Claude Code never sees a crash.

## Config

File: `~/.config/statusline/config.yaml` (override with `STATUSLINE_CONFIG` env var)

Key env vars:
- `CLAUDE_CODE_MAX_OUTPUT_TOKENS` — reserves tokens in context % calculation
- `CLAUDE_STATUSLINE_ENABLE_PROMPTS` — set "0" to disable prompt extraction
- `COLUMNS` — override terminal width detection

## slconfig TUI

`cmd/slconfig/` — bubbletea app with 5 tabs (Layout, Appearance, Prompts, Context, Tokens). Reads and writes the same config.yaml. Entry in `main.go`, all UI in `tui.go`.

## Setup Command

`cmd/setup/` — configures Claude Code's settings.json to use the statusline binary.
Detects/builds the binary, writes the `statusLine` entry, idempotent.

```bash
go run ./cmd/setup              # configure global settings
go run ./cmd/setup --local      # configure project-local settings
go build -o statusline-setup ./cmd/setup && ln -sf "$(pwd)/statusline-setup" ~/go/bin/statusline-setup
```

## Conventions

- Library choices: cobra + viper (CLI), bubbletea + lipgloss (TUI), testify (tests)
- Code style: 80-line function max, 4-level nesting max, config-not-code policy
- Testing: `t.Parallel()` on all tests, table-driven for multiple cases
