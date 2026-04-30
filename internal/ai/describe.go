package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const (
	claudeCLI                    = "claude-cli"
	codexCLI                     = "codex-cli"
	maxDescriptionDiffBytes      = 60000
	maxDescriptionGuidanceBytes  = 4000
	maxGeneratedDescriptionBytes = 4000
	maxFallbackFiles             = 20
	maxHunksPerFallbackFile      = 3
	maxFallbackPathBytes         = 160
	maxFallbackHunkContextBytes  = 120
	truncatedDescriptionMarker   = "\n\n[description truncated]\n"
)

func GenerateDescription(apiKey, provider, diff, prompt string) (string, error) {
	userPrompt := descriptionPrompt(diff, prompt)

	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "auto":
		if strings.TrimSpace(apiKey) != "" {
			return generatedDescription(generateWithAnthropicAPI(apiKey, userPrompt))
		}

		if out, err := generateWithCLI(claudeCLI, userPrompt); err == nil {
			return generatedDescription(out, nil)
		}

		if out, err := generateWithCLI(codexCLI, userPrompt); err == nil {
			return generatedDescription(out, nil)
		}

		return generatedDescription(fallbackDescription(diff), nil)
	case "anthropic", "anthropic-api", "claude-api":
		if strings.TrimSpace(apiKey) == "" {
			return "", fmt.Errorf("anthropic api key is required for provider %q", provider)
		}

		return generatedDescription(generateWithAnthropicAPI(apiKey, userPrompt))
	case "claude", claudeCLI:
		return generatedDescription(generateWithCLI(claudeCLI, userPrompt))
	case "codex", codexCLI:
		return generatedDescription(generateWithCLI(codexCLI, userPrompt))
	case "fallback":
		return generatedDescription(fallbackDescription(diff), nil)
	default:
		return "", fmt.Errorf("unknown ai provider %q", provider)
	}
}

func descriptionPrompt(diff, prompt string) string {
	userPrompt := "Write a concise merge request description for this diff. " +
		"Include a summary and test notes if evident."
	if prompt != "" {
		userPrompt += "\nAdditional guidance: " + truncate(prompt, maxDescriptionGuidanceBytes)
	}

	return userPrompt + "\n\nDiff:\n" + truncate(diff, maxDescriptionDiffBytes)
}

func generateWithAnthropicAPI(apiKey, userPrompt string) (string, error) {
	payload := map[string]any{
		"model":      "claude-3-5-sonnet-20241022",
		"max_tokens": 1200,
		"messages": []map[string]string{{
			"role": "user", "content": userPrompt,
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", apiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
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
			_, _ = out.WriteString(part.Text)
		}
	}

	return strings.TrimSpace(out.String()) + "\n", nil
}

func generateWithCLI(provider, prompt string) (string, error) {
	var (
		name string
		args []string
	)

	switch provider {
	case claudeCLI:
		name = "claude"
		args = []string{
			"--print",
			"--input-format",
			"text",
			"--output-format",
			"stream-json",
			"--no-session-persistence",
		}
	case codexCLI:
		name = "codex"
		args = []string{"exec", "--sandbox", "read-only", "--ask-for-approval", "never", "--ephemeral", "--json", "-"}
	default:
		return "", fmt.Errorf("unknown cli provider %q", provider)
	}

	if _, err := exec.LookPath(name); err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	//nolint:gosec // CLI provider names are selected from the fixed switch above.
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer

	cmd.Stdout = &stdout

	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("%s failed: %w: %s", name, err, msg)
		}

		return "", fmt.Errorf("%s failed: %w", name, err)
	}

	out := strings.TrimSpace(streamedCLIOutput(provider, stdout.Bytes()))
	if out == "" {
		return "", fmt.Errorf("%s returned empty output", name)
	}

	return out + "\n", nil
}

func streamedCLIOutput(provider string, data []byte) string {
	var (
		result    string
		fragments []string
	)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if provider == claudeCLI {
			if text := claudeStreamText([]byte(line)); text != "" {
				result = text
			}

			continue
		}

		if provider == codexCLI {
			if text := codexStreamText([]byte(line)); text != "" {
				fragments = append(fragments, text)
			}
		}
	}

	if provider == codexCLI && len(fragments) > 0 {
		return strings.Join(fragments, "")
	}

	if result != "" {
		return result
	}

	return string(data)
}

