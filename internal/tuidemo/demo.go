package tuidemo

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"review/internal/tui/widgets"
)

type Props = widgets.Props

func ForceColor() {
	lipgloss.SetColorProfile(termenv.TrueColor)
}

func Names() []string {
	return widgets.Names()
}

func DefaultProps() Props {
	return widgets.DefaultProps()
}

func MergeProps(raw string, props Props) (Props, error) {
	if raw == "" {
		return widgets.Normalize(props), nil
	}

	if err := json.Unmarshal([]byte(raw), &props); err != nil {
		return props, err
	}

	return widgets.Normalize(props), nil
}

func ApplyEvent(props Props, event string) Props {
	return widgets.ApplyEvent(props, event)
}

func Render(name string, props Props, tree bool) (string, error) {
	if tree {
		return widgets.RenderTree(name), nil
	}

	model := widgets.NewInteractiveModel(name, props)
	if model.Unknown() {
		return "", fmt.Errorf("unknown widget %q", name)
	}

	return model.View(), nil
}

func RunInteractive(name string, props Props) error {
	model := widgets.NewModel(name, props)
	if model.Unknown() {
		return fmt.Errorf("unknown widget %q", name)
	}

	_, err := tea.NewProgram(model).Run()

	return err
}
