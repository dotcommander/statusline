package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// ─── Input Reading ─────────────────────────────────────────────────────────

// readInput reads JSON from stdin. Returns zero value if stdin is a terminal
// or on any error. If STATUSLINE_DEBUG_INPUT is set, the raw stdin bytes are
// also written to that path for diagnostics. Any tee error is swallowed —
// the statusline must never fail because of an instrumentation problem.
func readInput() InputData {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return InputData{}
	}
	if fi.Mode()&os.ModeCharDevice != 0 {
		// stdin is a terminal, nothing piped
		return InputData{}
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return InputData{}
	}

	if path := os.Getenv("STATUSLINE_DEBUG_INPUT"); path != "" {
		_ = os.WriteFile(path, data, 0o644)
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return InputData{}
	}

	var input InputData
	if err := json.Unmarshal(trimmed, &input); err != nil {
		return InputData{}
	}
	return input
}
