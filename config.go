package main

import (
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ─── Config ───────────────────────────────────────────────────────────────

// PromptsConfig controls prompt history display.
type PromptsConfig struct {
	Max         int `yaml:"max"`
	NewestWords int `yaml:"newest_words"`
	OlderWords  int `yaml:"older_words"`
	CacheTTL    int `yaml:"cache_ttl"`
}

// ContextConfig controls context window health thresholds.
type ContextConfig struct {
	WarningPct  int `yaml:"warning_pct"`
	CriticalPct int `yaml:"critical_pct"`
	AlertPct    int `yaml:"alert_pct"`
}

// TokenConfig holds per-token appearance overrides.
type TokenConfig struct {
	Style     string `yaml:"style"`
	MaxLength int    `yaml:"max_length"`
}

// Config defines statusline layout and behavior.
type Config struct {
	Line1     string                  `yaml:"line1"`
	Line2     string                  `yaml:"line2"`
	Separator string                  `yaml:"separator"`
	Dot       string                  `yaml:"dot"`
	Prompts   *PromptsConfig          `yaml:"prompts"`
	Context   *ContextConfig          `yaml:"context"`
	Tokens    map[string]*TokenConfig `yaml:"tokens"`
}

var defaultConfig = Config{
	Line1:     "[dir] [prompts]",
	Line2:     "[label] [dc] [model] [ctx] [project] [git]",
	Separator: " › ",
	Dot:       "●",
	Prompts:   &PromptsConfig{Max: 3, NewestWords: 4, OlderWords: 3},
	Context:   &ContextConfig{WarningPct: 25, CriticalPct: 10, AlertPct: 30},
}

var tokenRe = regexp.MustCompile(`\[(\w+)\]`)

// parseTokens extracts ordered token names from a template string.
// "[dir] [prompts]" → ["dir", "prompts"]
func parseTokens(tmpl string) []string {
	matches := tokenRe.FindAllStringSubmatch(tmpl, -1)
	tokens := make([]string, len(matches))
	for i, m := range matches {
		tokens[i] = m[1]
	}
	return tokens
}

// loadConfig reads ~/.config/statusline/config.yaml (or STATUSLINE_CONFIG override).
// Returns defaults if the file is missing or unparseable.
func loadConfig() Config {
	path := os.Getenv("STATUSLINE_CONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return defaultConfig
		}
		path = filepath.Join(home, ".config", "statusline", "config.yaml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig
	}

	// Fill defaults for missing fields
	if cfg.Line1 == "" {
		cfg.Line1 = defaultConfig.Line1
	}
	if cfg.Line2 == "" {
		cfg.Line2 = defaultConfig.Line2
	}
	if cfg.Separator == "" {
		cfg.Separator = defaultConfig.Separator
	}
	if cfg.Dot == "" {
		cfg.Dot = defaultConfig.Dot
	}
	if cfg.Prompts == nil {
		cfg.Prompts = defaultConfig.Prompts
	} else {
		if cfg.Prompts.Max <= 0 {
			cfg.Prompts.Max = defaultConfig.Prompts.Max
		}
		if cfg.Prompts.NewestWords <= 0 {
			cfg.Prompts.NewestWords = defaultConfig.Prompts.NewestWords
		}
		if cfg.Prompts.OlderWords <= 0 {
			cfg.Prompts.OlderWords = defaultConfig.Prompts.OlderWords
		}
	}
	if cfg.Context == nil {
		cfg.Context = defaultConfig.Context
	} else {
		if cfg.Context.WarningPct <= 0 {
			cfg.Context.WarningPct = defaultConfig.Context.WarningPct
		}
		if cfg.Context.CriticalPct <= 0 {
			cfg.Context.CriticalPct = defaultConfig.Context.CriticalPct
		}
		if cfg.Context.AlertPct <= 0 {
			cfg.Context.AlertPct = defaultConfig.Context.AlertPct
		}
	}
	return cfg
}
