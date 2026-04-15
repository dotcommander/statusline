package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ─── DC Plugin Detection ─────────────────────────────────────────────────

// detectDCVersion reads the Claude Code installed_plugins.json and returns
// the dc@dotcommander plugin version, or "" if not installed.
// The [dc] token is hidden when dotcommander is not installed — this is optional.
func detectDCVersion() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(home, ".claude", "plugins", "installed_plugins.json"))
	if err != nil {
		return ""
	}

	var reg struct {
		Plugins map[string][]struct {
			Version string `json:"version"`
		} `json:"plugins"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		return ""
	}

	entries := reg.Plugins["dc@dotcommander"]
	if len(entries) == 0 {
		return ""
	}
	return entries[0].Version
}

func dcSegment(version string, active bool) Segment {
	fg := pal.muted
	if active {
		fg = pal.active
	}
	content := pal.paleOrange + "dc"
	if version != "" {
		content += fg + ":" + pal.soft + strings.ToLower(version)
	}
	return Segment{Content: content, FgAnsi: fg}
}
