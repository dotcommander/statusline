package configtui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dotcommander/statusline/internal/config"
)

// Run builds the model and runs the bubbletea TUI.
func Run() error {
	path := config.DefaultPath()
	if path == "" {
		fmt.Fprintln(os.Stderr, "cannot determine config path")
		return fmt.Errorf("cannot determine config path")
	}

	cfg := config.Load(path)
	m := newModel(cfg, path)

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
