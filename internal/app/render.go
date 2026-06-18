package app

import (
	"io"

	"github.com/charmbracelet/lipgloss"
)

func SetupRenderer(w io.Writer) {
	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(w))
	lipgloss.SetHasDarkBackground(true)
}
