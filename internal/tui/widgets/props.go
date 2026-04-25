package widgets

type Props struct {
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Query        string `json:"query"`
	ActiveFile   string `json:"active_file"`
	SelectedLine int    `json:"selected_line"`
	VisualStart  int    `json:"visual_start"`
	VisualEnd    int    `json:"visual_end"`
	YOffset      int    `json:"y_offset"`
	XOffset      int    `json:"x_offset"`
	ShowResolved bool   `json:"show_resolved"`
	MenuTarget   string `json:"menu_target"`
	MenuIndex    int    `json:"menu_index"`
}

func Names() []string {
	return []string{"workspace", "file-list", "diff", "context-menu", "goto-file", "search", "comments"}
}

func DefaultProps() Props {
	return Props{
		Width:        96,
		Height:       28,
		Query:        "widget",
		ActiveFile:   "cmd/root.go",
		SelectedLine: 83,
		VisualStart:  83,
		VisualEnd:    84,
		MenuTarget:   "visual-selection",
	}
}

func Normalize(props Props) Props {
	if props.Width <= 0 {
		props.Width = 96
	}

	if props.Height <= 0 {
		props.Height = 28
	}

	if props.ActiveFile == "" {
		props.ActiveFile = "cmd/root.go"
	}

	if props.SelectedLine <= 0 {
		props.SelectedLine = 83
	}

	if props.MenuTarget == "" {
		props.MenuTarget = "diff-line"
	}

	if props.VisualStart > 0 && props.VisualEnd == 0 {
		props.VisualEnd = props.VisualStart
	}

	if props.VisualEnd > 0 && props.VisualStart == 0 {
		props.VisualStart = props.VisualEnd
	}

	if props.VisualStart > props.VisualEnd {
		props.VisualStart, props.VisualEnd = props.VisualEnd, props.VisualStart
	}

	return props
}
