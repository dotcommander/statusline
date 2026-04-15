package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	statusLineType    = "command"
	statusLinePadding = 0
	binaryName        = "statusline"
)

func main() {
	scope := "global"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--global":
			scope = "global"
		case "--local":
			scope = "local"
		case "--help", "-h":
			printUsage()
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", arg)
			printUsage()
			os.Exit(1)
		}
	}

	if err := runSetup(scope); err != nil {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: statusline-setup [--global|--local]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Configures Claude Code's settings.json to use the statusline binary.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "  --global  configure ~/.claude/settings.json (default)")
	fmt.Fprintln(os.Stderr, "  --local   configure .claude/settings.json in cwd")
}

// resolveBinaryPath checks ~/go/bin/statusline, then exec.LookPath.
// Falls back to offerBuildSymlink if neither found.
func resolveBinaryPath() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil {
		candidate := filepath.Join(home, "go", "bin", binaryName)
		if info, statErr := os.Stat(candidate); statErr == nil {
			mode := info.Mode()
			if mode.IsRegular() && mode.Perm()&0111 != 0 {
				return candidate, nil
			}
		}
	}

	if looked, err := exec.LookPath(binaryName); err == nil {
		abs, err := filepath.Abs(looked)
		if err == nil {
			return abs, nil
		}
		return looked, nil
	}

	return offerBuildSymlink()
}

// offerBuildSymlink builds the statusline binary from the repo root
// and installs it to ~/go/bin/statusline.
func offerBuildSymlink() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	binDir := filepath.Join(home, "go", "bin")
	target := filepath.Join(binDir, binaryName)

	fmt.Fprintln(os.Stderr, "  Building statusline...")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("create bin directory: %w", err)
	}

	cmd := exec.Command("go", "build", "-o", target, ".")
	cmd.Dir = repoRoot()
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build failed: %w", err)
	}

	return target, nil
}

// repoRoot walks up from cwd to find go.mod.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "."
		}
		dir = parent
	}
}

// claudeSettingsPath returns the path to Claude Code's settings.json.
func claudeSettingsPath(scope string) (string, error) {
	switch scope {
	case "global":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		return filepath.Join(home, ".claude", "settings.json"), nil
	case "local":
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot determine working directory: %w", err)
		}
		return filepath.Join(wd, ".claude", "settings.json"), nil
	default:
		return "", fmt.Errorf("unknown scope: %s", scope)
	}
}

// readSettings reads a JSON file into map[string]any.
// Returns empty map if file does not exist.
func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("read settings: %w", err)
	}

	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse settings: %w", err)
	}

	if settings == nil {
		return make(map[string]any), nil
	}
	return settings, nil
}

// writeSettings atomically writes the settings map to path.
// Creates parent directories if needed.
func writeSettings(path string, settings map[string]any) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename settings file: %w", err)
	}

	return nil
}

// ensureStatusLine sets the statusLine key in the settings map.
// Returns true if the settings were modified.
func ensureStatusLine(settings map[string]any, binaryPath string) bool {
	cmd := formatCommand(binaryPath)

	existing, ok := settings["statusLine"].(map[string]any)
	if ok {
		if existing["type"] == statusLineType &&
			existing["command"] == cmd &&
			existing["padding"] == float64(statusLinePadding) {
			return false
		}
	}

	settings["statusLine"] = map[string]any{
		"type":    statusLineType,
		"command": cmd,
		"padding": float64(statusLinePadding),
	}
	return true
}

// formatCommand normalizes ~/go/bin/statusline to $HOME/go/bin/statusline
// for portability. Returns the absolute path for any other location.
func formatCommand(absPath string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return absPath
	}
	return formatCommandWithHome(absPath, home)
}

// formatCommandWithHome is the testable core of formatCommand.
func formatCommandWithHome(absPath, home string) string {
	if home == "" {
		return absPath
	}
	expected := filepath.Join(home, "go", "bin", binaryName)
	if absPath == expected {
		return "$HOME/go/bin/" + binaryName
	}
	return absPath
}

// runSetup orchestrates the setup flow.
func runSetup(scope string) error {
	binaryPath, err := resolveBinaryPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Statusline Setup")
		fmt.Fprintln(os.Stderr, "================")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "[!] statusline binary not found in ~/go/bin or PATH.")
		fmt.Fprintln(os.Stderr, "    Build it first:  go build -o ~/go/bin/statusline .")
		fmt.Fprintf(os.Stderr, "    Then re-run:     go run ./cmd/setup\n")
		return fmt.Errorf("binary not found: %w", err)
	}

	settingsPath, err := claudeSettingsPath(scope)
	if err != nil {
		return err
	}

	settings, err := readSettings(settingsPath)
	if err != nil {
		return err
	}

	modified := ensureStatusLine(settings, binaryPath)
	if modified {
		if err := writeSettings(settingsPath, settings); err != nil {
			return err
		}
	}

	cmd := formatCommand(binaryPath)
	fmt.Println("Statusline Setup")
	fmt.Println("================")
	fmt.Println()
	fmt.Printf("Binary: %s\n", binaryPath)
	fmt.Printf("Settings: %s\n", settingsPath)
	fmt.Println()

	if modified {
		fmt.Printf("[check] statusLine.command set to %s\n", cmd)
		fmt.Printf("[check] statusLine.padding set to %d\n", statusLinePadding)
	} else {
		fmt.Println("Already configured -- no changes needed.")
	}

	return nil
}
