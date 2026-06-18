package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorAccent   = lipgloss.Color("#8BE9FD")
	colorPink     = lipgloss.Color("#FF79C6")
	colorPurple   = lipgloss.Color("#BD93F9")
	colorShine    = lipgloss.Color("#F8F8F2")
	colorGreen    = lipgloss.Color("#50FA7B")
	colorYellow   = lipgloss.Color("#F1FA8C")
	colorMuted    = lipgloss.Color("#6272A4")
	colorDim      = lipgloss.Color("#44475A")
	colorError    = lipgloss.Color("#FF5555")
	colorSelectBG = lipgloss.Color("#44475A")
	colorWhite    = lipgloss.Color("#F8F8F2")
)

type theme struct {
	box         lipgloss.Style
	label       lipgloss.Style
	labelActive lipgloss.Style
	spinner     lipgloss.Style
	loading     lipgloss.Style
	error       lipgloss.Style
	empty       lipgloss.Style
	histPrompt  lipgloss.Style
	histArrow   lipgloss.Style
	histCmd     lipgloss.Style
	histSelect  lipgloss.Style
	helpKey     lipgloss.Style
	helpText    lipgloss.Style
	meta        lipgloss.Style
}

func newTheme() theme {
	return theme{
		box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorPurple).
			Padding(1, 2),

		label: lipgloss.NewStyle().
			Foreground(colorMuted),

		labelActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent),

		spinner: lipgloss.NewStyle().
			Foreground(colorPink),

		loading: lipgloss.NewStyle().
			Foreground(colorYellow).
			Italic(true),

		error: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorError),

		empty: lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true),

		histPrompt: lipgloss.NewStyle().
			Foreground(colorYellow),

		histArrow: lipgloss.NewStyle().
			Foreground(colorPurple),

		histCmd: lipgloss.NewStyle().
			Foreground(colorGreen),

		histSelect: lipgloss.NewStyle().
			Background(colorSelectBG).
			Foreground(colorWhite),

		helpKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPink),

		helpText: lipgloss.NewStyle().
			Foreground(colorMuted),

		meta: lipgloss.NewStyle().
			Foreground(colorDim).
			Italic(true),
	}
}

func (t theme) shimmerTitle(text string, frame int) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return ""
	}

	cycle := n + 6
	pos := frame % cycle

	var b strings.Builder
	for i, r := range runes {
		dist := abs(i - pos)
		color := colorPurple
		switch dist {
		case 0:
			color = colorShine
		case 1:
			color = lipgloss.Color("#D6ACFF")
		}
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(color).Render(string(r)))
	}
	return b.String()
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (t theme) sectionLabel(name string, active bool) string {
	if active {
		return t.labelActive.Render("● " + name)
	}
	return t.label.Render("  " + name)
}

func (t theme) metaSeparator(width int) string {
	if width < 12 {
		width = 40
	}
	return t.meta.Render(strings.Repeat("─", width))
}

func (t theme) helpLine() string {
	k := t.helpKey.Render
	d := t.helpText.Render
	return d("") +
		k("enter") + d(" submit  ") +
		k("tab") + d(" history  ") +
		k("↑↓") + d(" navigate  ") +
		k("ctrl+p") + d(" profiles  ") +
		k(infoKeyLabel()) + d(" info  ") +
		k("backspace") + d(" delete  ") +
		k("esc") + d(" cancel")
}
