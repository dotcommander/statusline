# statusline

A two-line status bar for [Claude Code](https://docs.anthropic.com/en/docs/claude-code), rendered as a standalone binary. Displays project context, git branch, model name, context window health, and recent prompts — all color-coded with a Tokyo Night palette.

```
● statusline › /commit --push
  cc 1.0.30 › opus 4.6 › ctx:42k › go 1.26 ›  main
```

## Features

- **Project detection** — Go, Rust, Python, Node (React/Next/Svelte/Vue/Astro), PHP (Laravel/Symfony)
- **Git status** — branch name with clean/dirty color indicator
- **Context window** — token usage with green/yellow/red health coloring
- **Alert mode** — entire status line turns red when context drops below 30%
- **Prompt trail** — shows up to 3 recent user prompts from the transcript
- **Responsive** — collapses breadcrumb segments to fit terminal width

## Install

```bash
go install github.com/dotcommander/statusline@latest
```

## Configure Claude Code

Add to your Claude Code `settings.json`:

```json
{
  "statusline": {
    "command": "statusline"
  }
}
```

## Environment Variables

| Variable | Description |
|---|---|
| `CLAUDE_CODE_MAX_OUTPUT_TOKENS` | Max output tokens (used for context % calculation) |
| `CLAUDE_STATUSLINE_ENABLE_PROMPTS` | Set to `0` to hide the prompt trail |
| `COLUMNS` | Override terminal width detection |

## How It Works

Claude Code pipes a JSON payload to stdin containing model info, context window stats, session ID, and transcript path. The binary reads this, detects the project type from the working directory, runs `git status`, and renders a two-line ANSI-colored status bar to stdout.

## License

MIT
