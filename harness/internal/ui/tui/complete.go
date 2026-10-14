package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxSuggestionRows = 7

type suggestion struct {
	label  string
	desc   string
	insert string
}

func matchSuggestions(raw string) []suggestion {
	if !strings.HasPrefix(raw, "/") || strings.Contains(raw, "\n") {
		return nil
	}
	body := strings.TrimPrefix(raw, "/")
	token, rest, hasSpace := strings.Cut(body, " ")
	token = strings.ToLower(token)
	if hasSpace {
		return argumentSuggestions(token, rest)
	}
	var names, aliases []suggestion
	for _, c := range commandDocs {
		item := suggestion{label: c.name, desc: c.desc, insert: commandInsert(c.name)}
		name := strings.TrimPrefix(c.name, "/")
		if token == "" || strings.HasPrefix(name, token) {
			names = append(names, item)
			continue
		}
		for _, alias := range commandAliases[c.name] {
			if strings.HasPrefix(alias, token) {
				aliases = append(aliases, item)
				break
			}
		}
	}
	if token == "" || len(names) > 0 {
		return names
	}
	return aliases
}

func commandInsert(name string) string {
	switch name {
	case "/providers", "/model":
		return name + " "
	default:
		return name
	}
}

func argumentSuggestions(token, rest string) []suggestion {
	if strings.Contains(rest, " ") {
		return nil
	}
	cmd, ok := canonicalCommand(token)
	if !ok || cmd.name != "/providers" {
		return nil
	}
	arg := strings.ToLower(rest)
	var out []suggestion
	for _, c := range ProviderChoices() {
		if arg != "" && !strings.HasPrefix(c.ID, arg) && !strings.HasPrefix(strings.ToLower(c.Title), arg) {
			continue
		}
		out = append(out, suggestion{
			label:  c.ID,
			desc:   c.Title + " · " + c.ModelName,
			insert: "/providers " + c.ID,
		})
	}
	return out
}

func canonicalCommand(token string) (commandDoc, bool) {
	for _, c := range commandDocs {
		if token == strings.TrimPrefix(c.name, "/") {
			return c, true
		}
		for _, alias := range commandAliases[c.name] {
			if token == alias {
				return c, true
			}
		}
	}
	return commandDoc{}, false
}

func (m Model) currentSuggestions() []suggestion {
	if m.completeDismissed || m.inputBusy() || m.awaitingKey != "" || m.pickerOpen || m.modelPicker || (m.modal != nil && !m.modal.Done) {
		return nil
	}
	if list, ok := m.modelSuggestions(m.input.Value()); ok {
		return list
	}
	return matchSuggestions(m.input.Value())
}

func (m Model) suggestionLimit() int {
	room := max(m.height, 12) - chromeLines - 3
	if room < 1 {
		return 1
	}
	if room > maxSuggestionRows {
		return maxSuggestionRows
	}
	return room
}

func (m Model) suggestionWindow() ([]suggestion, int) {
	list := m.currentSuggestions()
	if len(list) == 0 {
		return nil, 0
	}
	idx := m.completeIndex
	if idx < 0 || idx >= len(list) {
		idx = 0
	}
	limit := m.suggestionLimit()
	if len(list) <= limit {
		return list, idx
	}
	start := idx - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > len(list) {
		start = len(list) - limit
	}
	return list[start : start+limit], idx - start
}

func (m Model) suggestionRows() int {
	visible, _ := m.suggestionWindow()
	return len(visible)
}

func (m Model) suggestionView(width int) string {
	visible, selected := m.suggestionWindow()
	if len(visible) == 0 {
		return ""
	}
	nameWidth := 0
	for _, s := range visible {
		nameWidth = max(nameWidth, lipgloss.Width(s.label))
	}
	rows := make([]string, len(visible))
	for i, s := range visible {
		rows[i] = suggestionRow(s, i == selected, nameWidth, width)
	}
	return strings.Join(rows, "\n")
}

func suggestionRow(s suggestion, selected bool, nameWidth, width int) string {
	const cursorWidth = 2
	const gap = 2
	base := lipgloss.NewStyle()
	cursor, cursorStyle := "  ", base
	nameStyle := base.Foreground(colorText)
	if selected {
		base = base.Background(colorSelect)
		cursor, cursorStyle = "❯ ", base.Foreground(colorAccent)
		nameStyle = base.Foreground(colorAccent).Bold(true)
	}
	desc := s.desc
	room := width - cursorWidth - nameWidth - gap
	if room < 4 {
		desc = ""
	} else if lipgloss.Width(desc) > room {
		desc = truncateRight(desc, room)
	}
	line := cursorStyle.Render(cursor) + nameStyle.Render(padRight(s.label, nameWidth))
	if desc != "" {
		line += base.Foreground(colorMuted).Render(strings.Repeat(" ", gap) + desc)
	}
	if extra := width - lipgloss.Width(line); extra > 0 {
		line += base.Render(strings.Repeat(" ", extra))
	}
	return line
}

func (m *Model) applySuggestion(list []suggestion) {
	idx := m.completeIndex
	if idx < 0 || idx >= len(list) {
		idx = 0
	}
	m.input.SetValue(list[idx].insert)
	m.completeIndex = 0
	m.completeDismissed = false
}

func (m Model) updateComplete(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	list := m.currentSuggestions()
	if len(list) == 0 {
		return m, nil, false
	}
	switch msg.String() {
	case "up", "ctrl+p", "shift+tab":
		m.completeIndex = (m.completeIndex + len(list) - 1) % len(list)
		return m, nil, true
	case "down", "ctrl+n":
		m.completeIndex = (m.completeIndex + 1) % len(list)
		return m, nil, true
	case "tab":
		m.applySuggestion(list)
		return m, m.noteInputChange(), true
	case "esc":
		m.completeDismissed = true
		m.layout()
		return m, nil, true
	case "enter":
		m.applySuggestion(list)
		line := strings.TrimSpace(m.input.Value())
		m.input.Reset()
		m.completeIndex = 0
		m.completeDismissed = false
		m.layout()
		model, cmd := m.submit(line)
		return model, cmd, true
	default:
		return m, nil, false
	}
}
