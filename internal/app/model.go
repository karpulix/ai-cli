package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/karpulix/ai-cli/internal/config"
	"github.com/karpulix/ai-cli/internal/history"
	"github.com/karpulix/ai-cli/internal/llm"
)

type focus int

const (
	borderWidth = 2
	hPadding    = 4
	inputPrompt = "❯ "
)

const (
	focusInput focus = iota
	focusHistory
)

type errMsg struct{ err error }

type shimmerTickMsg struct{}

func shimmerTick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(time.Time) tea.Msg {
		return shimmerTickMsg{}
	})
}

type doneMsg struct{ response string }

type Model struct {
	input        textinput.Model
	spinner      spinner.Model
	store        *history.Store
	cfg          *config.Config
	llm          *llm.Client
	entries      []history.Entry
	theme        theme
	master       masterState
	info         infoState
	cursor       int
	focus        focus
	loading      bool
	result       string
	quitting     bool
	err          string
	width        int
	height       int
	shineFrame   int
	profileName  string
}

func New(store *history.Store, cfg *config.Config, client *llm.Client, openMaster bool) Model {
	th := newTheme()

	ti := textinput.New()
	ti.Placeholder = "Describe the command you need..."
	ti.Focus()
	ti.CharLimit = 2000
	ti.Width = 60
	ti.Prompt = inputPrompt
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorPink).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorWhite)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(colorDim).Italic(true)
	ti.Cursor.Style = lipgloss.NewStyle().Background(colorPink).Foreground(colorWhite)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = th.spinner

	m := Model{
		input:   ti,
		spinner: s,
		store:   store,
		cfg:     cfg,
		llm:     client,
		theme:   th,
		entries: store.Entries(),
		focus:   focusInput,
		master:  newMasterState(60),
	}

	if name, err := m.activeProfileName(); err == nil {
		m.profileName = name
	}

	if openMaster || !cfg.HasProfiles() {
		m.master.open = true
		m.master.refresh(cfg)
		m.input.Blur()
	}

	return m
}

func (m Model) activeProfileName() (string, error) {
	_, name, err := m.cfg.Active()
	return name, err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, shimmerTick())
}

func (m *Model) openMasterPanel() {
	m.master.open = true
	m.master.phase = masterList
	m.master.err = ""
	m.master.refresh(m.cfg)
	m.info.open = false
	m.input.Blur()
}