func claudeStreamText(line []byte) string {
	var event struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
		Result  string `json:"result"`
	}
	err := json.Unmarshal(line, &event)
	if err != nil {
		return ""
	}

	if event.Type == "result" || event.Subtype == "success" {
		return event.Result
	}

	return ""
}

func codexStreamText(line []byte) string {
	var event struct {
		Msg  string `json:"msg"`
		Type string `json:"type"`
		Item struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"item"`
		Delta string `json:"delta"`
	}
	err := json.Unmarshal(line, &event)
	if err != nil {
		return ""
	}

	if event.Type == "agent_message_delta" || event.Msg == "agent_message_delta" {
		return event.Delta
	}

	if event.Item.Type == "message" || event.Item.Type == "assistant_message" {
		return event.Item.Text
	}

	return ""
}

func fallbackDescription(diff string) string {
	files := 0
	adds := 0
	dels := 0
	changed := changedFileSummaries(diff)

	for line := range strings.SplitSeq(diff, "\n") {
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

	var summary strings.Builder
	_, _ = fmt.Fprintf(
		&summary,
		"Updates %d file(s), with %d additions and %d deletions.",
		files,
		adds,
		dels,
	)
	if len(changed) > 0 {
		_, _ = fmt.Fprintf(&summary, "\n\nChanged files:\n")
		for _, file := range changed {
			_, _ = fmt.Fprintf(&summary, "- %s", file.Path)
			if len(file.Hunks) > 0 {
				_, _ = fmt.Fprintf(&summary, ": %s", strings.Join(file.Hunks, "; "))
			}
			_, _ = summary.WriteString("\n")
		}
		if files > len(changed) {
			_, _ = fmt.Fprintf(&summary, "- ...and %d more file(s)\n", files-len(changed))
		}
	}

	return fmt.Sprintf(
		"## Summary\n\n%s\n\n## Testing\n\nNot specified.\n",
		summary.String(),
	)
}

type changedFileSummary struct {
	Path  string
	Hunks []string
}

func changedFileSummaries(diff string) []changedFileSummary {
	out := []changedFileSummary{}
	acceptingFile := false
	for line := range strings.SplitSeq(diff, "\n") {
		if strings.HasPrefix(line, "diff --git ") {
			acceptingFile = len(out) < maxFallbackFiles
			if len(out) >= maxFallbackFiles {
				continue
			}

			parts := strings.Fields(line)
			path := ""
			if len(parts) >= 4 {
				path = truncateInline(strings.TrimPrefix(parts[3], "b/"), maxFallbackPathBytes)
			}
			out = append(out, changedFileSummary{Path: path})

			continue
		}

		if !acceptingFile || len(out) == 0 || !strings.HasPrefix(line, "@@") {
			continue
		}

		file := &out[len(out)-1]
		if len(file.Hunks) >= maxHunksPerFallbackFile {
			continue
		}

		if hunk := hunkContext(line); hunk != "" {
			file.Hunks = append(file.Hunks, hunk)
		}
	}

	return out
}

func hunkContext(line string) string {
	end := strings.LastIndex(line, "@@")
	if end <= 1 || end+2 >= len(line) {
		return ""
	}

	return truncateInline(strings.TrimSpace(line[end+2:]), maxFallbackHunkContextBytes)
}

func generatedDescription(body string, err error) (string, error) {
	if err != nil {
		return "", err
	}

	return truncateDescription(body), nil
}

func truncateDescription(body string) string {
	if len(body) <= maxGeneratedDescriptionBytes {
		return body
	}

	limit := maxGeneratedDescriptionBytes - len(truncatedDescriptionMarker)
	if limit <= 0 {
		return truncatedDescriptionMarker
	}

	return strings.TrimRight(body[:limit], "\n") + truncatedDescriptionMarker
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}

	return s[:limit] + "\n\n[diff truncated]\n"
}

func truncateInline(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	if limit <= 3 {
		return s[:limit]
	}

	return s[:limit-3] + "..."
}
