package main

import (
	"regexp"

	"github.com/dotcommander/statusline/internal/config"
)

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

// loadConfig reads the config file and returns a populated Config.
func loadConfig() config.Config {
	return config.Load(config.DefaultPath())
}
