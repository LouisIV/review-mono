package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func GenerateDescription(apiKey, diff, prompt string) (string, error) {
	if strings.TrimSpace(apiKey) == "" {
		return fallbackDescription(diff), nil
	}
	userPrompt := "Write a concise merge request description for this diff. Include a summary and test notes if evident."
	if prompt != "" {
		userPrompt += "\nAdditional guidance: " + prompt
	}
	userPrompt += "\n\nDiff:\n" + truncate(diff, 120000)
	payload := map[string]any{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 1200,
		"messages": []map[string]string{{
			"role": "user", "content": userPrompt,
		}},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("content-type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic api returned %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", err
	}
	var out strings.Builder
	for _, part := range parsed.Content {
		if part.Text != "" {
			out.WriteString(part.Text)
		}
	}
	return strings.TrimSpace(out.String()) + "\n", nil
}

func fallbackDescription(diff string) string {
	files := 0
	adds := 0
	dels := 0
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			files++
		}
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			adds++
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			dels++
		}
	}
	return fmt.Sprintf("## Summary\n\nUpdates %d file(s), with %d additions and %d deletions.\n\n## Testing\n\nNot specified.\n", files, adds, dels)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n\n[diff truncated]\n"
}
