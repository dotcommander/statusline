package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

// ─── Config types (mirrored from parent package) ──────────────────────────

type PromptsConfig struct {
	Max         int `yaml:"max"`
	NewestWords int `yaml:"newest_words"`
	OlderWords  int `yaml:"older_words"`
	CacheTTL    int `yaml:"cache_ttl"`
}

type ContextConfig struct {
	WarningPct  int `yaml:"warning_pct"`
	CriticalPct int `yaml:"critical_pct"`
	AlertPct    int `yaml:"alert_pct"`
}

type TokenConfig struct {
	Style     string `yaml:"style,omitempty"`
	MaxLength int    `yaml:"max_length,omitempty"`
}

type Config struct {
	Line1     string                  `yaml:"line1"`
	Line2     string                  `yaml:"line2"`
	Separator string                  `yaml:"separator"`
	Dot       string                  `yaml:"dot"`
	Prompts   *PromptsConfig          `yaml:"prompts"`
	Context   *ContextConfig          `yaml:"context"`
	Tokens    map[string]*TokenConfig `yaml:"tokens,omitempty"`
}

var defaultConfig = Config{
	Line1:     "[dir] [prompts]",
	Line2:     "[label] [dc] [model] [ctx] [project] [git]",
	Separator: " › ",
	Dot:       "●",
	Prompts:   &PromptsConfig{Max: 3, NewestWords: 4, OlderWords: 3},
	Context:   &ContextConfig{WarningPct: 25, CriticalPct: 10, AlertPct: 30},
}

// ─── Config I/O ───────────────────────────────────────────────────────────

func configPath() string {
	if p := os.Getenv("STATUSLINE_CONFIG"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "statusline", "config.yaml")
}

func loadConfig(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig
	}
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

func saveConfig(path string, cfg Config) error {
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

	header := "# Statusline configuration — edited by slconfig\n" +
		"# Full reference: ~/.config/statusline/config.yaml.example\n\n"

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(header+string(data)), 0644)
}

// ─── Entry point ──────────────────────────────────────────────────────────

func main() {
	path := configPath()
	if path == "" {
		fmt.Fprintln(os.Stderr, "cannot determine config path")
		os.Exit(1)
	}

	cfg := loadConfig(path)
	m := newModel(cfg, path)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
