# statusline

A two-line ANSI status bar for [Claude Code](https://docs.anthropic.com/en/docs/claude-code). Displays project context, git branch, model name, context window health, and recent prompts — color-coded with a Tokyo Night palette.

```
● statusline › + add per-token style overrides › » /dc:cpt
  cc 1.0.33 › dc:1.6.1 › claude opus 4.6 › ctx:42k › go 1.26 ›  main
```

## Install

```bash
go install github.com/dotcommander/statusline@latest
go install github.com/dotcommander/statusline/cmd/slconfig@latest
```

`statusline` is the renderer; `slconfig` is the interactive configurator.

## Claude Code Hook

Add to your Claude Code `settings.json`:

```json
{
  "statusline": {
    "command": "statusline"
  }
}
```

Claude Code pipes a JSON payload to `statusline` on every render. The binary reads it, renders two lines of ANSI output, and exits.

## Tokens

Each piece of information in the status bar is a **token** — a named placeholder you arrange in `line1` and `line2`.

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
| `[ctx]` | `ctx:42k` token usage | Green → yellow (≤25%) → red (≤10%). Hidden at session start. Shows `ctx:1.2M !!!` when full. |
| `[label]` | `cc 1.0.33` Claude Code version | Always visible |
| `[dc]` | `dc:1.6.1` dotcommander version | Hidden if dotcommander isn't installed |
| `[prompts]` | Up to N recent user prompts | Icons: `›` general, `»` slash cmd, `?` question, `+` create, `×` fix |

> **Note:** Line 1 drops tokens from the right when the terminal is too narrow. Put your most important tokens first.

## Configuration

The config file lives at `~/.config/statusline/config.yaml`. Override the path with `STATUSLINE_CONFIG`.

Run `slconfig` to edit it interactively, or edit the file directly.

### Layout

```yaml
line1: "[dir] [prompts]"
line2: "[label] [dc] [model] [ctx] [project] [git]"
separator: " › "
dot: "●"
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

Prompt slots collapse individually — oldest drops first when space is tight. Set `cache_ttl: 5` to avoid re-reading the transcript on every render.

### Context window

```yaml
context:
  warning_pct: 25   # remaining % → yellow dot and ctx token
  critical_pct: 10  # remaining % → red dot and ctx token
  alert_pct: 30     # remaining % → entire bar turns red
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
| `"bold italic #f7768e"` | Combined — space-separated |

## Layout Examples

### Minimal — project + git only

```yaml
line1: "[dir]"
line2: "[label] [model] [ctx] [project] [git]"
```

### Prompt-heavy — full breadcrumb trail

```yaml
line1: "[dir] [prompts]"
line2: "[label] [dc] [model] [ctx] [project] [git]"
separator: " › "
prompts:
  max: 5
  newest_words: 6
  older_words: 3
```

### Context-focused — ctx on line 1

```yaml
line1: "[ctx] [prompts]"
line2: "[label] [model] [project] [git]"
```

### Model + dir on line 1

```yaml
line1: "[dir] [model] [ctx]"
line2: "[label] [dc] [project] [git]"
```

## slconfig

`slconfig` opens a full-screen TUI for editing your config without touching YAML. It covers five sections: **Layout**, **Appearance**, **Prompts**, **Context**, and **Tokens**. Press `s` to save to `~/.config/statusline/config.yaml`, `q` to quit.

```bash
slconfig
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `STATUSLINE_CONFIG` | Override config file path |
| `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | Max output tokens for context % calculation |
| `CLAUDE_STATUSLINE_ENABLE_PROMPTS` | Set to `0` to hide the prompt trail |
| `COLUMNS` | Override terminal width detection |

## How It Works

Claude Code pipes a JSON payload to stdin containing model info, context window stats, session ID, and transcript path. `statusline` reads this, detects the project type from the working directory, runs `git status`, and renders a two-line ANSI-colored status bar to stdout.

## License

MIT
