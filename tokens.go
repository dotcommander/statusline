package main

import (
	"fmt"
	"strconv"
)

// ─── Token Math and Health ─────────────────────────────────────────────────

// getEffectiveTokens calculates used tokens and remaining percentage.
// Exact port of the TS implementation with all 4 priority paths.
func getEffectiveTokens(ctx *ContextWindowData, maxOutputTokens int) (used, remainingPct int) {
	if ctx == nil || ctx.ContextWindowSize == nil || *ctx.ContextWindowSize == 0 {
		return 0, 100
	}

	total := *ctx.ContextWindowSize
	reserved := maxOutputTokens
	available := max(total-reserved, 0)

	calcPct := func(u int) int {
		if available > 0 {
			remaining := max(available-u, 0)
			return int(float64(remaining)/float64(available)*100 + 0.5)
		}
		return int(float64(total-u)/float64(total)*100 + 0.5)
	}

	// Priority 1: used_percentage
	if ctx.UsedPercentage != nil && *ctx.UsedPercentage > 0 {
		u := int(*ctx.UsedPercentage/100*float64(total) + 0.5)
		return u, calcPct(u)
	}

	// Priority 2: remaining_percentage
	if ctx.RemainingPercentage != nil && *ctx.RemainingPercentage > 0 && *ctx.RemainingPercentage < 100 {
		u := int((100-*ctx.RemainingPercentage)/100*float64(total) + 0.5)
		return u, calcPct(u)
	}

	// Priority 3: current_usage
	if ctx.CurrentUsage != nil {
		cu := ctx.CurrentUsage
		cuTotal := cu.InputTokens + cu.CacheCreationInputTokens + cu.CacheReadInputTokens
		if cuTotal > 0 {
			u := min(cuTotal, total)
			return u, calcPct(u)
		}
	}

	// Priority 4: cumulative totals
	cumulative := 0
	if ctx.TotalInputTokens != nil {
		cumulative += *ctx.TotalInputTokens
	}
	if ctx.TotalOutputTokens != nil {
		cumulative += *ctx.TotalOutputTokens
	}
	if cumulative == 0 {
		return 0, 100
	}
	u := min(cumulative, total)
	return u, calcPct(u)
}

// contextHealth determines health level from remaining percentage.
func contextHealth(remainingPct int) ContextHealth {
	switch {
	case remainingPct > contextWarningPct:
		return HealthGreen
	case remainingPct > contextCriticalPct:
		return HealthYellow
	default:
		return HealthRed
	}
}

// dotColor returns the ANSI foreground color for the mode dot based on context health.
// Uses Catppuccin Mocha colors: green (#a6e3a1), yellow (#f9e2af), red (#f38ba8), orange/peach (#fab387).
func dotColor(h ContextHealth) string {
	switch h {
	case HealthGreen:
		return pal.green
	case HealthYellow:
		return pal.yellow
	case HealthRed:
		return pal.red
	default: // HealthNone
		return pal.orange
	}
}

// formatTokens formats a token count as a human-readable string.
func formatTokens(tokens int) string {
	switch {
	case tokens >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(tokens)/1_000_000)
	case tokens >= 10_000:
		return fmt.Sprintf("%dk", (tokens+500)/1_000)
	case tokens >= 1_000:
		return fmt.Sprintf("%.1fk", float64(tokens)/1_000)
	default:
		return strconv.Itoa(tokens)
	}
}
