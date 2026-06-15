package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/elmerescandon/greg-ui/internal/ui"
)

func main() {
	vault := os.Getenv("GREG_VAULT")
	if vault == "" {
		home, _ := os.UserHomeDir()
		vault = home
	}

	m := ui.NewModel(vault)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
