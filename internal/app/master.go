package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/karpulix/ai-cli/internal/config"
	"github.com/karpulix/ai-cli/internal/llm"
)

type masterPhase int

const (
	masterList masterPhase = iota
	masterForm
)

const masterFieldCount = 4

type profileRow struct {
	name    string
	model   string
	baseURL string
	active  bool
}

type masterState struct {
	open      bool
	phase     masterPhase
	cursor    int
	names     []string
	rows      []profileRow
	active    string
	formField int
	name      textinput.Model
	apiKey    textinput.Model
	model     textinput.Model
	baseURL   textinput.Model
	err       string
}

func newMasterState(width int) masterState {
	mk := func(placeholder string) textinput.Model {
		ti := textinput.New()
		ti.Placeholder = placeholder
		ti.CharLimit = 512
		ti.Width = max(20, width-8)
		return ti
	}

	name := mk("profile name")
	name.Focus()

	return masterState{
		phase:     masterList,
		name:      name,
		apiKey:    mk("sk-... (optional for ollama)"),
		model:     mk("gpt-4o-mini"),
		baseURL:   mk("http://localhost:11434/v1"),
		formField: 0,
	}
}

func (m *masterState) refresh(cfg *config.Config) {
	m.names = cfg.ProfileNames()
	m.active = cfg.ActiveProfile
	m.rows = m.rows[:0]
	for _, name := range m.names {
		p := cfg.Profiles[name]
		m.rows = append(m.rows, profileRow{
			name:    name,
			model:   p.Model,
			baseURL: p.BaseURL,
			active:  name == cfg.ActiveProfile,
		})
	}
	if m.cursor >= len(m.names) {
		m.cursor = max(0, len(m.names)-1)
	}
}

func (m *masterState) formInputs() []*textinput.Model {
	return []*textinput.Model{&m.name, &m.apiKey, &m.model, &m.baseURL}
}

func (m *masterState) focusForm(field int) {
	m.formField = field
	for i, ti := range m.formInputs() {
		if i == field {
			ti.Focus()
		} else {
			ti.Blur()
		}
	}
}

func (m *masterState) openForm(width int) {
	m.phase = masterForm
	m.err = ""
	m.name.SetValue("")
	m.apiKey.SetValue("")
	m.model.SetValue("gpt-4o-mini")
	m.baseURL.SetValue("")
	for _, ti := range m.formInputs() {
		ti.Width = max(20, width-8)
	}
	m.focusForm(0)
}

