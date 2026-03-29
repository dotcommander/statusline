package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
)

// ─── Prompts ───────────────────────────────────────────────────────────────

const transcriptTailBytes = 262144 // 256KB — tool_result entries can push real prompts far back

var (
	promptCreateRe = regexp.MustCompile(`create|write|add|implement`)
	promptFixRe    = regexp.MustCompile(`fix|debug|error`)
)

// fetchPrompts reads the transcript file and returns up to 3 recent user prompts.
func fetchPrompts(transcriptPath string) []string {
	if transcriptPath == "" {
		return nil
	}

	f, err := os.Open(transcriptPath)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil
	}

	size := fi.Size()
	if size > transcriptTailBytes {
		if _, err := f.Seek(size-transcriptTailBytes, io.SeekStart); err != nil {
			return nil
		}
	}

	tail, err := io.ReadAll(f)
	if err != nil {
		return nil
	}

	lines := bytes.Split(tail, []byte("\n"))
	var prompts []string

	for i := len(lines) - 1; i >= 0 && len(prompts) < 3; i-- {
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		text := extractUserPromptText(line)
		if text != "" {
			prompts = append([]string{text}, prompts...)
		}
	}

	return prompts
}

// extractUserPromptText parses a JSONL line and extracts the user prompt text,
// applying all filters from the TS implementation.
func extractUserPromptText(line []byte) string {
	// Fast pre-check: must contain "type":"user"
	if !bytes.Contains(line, []byte(`"type":"user"`)) {
		return ""
	}

	var entry struct {
		Type    string          `json:"type"`
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(line, &entry); err != nil {
		return ""
	}
	if entry.Type != "user" {
		return ""
	}

	// Parse message content
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(entry.Message, &msg); err != nil {
		return ""
	}

	var text string

	// Try string first
	if err := json.Unmarshal(msg.Content, &text); err != nil {
		// Try array of content blocks
		var blocks []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err2 := json.Unmarshal(msg.Content, &blocks); err2 != nil {
			return ""
		}
		for _, b := range blocks {
			if b.Type == "text" {
				text = b.Text
				break
			}
		}
	}

	if text == "" {
		return ""
	}

	// Extract slash commands from transcript XML wrapper.
	// Claude Code records "/commit --push" as:
	//   <command-name>/commit</command-name>\n<command-message>commit --push</command-message>\n...
	// Pull the command-message content and use it as the prompt text.
	slashExtracted := false
	if strings.Contains(text, "<command-message>") {
		if start := strings.Index(text, "<command-message>"); start >= 0 {
			start += len("<command-message>")
			if end := strings.Index(text[start:], "</command-message>"); end >= 0 {
				cmdText := strings.TrimSpace(text[start : start+end])
				if cmdText != "" {
					if !strings.HasPrefix(cmdText, "/") {
						cmdText = "/" + cmdText
					}
					text = cmdText
					slashExtracted = true
					// Fall through to remaining filters.
				}
			}
		}
	}

	// Apply filters
	// Filter messages that look like XML/system tags (not real user prompts)
	if strings.HasPrefix(text, "<") {
		return ""
	}
	if strings.HasPrefix(text, "You are a") {
		return ""
	}
	if strings.HasPrefix(text, `{"`) {
		return ""
	}
	if len(text) > 500 {
		return ""
	}
	if strings.Contains(text, "<system-reminder>") {
		return ""
	}
	if !slashExtracted && (strings.Contains(text, "<command-name>") || strings.Contains(text, "<command-message>")) {
		return ""
	}
	if strings.Contains(text, "<task-notification>") {
		return ""
	}

	return strings.TrimSpace(strings.ReplaceAll(text, "\n", " "))
}

// truncateWords truncates text to at most max words, appending "..." if needed.
func truncateWords(text string, max int) string {
	words := strings.Fields(text)
	if len(words) > max {
		return strings.Join(words[:max], " ") + "..."
	}
	return strings.Join(words, " ")
}

// getPromptIcon returns an icon character based on the prompt content.
func getPromptIcon(prompt string) string {
	if strings.HasPrefix(prompt, "/") {
		return "»"
	}
	if strings.Contains(prompt, "?") {
		return "?"
	}
	lower := strings.ToLower(prompt)
	if promptCreateRe.MatchString(lower) {
		return "+"
	}
	if promptFixRe.MatchString(lower) {
		return "×"
	}
	return "›"
}
