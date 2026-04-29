package widgets

import (
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/lipgloss"
)

type syntaxRenderer struct {
	lexer      chroma.Lexer
	query      string
	queryLower string
}

func newSyntaxRenderer(filename, query string) syntaxRenderer {
	lx := lexers.Match(filename)
	if lx != nil {
		lx = chroma.Coalesce(lx)
	}

	return syntaxRenderer{
		lexer:      lx,
		query:      query,
		queryLower: strings.ToLower(query),
	}
}

// renderSyntaxLine applies syntax and search highlighting to a single line of
// code. filename drives language detection; query is the active search string.
// content must already have tabs expanded and be truncated to display width.
func renderSyntaxLine(filename, content, query string, bg lipgloss.Color) string {
	renderer := newSyntaxRenderer(filename, query)

	return renderer.renderLine(content, bg)
}

func (r syntaxRenderer) renderLine(content string, bg lipgloss.Color) string {
	if r.lexer == nil {
		return highlight(content, r.query, bg)
	}

	iter, err := r.lexer.Tokenise(nil, strings.TrimRight(content, "\n"))
	if err != nil {
		return highlight(content, r.query, bg)
	}

	tokens := iter.Tokens()
	matchStart, matchEnd := searchRangeLower(content, r.queryLower)

	var sb strings.Builder
	pos := 0
	for _, tok := range tokens {
		appendTokenSpan(&sb, tok.Value, pos, syntaxStyle(tok.Type, bg), matchStart, matchEnd)
		pos += len(tok.Value)
	}

	return sb.String()
}

func searchRangeLower(s, queryLower string) (int, int) {
	if queryLower == "" {
		return -1, -1
	}

	idx := strings.Index(strings.ToLower(s), queryLower)
	if idx < 0 {
		return -1, -1
	}

	return idx, idx + len(queryLower)
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
	matchLen := min(matchEnd-pos, len(tv))
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
func syntaxStyle(tt chroma.TokenType, bg lipgloss.Color) lipgloss.Style {
	base := lineBgStyle(bg)

	switch {
	case tt.InCategory(chroma.Comment):
		return base.Foreground(lipgloss.Color("244"))
	case tt == chroma.KeywordType:
		return base.Foreground(lipgloss.Color("117"))
	case tt.InCategory(chroma.Keyword):
		return base.Foreground(lipgloss.Color("204"))
	case tt.InSubCategory(chroma.LiteralString):
		return base.Foreground(lipgloss.Color("221"))
	case tt.InSubCategory(chroma.LiteralNumber):
		return base.Foreground(lipgloss.Color("141"))
	case tt.InSubCategory(chroma.NameFunction):
		return base.Foreground(lipgloss.Color("148"))
	case tt.InSubCategory(chroma.NameBuiltin):
		return base.Foreground(lipgloss.Color("80"))
	case tt == chroma.NameDecorator:
		return base.Foreground(lipgloss.Color("214"))
	default:
		return base
	}
}
