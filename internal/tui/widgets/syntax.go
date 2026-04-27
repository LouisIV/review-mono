package widgets

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
)

// renderSyntaxLine applies syntax and search highlighting to a single line of
// code. filename drives language detection; query is the active search string.
// content must already have tabs expanded and be truncated to display width.
func renderSyntaxLine(filename, content, query string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		return highlight(content, query)
	}

	lexer = chroma.Coalesce(lexer)
	iter, err := lexer.Tokenise(nil, strings.TrimRight(content, "\n"))
	if err != nil {
		return highlight(content, query)
	}

	tokens := iter.Tokens()
	matchStart, matchEnd := searchRange(content, query)

	var sb strings.Builder
	pos := 0
	for _, tok := range tokens {
		appendTokenSpan(&sb, tok.Value, pos, syntaxStyle(tok.Type), matchStart, matchEnd)
		pos += len(tok.Value)
	}

	return sb.String()
}

// searchRange returns the byte range [start, end) of the first case-insensitive
// occurrence of query in s. Returns -1, -1 when not found or query is empty.
func searchRange(s, query string) (int, int) {
	if query == "" {
		return -1, -1
	}

	idx := strings.Index(strings.ToLower(s), strings.ToLower(query))
	if idx < 0 {
		return -1, -1
	}

	return idx, idx + len(query)
}

// appendTokenSpan writes tv to sb with ts applied, substituting searchStyle for
// any bytes that fall within [matchStart, matchEnd).
func appendTokenSpan(sb *strings.Builder, tv string, pos int, ts lipgloss.Style, matchStart, matchEnd int) {
	tEnd := pos + len(tv)

	if matchStart < 0 || tEnd <= matchStart || pos >= matchEnd {
		if tv != "" {
			sb.WriteString(ts.Render(tv))
		}

		return
	}

	// Part before the match.
	if pos < matchStart {
		sb.WriteString(ts.Render(tv[:matchStart-pos]))
		tv = tv[matchStart-pos:]
		pos = matchStart
	}

	// Match part (may be a prefix of tv if match extends past this token).
	matchLen := matchEnd - pos
	if matchLen > len(tv) {
		matchLen = len(tv)
	}
	if matchLen > 0 {
		sb.WriteString(searchStyle.Render(tv[:matchLen]))
		tv = tv[matchLen:]
	}

	// Part after the match.
	if len(tv) > 0 {
		sb.WriteString(ts.Render(tv))
	}
}

// syntaxStyle maps a Chroma token type to the corresponding lipgloss style.
// More-specific sub-categories are checked before their parent categories so
// that, e.g., type-keywords get cyan while other keywords get pink.
func syntaxStyle(tt chroma.TokenType) lipgloss.Style {
	switch {
	case tt.InCategory(chroma.Comment):
		// All Comment subtypes → muted gray.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	case tt == chroma.KeywordType:
		// Type keywords (int, string, bool, …) → cyan.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	case tt.InCategory(chroma.Keyword):
		// Other keywords (if, for, func, …) → pink.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("204"))
	case tt.InSubCategory(chroma.LiteralString):
		// String literals → yellow.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("221"))
	case tt.InSubCategory(chroma.LiteralNumber):
		// Numeric literals → purple.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("141"))
	case tt.InSubCategory(chroma.NameFunction):
		// Function names → light green.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("148"))
	case tt.InSubCategory(chroma.NameBuiltin):
		// Built-in names → cyan-green.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("80"))
	case tt == chroma.NameDecorator:
		// Decorators / attributes → orange.
		return lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	default:
		return lipgloss.NewStyle()
	}
}
