package setupcmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFormatCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		home  string
		want  string
	}{
		{
			name:  "home go bin path uses dollar HOME",
			input: "/home/user/go/bin/statusline",
			home:  "/home/user",
			want:  "$HOME/go/bin/statusline",
		},
		{
			name:  "other path returns absolute",
			input: "/usr/local/bin/statusline",
			home:  "/home/user",
			want:  "/usr/local/bin/statusline",
		},
		{
			name:  "subdirectory of go bin returns absolute",
			input: "/home/user/go/bin/some/sub/dir",
			home:  "/home/user",
			want:  "/home/user/go/bin/some/sub/dir",
		},
		{
			name:  "empty home returns input unchanged",
			input: "/usr/local/bin/statusline",
			home:  "",
			want:  "/usr/local/bin/statusline",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := formatCommandWithHome(tt.input, tt.home)
			if got != tt.want {
				t.Errorf("formatCommandWithHome(%q, %q) = %q, want %q", tt.input, tt.home, got, tt.want)
			}
		})
	}
}

func TestEnsureStatusLine(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	homeBinPath := filepath.Join(home, "go", "bin", "statusline")
	expectedCmd := "$HOME/go/bin/statusline"

	t.Run("empty settings adds statusLine", func(t *testing.T) {
		t.Parallel()
		settings := make(map[string]any)
		modified := ensureStatusLine(settings, homeBinPath)
		if !modified {
			t.Fatal("expected modified=true for empty settings")
		}
		sl := settings["statusLine"].(map[string]any)
		if sl["type"] != "command" {
			t.Errorf("type = %v, want command", sl["type"])
		}
		if sl["command"] != expectedCmd {
			t.Errorf("command = %v, want %s", sl["command"], expectedCmd)
		}
		if sl["padding"] != float64(0) {
			t.Errorf("padding = %v, want 0", sl["padding"])
		}
	})

	t.Run("already correct returns false", func(t *testing.T) {
		t.Parallel()
		settings := map[string]any{
			"statusLine": map[string]any{
				"type":    "command",
				"command": expectedCmd,
				"padding": float64(0),
			},
		}
		modified := ensureStatusLine(settings, homeBinPath)
		if modified {
			t.Fatal("expected modified=false when already configured")
		}
	})

	t.Run("different command updates", func(t *testing.T) {
		t.Parallel()
		settings := map[string]any{
			"statusLine": map[string]any{
				"type":    "command",
				"command": "/usr/local/bin/statusline",
				"padding": float64(0),
			},
		}
		modified := ensureStatusLine(settings, homeBinPath)
		if !modified {
			t.Fatal("expected modified=true when command differs")
		}
		sl := settings["statusLine"].(map[string]any)
		if sl["command"] != expectedCmd {
			t.Errorf("command = %v, want %s", sl["command"], expectedCmd)
		}
	})

	t.Run("wrong type updates", func(t *testing.T) {
		t.Parallel()
		settings := map[string]any{
			"statusLine": map[string]any{
				"type":    "other",
				"command": expectedCmd,
				"padding": float64(0),
			},
		}
		modified := ensureStatusLine(settings, homeBinPath)
		if !modified {
			t.Fatal("expected modified=true when type differs")
		}
		sl := settings["statusLine"].(map[string]any)
		if sl["type"] != "command" {
			t.Errorf("type = %v, want command", sl["type"])
		}
	})
}

func TestReadSettings(t *testing.T) {
	t.Parallel()

	t.Run("missing file returns empty map", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		settings, err := readSettings(filepath.Join(dir, "nonexistent.json"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(settings) != 0 {
			t.Errorf("got %d keys, want 0", len(settings))
		}
	})

	t.Run("valid file reads correctly", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "settings.json")
		original := map[string]any{"key": "value", "num": float64(42)}
		data, _ := json.MarshalIndent(original, "", "  ")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("write test file: %v", err)
		}

		settings, err := readSettings(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if settings["key"] != "value" {
			t.Errorf("key = %v, want value", settings["key"])
		}
		if settings["num"] != float64(42) {
			t.Errorf("num = %v, want 42", settings["num"])
		}
	})
}

func TestWriteSettings(t *testing.T) {
	t.Parallel()

	t.Run("round trip write then read", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "subdir", "settings.json")

		original := map[string]any{
			"statusLine": map[string]any{
				"type":    "command",
				"command": "$HOME/go/bin/statusline",
				"padding": float64(0),
			},
			"otherKey": "preserved",
		}

		if err := writeSettings(path, original); err != nil {
			t.Fatalf("writeSettings: %v", err)
		}

		read, err := readSettings(path)
		if err != nil {
			t.Fatalf("readSettings: %v", err)
		}

		sl := read["statusLine"].(map[string]any)
		if sl["type"] != "command" {
			t.Errorf("type = %v, want command", sl["type"])
		}
		if sl["command"] != "$HOME/go/bin/statusline" {
			t.Errorf("command = %v, want $HOME/go/bin/statusline", sl["command"])
		}
		if read["otherKey"] != "preserved" {
			t.Errorf("otherKey = %v, want preserved", read["otherKey"])
		}
	})
}
