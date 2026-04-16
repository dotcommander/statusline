# statusline

A two-line ANSI status bar for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Displays project context, git branch, model name, context window health, and recent prompts -- color-coded with a Tokyo Night palette.

```
● statusline › + add per-token style overrides › /dc:cpt
  cc 1.0.33 › dc:1.6.1 › claude opus 4.6 › ctx:42k › go 1.26 ›  main
```

Line 1 collapses from the right when the terminal is too narrow. Line 2 stays fixed.

## Install

```bash
go install github.com/dotcommander/statusline@latest
```

Or build from source:

```bash
git clone https://github.com/dotcommander/statusline.git
cd statusline
go build -o statusline . && ln -sf "$(pwd)/statusline" ~/go/bin/statusline
```

## Setup

The `setup` subcommand configures Claude Code's `settings.json` to use statusline. It detects the binary in `~/go/bin`, falls back to `PATH`, or builds from source if needed.

```bash
statusline setup                # configure global settings (~/.claude/settings.json)
statusline setup --local        # configure project-local settings (.claude/settings.json)
```

Idempotent -- running it again when already configured reports "no changes needed."

## Tokens

Each piece of information in the status bar is a **token** -- a named placeholder you arrange in `line1` and `line2`.

```
● statusline › + add error handling › ctx:42k
  cc 1.0.33 › claude sonnet 4.6 › ctx:42k › go 1.26 ›  feat/tokens
```

| Token | Shows | Notes |
|-------|-------|-------|
| `[dir]` | cwd basename, lowercased | Blue (#7aa2f7) |
| `[git]` | branch name | Green = clean, red = dirty. Hidden outside git repos. |
| `[project]` | detected stack + version | Go, Rust, Python, Node (React/Next/Svelte/Vue/Astro), PHP (Laravel/Symfony) |
| `[model]` | active model name, lowercased | Opus = paleOrange, Sonnet/Haiku = orange, others = muted |
| `[ctx]` | `ctx:42k` token usage | Green, yellow (<=25% remaining), red (<=10%). Hidden at session start. Shows `ctx:1.2M !!!` when full. |
| `[label]` | `cc 1.0.33` Claude Code version | Always visible |
| `[dc]` | `dc:1.6.1` dotcommander version | Hidden if dotcommander isn't installed |
| `[prompts]` | Up to N recent user prompts | Icons: `>` general, `>>` slash cmd, `?` question, `+` create, `x` fix |

> Line 1 drops tokens from the right when the terminal is too narrow. Put your most important tokens first.

## Configuration

Config file: `~/.config/statusline/config.yaml` (override with `STATUSLINE_CONFIG`).

Run `statusline config` to edit interactively, or edit the file directly.

### Layout

```yaml
line1: "[dir] [prompts]"
line2: "[label] [dc] [model] [ctx] [project] [git]"
separator: " > "
dot: ">"
```

`separator` appears between every token. `dot` is the leading bullet on line 1.

### Prompts

```yaml
prompts:
  max: 3           # number of recent prompts to show
  newest_words: 4  # word count for the most recent prompt
  older_words: 3   # word count for older prompts
  cache_ttl: 0     # seconds; 0 = disabled; recommended: 5
```

Prompt slots collapse individually -- oldest drops first when space is tight. Set `cache_ttl: 5` to avoid re-reading the transcript on every render.

### Context window

```yaml
context:
  warning_pct: 25   # remaining % -> yellow dot and ctx token
  critical_pct: 10  # remaining % -> red dot and ctx token
  alert_pct: 30     # remaining % -> entire bar turns red
```

When context remaining falls below `alert_pct`, the entire status bar renders in red regardless of individual token colors.

### Per-token overrides

```yaml
tokens:
  git:
    max_length: 24        # truncates branch name with "..."
    style: "bold"
  model:
    style: "italic #ff9e64"
  dir:
    style: "bold #7aa2f7"
```

The `style` field accepts keywords, hex colors, or both combined. Setting it replaces all default colors for that token.

**Style syntax:**

| Value | Effect |
|-------|--------|
| `"bold"` | Bold text |
| `"italic"` | Italic text |
| `"underline"` | Underline |
| `"dim"` | Dimmed |
| `"strikethrough"` | Strikethrough |
| `"#f7768e"` | Hex color |
| `"bold italic #f7768e"` | Combined -- space-separated |

## Layout Examples

### Minimal -- project + git only

```yaml
line1: "[dir]"
line2: "[label] [model] [ctx] [project] [git]"
```

### Prompt-heavy -- full breadcrumb trail

```yaml
line1: "[dir] [prompts]"
line2: "[label] [dc] [model] [ctx] [project] [git]"
separator: " > "
prompts:
  max: 5
  newest_words: 6
  older_words: 3
```

### Context-focused -- ctx on line 1

```yaml
line1: "[ctx] [prompts]"
line2: "[label] [model] [project] [git]"
```

### Model + dir on line 1

```yaml
line1: "[dir] [model] [ctx]"
line2: "[label] [dc] [project] [git]"
```

## statusline config

`statusline config` opens a full-screen TUI for editing your config without touching YAML. Five tabs: **Layout**, **Appearance**, **Prompts**, **Context**, and **Tokens**. Press `s` to save, `q` to quit.

```bash
statusline config
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `STATUSLINE_CONFIG` | Override config file path |
| `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | Max output tokens for context % calculation |
| `CLAUDE_STATUSLINE_ENABLE_PROMPTS` | Set to `0` to hide the prompt trail |
| `COLUMNS` | Override terminal width detection |

## How It Works

Claude Code pipes a JSON payload to stdin containing model info, context window stats, session ID, and transcript path. `statusline` reads this, detects the project type from the working directory, runs `git status` (with a 1s timeout), tails the transcript for recent prompts, and renders a two-line ANSI-colored status bar to stdout.

The rendering pipeline:

1. **input.go** -- reads JSON from stdin
2. **config.go** -- loads config, parses token templates
3. **project.go** -- detects language stack from directory markers
4. **dc.go** -- detects dotcommander version
5. **internal/gitutil/** -- runs `git status --porcelain=v2 --branch`, parses branch + file counts
6. **tokens.go** -- computes context window health from token usage
7. **prompts.go** -- tails transcript JSONL, extracts user prompts, classifies with icons
8. **theme.go** -- Tokyo Night palette (normal + alert mode)
9. **style.go** -- per-token style overrides
10. **main.go** -- orchestrates rendering: builds deferred slot builders, collapses line 1, outputs two ANSI lines

Panic recovery ensures Claude Code never sees a crash -- the fallback output is a dot and "cc".

## License

MIT
