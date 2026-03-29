package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// ─── Input Reading ─────────────────────────────────────────────────────────

// readInput reads JSON from stdin. Returns zero value if stdin is a terminal
// or on any error.
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
