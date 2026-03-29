package main

// ─── Types ─────────────────────────────────────────────────────────────────

// InputData is the JSON payload piped to stdin by Claude Code.
type InputData struct {
	Version        string             `json:"version"`
	SessionID      string             `json:"session_id"`
	TranscriptPath string             `json:"transcript_path"`
	Model          *ModelInfo         `json:"model"`
	ContextWindow  *ContextWindowData `json:"context_window"`
}

// ModelInfo holds model display name.
type ModelInfo struct {
	DisplayName string `json:"display_name"`
}

// ContextWindowData holds context usage statistics.
type ContextWindowData struct {
	TotalInputTokens              *int          `json:"total_input_tokens"`
	TotalOutputTokens             *int          `json:"total_output_tokens"`
	TotalCacheCreationInputTokens *int          `json:"total_cache_creation_input_tokens"`
	TotalCacheReadInputTokens     *int          `json:"total_cache_read_input_tokens"`
	ContextWindowSize             *int          `json:"context_window_size"`
	RemainingPercentage           *float64      `json:"remaining_percentage"`
	UsedPercentage                *float64      `json:"used_percentage"`
	CurrentUsage                  *CurrentUsage `json:"current_usage"`
}

// CurrentUsage holds per-turn token counts.
type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// ProjectInfo holds detected project badge and version for status bar rendering.
// Intentionally separate from hooks/session.go's ProjectInfo — different fields
// (Badge+Version here vs Type+Emoji there), no JSON tags, no shared behavior.
type ProjectInfo struct {
	Badge   string
	Version string
}

// ─── Health Types ──────────────────────────────────────────────────────────

// ContextHealth represents context window health level.
type ContextHealth int

const (
	HealthGreen  ContextHealth = iota // >25% remaining
	HealthYellow                      // 10-25% remaining
	HealthRed                         // <10% remaining
	HealthNone                        // no data available
)
