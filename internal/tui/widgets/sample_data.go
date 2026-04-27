package widgets

type fileRow struct {
	Path       string
	Additions  int
	Deletions  int
	Unresolved int
	Viewed     bool
}

type diffRow struct {
	Kind    string
	Line    int
	Content string
}

type commentRow struct {
	ID       string
	File     string
	Line     int
	EndLine  int
	Body     string
	Resolved bool
}

var files = []fileRow{
	{Path: "cmd/root.go", Additions: 18, Deletions: 3, Unresolved: 1, Viewed: true},
	{Path: "internal/tui/diff_view.go", Additions: 96, Deletions: 0, Unresolved: 2, Viewed: true},
	{Path: "internal/tui/file_picker.go", Additions: 74, Deletions: 0, Unresolved: 0},
	{Path: "internal/tui/context_menu.go", Additions: 63, Deletions: 0, Unresolved: 1},
	{Path: "docs/tui-review.md", Additions: 42, Deletions: 2, Unresolved: 0, Viewed: true},
}

var diffRows = []diffRow{
	{Kind: "hunk", Content: "@@ -78,6 +78,25 @@ func run(args []string, g globals, cfg config.Config) error"},
	{Kind: "context", Line: 81, Content: `	case "watch":`},
	{Kind: "context", Line: 82, Content: `		return watchCmd(args[1:], g)`},
	{Kind: "add", Line: 83, Content: `	case "widget":`},
	{Kind: "add", Line: 84, Content: `		return widgetCmd(args[1:], g)`},
	{Kind: "context", Line: 85, Content: `	default:`},
	{Kind: "context", Line: 86, Content: `		return fmt.Errorf("unknown command %q", args[0])`},
	{Kind: "context", Line: 87, Content: `	}`},
	{Kind: "context", Line: 88, Content: ``},
	{Kind: "context", Line: 89, Content: `func widgetCmd(args []string, _ globals) error {`},
	{Kind: "context", Line: 90, Content: `	if len(args) == 0 || isHelpArg(args[0]) {`},
	{Kind: "context", Line: 91, Content: `		widgetUsage()`},
	{Kind: "context", Line: 92, Content: `		return nil`},
	{Kind: "context", Line: 93, Content: `	}`},
	{Kind: "context", Line: 94, Content: ``},
	{Kind: "add", Line: 95, Content: `	model := tuidemo.NewModel(name, props)`},
	{Kind: "add", Line: 96, Content: `	next, _ := model.Update(msg)`},
	{Kind: "add", Line: 97, Content: `	fmt.Print(next.View())`},
	{Kind: "hunk", Content: "@@ -228,7 +247,7 @@ func usage()"},
	{Kind: "remove", Line: 248, Content: `			"description, approve, request-changes, watch",`},
	{Kind: "add", Line: 249, Content: `			"description, approve, request-changes, watch, widget",`},
	{Kind: "context", Line: 250, Content: `	)`},
	{Kind: "context", Line: 251, Content: `	fmt.Println("global flags: --repo <path>, --port <port>, --json")`},
	{Kind: "context", Line: 252, Content: `}`},
}

var comments = []commentRow{
	{ID: "C-102a", File: "cmd/root.go", Line: 83,
		Body: "This is a good place for the widget playground command.", Resolved: false},
	{ID: "C-7f31", File: "internal/tui/diff_view.go", Line: 42, EndLine: 47,
		Body: "Range comments should use the visual selection bounds.", Resolved: false},
	{ID: "C-8b20", File: "docs/tui-review.md", Line: 118,
		Body: "Resolved after moving file navigation off n/N.", Resolved: true},
}
