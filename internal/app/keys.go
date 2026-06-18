package app

import tea "github.com/charmbracelet/bubbletea"

func isInfoKey(msg tea.KeyMsg) bool {
	return msg.Type == tea.KeyCtrlO || msg.String() == "alt+i"
}

func infoKeyLabel() string {
	return "ctrl+o"
}
