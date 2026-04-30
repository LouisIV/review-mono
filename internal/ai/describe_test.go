package ai_test

import (
	"strings"
	"testing"

	"review/internal/ai"
)

func TestFallbackDescriptionIncludesChangedFileContext(t *testing.T) {
	t.Parallel()

	diff := strings.Join([]string{
		"diff --git a/internal/tui/app.go b/internal/tui/app.go",
		"index 1111111..2222222 100644",
		"--- a/internal/tui/app.go",
		"+++ b/internal/tui/app.go",
		"@@ -10,6 +10,7 @@ func renderApp() {",
		"+\treturn",
		"diff --git a/internal/ai/describe.go b/internal/ai/describe.go",
		"index 3333333..4444444 100644",
		"--- a/internal/ai/describe.go",
		"+++ b/internal/ai/describe.go",
		"@@ -20,6 +20,7 @@ func fallbackDescription(diff string) string {",
		"-\treturn old",
		"+\treturn new",
	}, "\n")

	out, err := ai.GenerateDescription("", "fallback", diff, "")
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{
		"Updates 2 file(s), with 2 additions and 1 deletions.",
		"- internal/tui/app.go: func renderApp() {",
		"- internal/ai/describe.go: func fallbackDescription(diff string) string {",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("fallback description missing %q:\n%s", want, out)
		}
	}
}
