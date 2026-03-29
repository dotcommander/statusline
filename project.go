package main

import (
	"encoding/json"
	"maps"
	"os"
	"regexp"
	"strings"
)

// ─── Project Detection ─────────────────────────────────────────────────────

var projectMarkers = []struct {
	file  string
	badge string
}{
	{"go.mod", "Go"},
	{"Cargo.toml", "Rust"},
	{"pyproject.toml", "Python"},
	{"requirements.txt", "Python"},
	{"composer.json", "PHP"},
	{"package.json", "Node"},
}

var jsFrameworks = []struct {
	key   string
	badge string
}{
	{"@sveltejs/kit", "SvelteKit"},
	{"svelte", "Svelte"},
	{"next", "Next.js"},
	{"react", "React"},
	{"vue", "Vue"},
	{"astro", "Astro"},
}

var phpFrameworks = []struct {
	key   string
	badge string
}{
	{"laravel/framework", "Laravel"},
	{"symfony/framework-bundle", "Symfony"},
}

var goVersionRe = regexp.MustCompile(`(?m)^go\s+(\d+\.\d+)`)

// detectProject reads the current directory and identifies the project type.
func detectProject(cwd string) ProjectInfo {
	entries, err := os.ReadDir(cwd)
	if err != nil {
		if strings.Contains(cwd, "/.claude") {
			return ProjectInfo{Badge: "Claude"}
		}
		return ProjectInfo{Badge: "Project"}
	}

	// Build a set for O(1) lookups
	fileSet := make(map[string]struct{}, len(entries))
	for _, e := range entries {
		fileSet[e.Name()] = struct{}{}
	}

	for _, m := range projectMarkers {
		if _, ok := fileSet[m.file]; !ok {
			continue
		}
		switch m.file {
		case "go.mod":
			return detectGoProject(cwd + "/go.mod")
		case "package.json":
			return detectNodeProject(cwd + "/package.json")
		case "composer.json":
			return detectPhpProject(cwd + "/composer.json")
		default:
			return ProjectInfo{Badge: m.badge}
		}
	}

	if strings.Contains(cwd, "/.claude") {
		return ProjectInfo{Badge: "Claude"}
	}
	return ProjectInfo{Badge: "Project"}
}

func detectGoProject(path string) ProjectInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectInfo{Badge: "Go"}
	}
	m := goVersionRe.FindSubmatch(data)
	if m != nil {
		return ProjectInfo{Badge: "Go", Version: string(m[1])}
	}
	return ProjectInfo{Badge: "Go"}
}

func detectNodeProject(path string) ProjectInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectInfo{Badge: "Node"}
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ProjectInfo{Badge: "Node"}
	}
	// Merge deps
	deps := make(map[string]string)
	maps.Copy(deps, pkg.Dependencies)
	maps.Copy(deps, pkg.DevDependencies)
	for _, f := range jsFrameworks {
		if v, ok := deps[f.key]; ok {
			return ProjectInfo{Badge: f.badge, Version: cleanVersion(v)}
		}
	}
	return ProjectInfo{Badge: "Node"}
}

func detectPhpProject(path string) ProjectInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectInfo{Badge: "PHP"}
	}
	var pkg struct {
		Require map[string]string `json:"require"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ProjectInfo{Badge: "PHP"}
	}
	for _, f := range phpFrameworks {
		if v, ok := pkg.Require[f.key]; ok {
			return ProjectInfo{Badge: f.badge, Version: cleanVersion(v)}
		}
	}
	return ProjectInfo{Badge: "PHP"}
}

func cleanVersion(v string) string {
	// Strip leading ^~>=< characters
	return strings.TrimLeft(v, "^~>=<")
}