func (m *Model) openInfoPanel() {
	m.info.open = true
	m.master.open = false
	m.input.Blur()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = m.inputWidth()
		for _, ti := range m.master.formInputs() {
			ti.Width = max(20, m.innerWidth())
		}
		return m, nil

	case shimmerTickMsg:
		if !m.quitting {
			m.shineFrame++
			return m, shimmerTick()
		}
		return m, nil

	case tea.KeyMsg:
		if m.loading {
			if msg.String() == "ctrl+c" || msg.String() == "esc" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}

		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		if isInfoKey(msg) {
			if m.info.open {
				m.info.open = false
				m.input.Focus()
			} else {
				m.openInfoPanel()
			}
			return m, textinput.Blink
		}

		if m.master.open {
			var client *llm.Client
			var cmd tea.Cmd
			m.master, m.cfg, client, cmd = m.master.Update(msg, m.cfg, m.innerWidth())
			if client != nil {
				m.llm = client
				if name, err := m.activeProfileName(); err == nil {
					m.profileName = name
				}
				m.input.Focus()
			}
			if !m.master.open {
				m.input.Focus()
			}
			return m, cmd
		}

		if m.info.open {
			var cmd tea.Cmd
			m.info, cmd = m.info.Update(msg)
			if !m.info.open {
				m.input.Focus()
			}
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit

		case "ctrl+p":
			m.openMasterPanel()
			return m, textinput.Blink

		case "tab":
			if m.focus == focusHistory && len(m.entries) > 0 {
				m.input.SetValue(m.entries[m.cursor].Prompt)
				m.input.CursorEnd()
				m.focus = focusInput
				m.input.Focus()
			} else if m.focus == focusInput && len(m.entries) > 0 {
				m.focus = focusHistory
				m.input.Blur()
			}
			return m, textinput.Blink

		case "up":
			if m.focus == focusHistory {
				if m.cursor > 0 {
					m.cursor--
				} else {
					m.focus = focusInput
					m.input.Focus()
				}
				return m, textinput.Blink
			}
			return m, nil

		case "down":
			if m.focus == focusHistory {
				if m.cursor < len(m.entries)-1 {
					m.cursor++
				}
				return m, nil
			}
			if m.focus == focusInput && len(m.entries) > 0 {
				m.focus = focusHistory
				m.input.Blur()
			}
			return m, nil

		case "backspace", "ctrl+h":
			if m.focus == focusHistory && len(m.entries) > 0 {
				_ = m.store.Delete(m.cursor)
				m.entries = m.store.Entries()
				if m.cursor >= len(m.entries) && m.cursor > 0 {
					m.cursor--
				}
				if len(m.entries) == 0 {
					m.focus = focusInput
					m.input.Focus()
				}
				return m, textinput.Blink
			}
			if m.focus == focusInput {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}

		case "enter":
			if m.focus == focusHistory && len(m.entries) > 0 {
				m.result = m.entries[m.cursor].Response
				m.quitting = true
				return m, tea.Quit
			}
			if m.master.open || m.llm == nil {
				if m.cfg.HasProfiles() {
					var err error
					m.llm, err = llm.New()
					if err != nil {
						m.err = err.Error()
						return m, nil
					}
				} else {
					m.openMasterPanel()
					return m, textinput.Blink
				}
			}
			prompt := strings.TrimSpace(m.input.Value())
			if prompt == "" {
				return m, nil
			}
			m.loading = true
			m.err = ""
			return m, tea.Batch(m.spinner.Tick, m.fetch(prompt))

		default:
			if m.focus == focusInput {
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		}

	case errMsg:
		m.loading = false
		m.err = msg.err.Error()
		return m, nil

	case doneMsg:
		m.loading = false
		prompt := strings.TrimSpace(m.input.Value())
		m.result = msg.response
		_ = m.store.Add(prompt, msg.response)
		m.entries = m.store.Entries()
		m.quitting = true
		return m, tea.Quit

	default:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		if m.master.open && m.master.phase == masterForm {
			var client *llm.Client
			var cmd tea.Cmd
			m.master, m.cfg, client, cmd = m.master.Update(msg, m.cfg, m.innerWidth())
			if client != nil {
				m.llm = client
			}
			return m, cmd
		}
		if m.focus == focusInput && !m.master.open && !m.info.open {
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m Model) fetch(prompt string) tea.Cmd {
	return func() tea.Msg {
		client := m.llm
		if client == nil {
			var err error
			client, err = llm.New()
			if err != nil {
				return errMsg{err: err}
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		resp, err := client.Complete(ctx, prompt)
		if err != nil {
			return errMsg{err: err}
		}
		return doneMsg{response: resp}
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	th := m.theme
	var b strings.Builder

	boxStyle := th.box.Width(m.boxWidth())
	inner := m.innerWidth()

	b.WriteString(th.shimmerTitle("ai-cli", m.shineFrame))
	if m.profileName != "" && !m.master.open && !m.info.open {
		b.WriteString(th.label.Render("  · " + m.profileName))
	}
	b.WriteString("\n\n")

	if m.master.open {
		b.WriteString(m.master.View(th, m.cfg, inner))
		return boxStyle.Render(b.String())
	}

	if m.info.open {
		b.WriteString(m.info.View(th, m.cfg, m.store, m.profileName, inner))
		return boxStyle.Render(b.String())
	}

	if m.loading {
		b.WriteString(m.spinner.View())
		b.WriteString(th.loading.Render("  Generating command..."))
		b.WriteString("\n\n")
	} else if m.err != "" {
		b.WriteString(th.error.Render(fmt.Sprintf("✗ Error: %v", m.err)))
		b.WriteString("\n\n")
	}

	b.WriteString(th.sectionLabel("Prompt", m.focus == focusInput))
	b.WriteString("\n")
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	b.WriteString(th.sectionLabel("History", m.focus == focusHistory))
	b.WriteString("\n")

	if len(m.entries) == 0 {
		b.WriteString(th.empty.Render("  No history yet"))
	} else {
		maxShow := min(len(m.entries), max(3, m.height-16))
		offset := 0
		if m.cursor >= maxShow {
			offset = m.cursor - maxShow + 1
		}
		if offset > len(m.entries)-maxShow {
			offset = max(0, len(m.entries)-maxShow)
		}
		end := min(offset+maxShow, len(m.entries))
		for i := offset; i < end; i++ {
			e := m.entries[i]
			sep := " → "
			prefixW := 2
			if i == m.cursor && m.focus == focusHistory {
				prefixW = lipgloss.Width("› ")
			}
			avail := inner - prefixW - lipgloss.Width(sep)
			promptW := max(8, avail*3/5)
			respW := max(8, avail-promptW)
			plainPrompt := truncateWidth(e.Prompt, promptW)
			plainResp := truncateWidth(e.Response, respW)

			if i == m.cursor && m.focus == focusHistory {
				line := th.histSelect.Render("› " + plainPrompt + sep + plainResp)
				b.WriteString(line)
			} else {
				b.WriteString("  ")
				b.WriteString(th.histPrompt.Render(plainPrompt))
				b.WriteString(th.histArrow.Render(sep))
				b.WriteString(th.histCmd.Render(plainResp))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(th.helpLine())

	return boxStyle.Render(b.String())
}

func (m Model) Result() string {
	return m.result
}

func (m Model) terminalWidth() int {
	if m.width <= 0 {
		return 80
	}
	return m.width
}

func (m Model) boxWidth() int {
	return max(20, m.terminalWidth()-borderWidth)
}

func (m Model) innerWidth() int {
	return max(20, m.boxWidth()-hPadding)
}

func (m Model) inputWidth() int {
	return max(10, m.innerWidth()-lipgloss.Width(inputPrompt))
}

func truncateWidth(s string, maxW int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if lipgloss.Width(s) <= maxW {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if lipgloss.Width(b.String()+string(r)+"…") > maxW {
			break
		}
		b.WriteRune(r)
	}
	return b.String() + "…"
}
