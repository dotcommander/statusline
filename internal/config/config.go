package config

import (
	"os"
	"path/filepath"

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
	Style     string `yaml:"style,omitempty"`
	MaxLength int    `yaml:"max_length,omitempty"`
}

// Config defines statusline layout and behavior.
type Config struct {
	Line1     string                  `yaml:"line1"`
	Line2     string                  `yaml:"line2"`
	Separator string                  `yaml:"separator"`
	Dot       string                  `yaml:"dot"`
	Prompts   *PromptsConfig          `yaml:"prompts"`
	Context   *ContextConfig          `yaml:"context"`
	Tokens    map[string]*TokenConfig `yaml:"tokens,omitempty"`
}

// DefaultConfig is the baseline configuration used when fields are missing.
var DefaultConfig = Config{
	Line1:     "[dir] [prompts]",
	Line2:     "[label] [dc] [model] [ctx] [project] [git]",
	Separator: " › ",
	Dot:       "●",
	Prompts:   &PromptsConfig{Max: 3, NewestWords: 4, OlderWords: 3},
	Context:   &ContextConfig{WarningPct: 25, CriticalPct: 10, AlertPct: 30},
}

// DefaultPath returns the config file path, honoring the STATUSLINE_CONFIG
// environment variable override.
func DefaultPath() string {
	if p := os.Getenv("STATUSLINE_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "statusline", "config.yaml")
}

// Load reads the config file at path and returns a Config with defaults merged in.
// Returns DefaultConfig if the file is missing or unparseable.
func Load(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig
	}

	// Fill defaults for missing fields
	if cfg.Line1 == "" {
		cfg.Line1 = DefaultConfig.Line1
	}
	if cfg.Line2 == "" {
		cfg.Line2 = DefaultConfig.Line2
	}
	if cfg.Separator == "" {
		cfg.Separator = DefaultConfig.Separator
	}
	if cfg.Dot == "" {
		cfg.Dot = DefaultConfig.Dot
	}
	if cfg.Prompts == nil {
		cfg.Prompts = DefaultConfig.Prompts
	} else {
		if cfg.Prompts.Max <= 0 {
			cfg.Prompts.Max = DefaultConfig.Prompts.Max
		}
		if cfg.Prompts.NewestWords <= 0 {
			cfg.Prompts.NewestWords = DefaultConfig.Prompts.NewestWords
		}
		if cfg.Prompts.OlderWords <= 0 {
			cfg.Prompts.OlderWords = DefaultConfig.Prompts.OlderWords
		}
	}
	if cfg.Context == nil {
		cfg.Context = DefaultConfig.Context
	} else {
		if cfg.Context.WarningPct <= 0 {
			cfg.Context.WarningPct = DefaultConfig.Context.WarningPct
		}
		if cfg.Context.CriticalPct <= 0 {
			cfg.Context.CriticalPct = DefaultConfig.Context.CriticalPct
		}
		if cfg.Context.AlertPct <= 0 {
			cfg.Context.AlertPct = DefaultConfig.Context.AlertPct
		}
	}
	return cfg
}

// Save writes cfg to path with a YAML header comment.
func Save(path string, cfg Config) error {
	// Clean empty token overrides
	for name, tc := range cfg.Tokens {
		if tc.Style == "" && tc.MaxLength == 0 {
			delete(cfg.Tokens, name)
		}
	}
	if len(cfg.Tokens) == 0 {
		cfg.Tokens = nil
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	header := "# Statusline configuration — edited by `statusline config`\n" +
		"# Full reference: ~/.config/statusline/config.yaml.example\n\n"

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(header+string(data)), 0644)
}
