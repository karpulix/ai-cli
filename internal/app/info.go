package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karpulix/ai-cli/internal/config"
	"github.com/karpulix/ai-cli/internal/history"
	"github.com/karpulix/ai-cli/internal/prompt"
	"github.com/karpulix/ai-cli/internal/sysinfo"
	"github.com/karpulix/ai-cli/internal/version"
)

type infoState struct {
	open bool
}

func (s infoState) Update(msg tea.KeyMsg) (infoState, tea.Cmd) {
	if !s.open {
		return s, nil
	}
	switch {
	case msg.String() == "esc", isInfoKey(msg):
		s.open = false
		return s, textinput.Blink
	}
	return s, nil
}

func (s infoState) View(th theme, cfg *config.Config, store *history.Store, profileName string, width int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(th.labelActive.Render("● Info"))
	b.WriteString("\n\n")

	writeSection := func(title, body string) {
		b.WriteString(th.label.Render(title))
		b.WriteString("\n")
		b.WriteString(th.meta.Render(body))
		b.WriteString("\n\n")
	}

	writeSection("Version", version.Display())

	if path, err := config.Path(); err == nil {
		writeSection("Config", path)
	} else {
		writeSection("Config", err.Error())
	}

	histPath := store.Path()
	if histPath == "" {
		if p, err := history.DefaultPath(); err == nil {
			histPath = p
		}
	}
	writeSection("History", fmt.Sprintf("%s (%d entries)", histPath, store.Count()))

	if profileName != "" {
		line := profileName
		if p, ok := cfg.Profiles[cfg.ActiveProfile]; ok {
			if p.Model != "" {
				line += " · " + p.Model
			}
			if p.BaseURL != "" {
				line += " @ " + p.BaseURL
			}
		}
		writeSection("Active profile", line)
	}

	info := sysinfo.Detect()
	b.WriteString(th.label.Render("System"))
	b.WriteString("\n")
	for _, line := range strings.Split(prompt.FormatSystemInfo(info), "\n") {
		b.WriteString(th.meta.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if cwd, err := os.Getwd(); err == nil {
		writeSection("Working directory", cwd)
	}

	if term := os.Getenv("TERM"); term != "" {
		writeSection("Terminal", term)
	}

	b.WriteString(th.metaSeparator(width))
	b.WriteString("\n\n")

	k := th.helpKey.Render
	d := th.helpText.Render
	b.WriteString(d("") + k(infoKeyLabel()) + d(" close  ") + k("esc") + d(" back"))

	return b.String()
}