func (m *masterState) saveForm(cfg *config.Config) (*config.Config, error) {
	name := strings.TrimSpace(m.name.Value())
	if name == "" {
		return cfg, fmt.Errorf("profile name required")
	}

	p := config.Profile{
		APIKey:  strings.TrimSpace(m.apiKey.Value()),
		Model:   strings.TrimSpace(m.model.Value()),
		BaseURL: strings.TrimSpace(m.baseURL.Value()),
	}
	if p.APIKey == "" && p.BaseURL == "" {
		return cfg, fmt.Errorf("api key or base url required")
	}
	if err := cfg.Upsert(name, p); err != nil {
		return cfg, err
	}
	cfg.ActiveProfile = name
	if err := cfg.Save(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func (m *masterState) selectProfile(cfg *config.Config, name string) (*config.Config, *llm.Client, error) {
	if err := cfg.SetActive(name); err != nil {
		return cfg, nil, err
	}
	client, err := llm.New()
	if err != nil {
		return cfg, nil, err
	}
	return cfg, client, nil
}

func (m masterState) Update(msg tea.Msg, cfg *config.Config, width int) (masterState, *config.Config, *llm.Client, tea.Cmd) {
	if !m.open {
		return m, cfg, nil, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.phase == masterForm {
			return m.updateForm(msg, cfg)
		}
		return m.updateList(msg, cfg, width)
	}

	return m, cfg, nil, nil
}

func (m masterState) updateForm(msg tea.KeyMsg, cfg *config.Config) (masterState, *config.Config, *llm.Client, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+p":
		m.open = false
		m.phase = masterList
		m.err = ""
		return m, cfg, nil, textinput.Blink

	case "tab", "down":
		m.focusForm((m.formField + 1) % masterFieldCount)
		return m, cfg, nil, textinput.Blink

	case "shift+tab", "up":
		m.focusForm((m.formField + masterFieldCount - 1) % masterFieldCount)
		return m, cfg, nil, textinput.Blink

	case "enter":
		if m.formField < masterFieldCount-1 {
			m.focusForm(m.formField + 1)
			return m, cfg, nil, textinput.Blink
		}

		var err error
		cfg, err = m.saveForm(cfg)
		if err != nil {
			m.err = err.Error()
			return m, cfg, nil, nil
		}
		client, err := llm.New()
		if err != nil {
			m.err = err.Error()
			return m, cfg, nil, nil
		}
		m.phase = masterList
		m.open = false
		m.refresh(cfg)
		m.err = ""
		return m, cfg, client, textinput.Blink
	}

	ti := m.formInputs()[m.formField]
	var cmd tea.Cmd
	*ti, cmd = ti.Update(msg)
	return m, cfg, nil, cmd
}

func (m masterState) updateList(msg tea.KeyMsg, cfg *config.Config, width int) (masterState, *config.Config, *llm.Client, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+p":
		m.open = false
		m.err = ""
		return m, cfg, nil, textinput.Blink

	case "ctrl+n":
		m.openForm(width)
		return m, cfg, nil, textinput.Blink

	case "down":
		if len(m.names) > 0 && m.cursor < len(m.names)-1 {
			m.cursor++
		}
		return m, cfg, nil, nil

	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, cfg, nil, nil

	case "backspace", "ctrl+h":
		if len(m.names) > 0 {
			name := m.names[m.cursor]
			_ = cfg.Delete(name)
			m.refresh(cfg)
		}
		return m, cfg, nil, nil

	case "enter":
		if len(m.names) == 0 {
			m.openForm(width)
			return m, cfg, nil, textinput.Blink
		}
		name := m.names[m.cursor]
		var client *llm.Client
		var err error
		cfg, client, err = m.selectProfile(cfg, name)
		if err != nil {
			m.err = err.Error()
			return m, cfg, nil, nil
		}
		m.open = false
		m.refresh(cfg)
		return m, cfg, client, textinput.Blink
	}

	return m, cfg, nil, nil
}

func (m masterState) View(th theme, cfg *config.Config, width int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(th.labelActive.Render("● Profiles"))
	b.WriteString("\n\n")

	if m.err != "" {
		b.WriteString(th.error.Render("✗ " + m.err))
		b.WriteString("\n\n")
	}

	if m.phase == masterForm {
		b.WriteString(th.label.Render("New profile"))
		b.WriteString("\n\n")
		labels := []string{"Name", "API key", "Model", "Base URL"}
		inputs := m.formInputs()
		for i, label := range labels {
			if i == m.formField {
				b.WriteString(th.labelActive.Render("● " + label))
			} else {
				b.WriteString(th.label.Render("  " + label))
			}
			b.WriteString("\n")
			b.WriteString(inputs[i].View())
			b.WriteString("\n\n")
		}
	} else {
		if len(m.rows) == 0 {
			b.WriteString(th.empty.Render("  No profiles — press ctrl+n to add"))
		} else {
			for i, row := range m.rows {
				marker := "  "
				if i == m.cursor {
					marker = "› "
				}
				active := "  "
				if row.active {
					active = "● "
				}
				line := active + row.name
				if row.model != "" {
					line += th.histArrow.Render(" — ") + th.histCmd.Render(row.model)
				}
				if row.baseURL != "" {
					line += th.histArrow.Render(" @ ") + th.histPrompt.Render(truncateWidth(row.baseURL, 20))
				}
				if i == m.cursor {
					b.WriteString(th.histSelect.Render(marker + line))
				} else {
					b.WriteString(th.label.Render(marker + line))
				}
				b.WriteString("\n")
			}
		}
	}

	b.WriteString("\n")
	k := th.helpKey.Render
	d := th.helpText.Render
	if m.phase == masterForm {
		b.WriteString(d("") + k("enter") + d(" next/save  ") + k("tab") + d(" field  ") + k("esc") + d(" cancel"))
	} else {
		b.WriteString(d("") + k("enter") + d(" select  ") + k("ctrl+n") + d(" new  ") + k("backspace") + d(" delete  ") + k(infoKeyLabel()) + d(" info  ") + k("ctrl+p") + d(" close"))
	}

	return b.String()
}
