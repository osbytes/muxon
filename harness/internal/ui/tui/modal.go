package tui

import "github.com/charmbracelet/lipgloss"

// Modal is an approval or trust prompt.
type Modal struct {
	Kind      string
	Decision  string
	Done      bool
	Workspace string
}

// NewApprovalModal starts a tool-approval prompt.
func NewApprovalModal() *Modal {
	return &Modal{Kind: "approval"}
}

// NewTrustModal starts a first-run trust prompt for workspace.
func NewTrustModal(workspace string) *Modal {
	return &Modal{Kind: "trust", Workspace: workspace}
}

// Press applies one key. Approval: y, a, n, q. Trust: t, o, d (esc is don't).
func (m *Modal) Press(key string) {
	if m == nil || m.Done {
		return
	}
	switch m.Kind {
	case "approval":
		switch key {
		case "y":
			m.Decision = "allow_once"
		case "a":
			m.Decision = "allow_session"
		case "n":
			m.Decision = "deny"
		case "q":
			m.Decision = "cancel"
		default:
			return
		}
	case "trust":
		switch key {
		case "t":
			m.Decision = "trust"
		case "o":
			m.Decision = "trust_once"
		case "d", "esc":
			m.Decision = "dont_trust"
		default:
			return
		}
	default:
		return
	}
	m.Done = true
}

// View renders the modal body.
func (m *Modal) View() string {
	if m == nil {
		return ""
	}
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colorAccent).Padding(1, 3)
	key := func(k, label string) string {
		return accentStyle.Bold(true).Render(k) + " " + mutedStyle.Render(label)
	}
	if m.Kind == "trust" {
		return box.Render(textStyle.Bold(true).Render("Trust this workspace?") + "\n\n" +
			accentStyle.Render(shortPath(m.Workspace)) + "\n\n" +
			mutedStyle.Render("thee will read and edit files here and run its verify\ncommand and tools as you. Only trust code you know.") + "\n\n" +
			key("t", "trust") + "   " + key("o", "this session only") + "   " + key("d", "don't trust"))
	}
	return box.Render(textStyle.Bold(true).Render("Approval required") + "\n\n" +
		key("y", "allow once") + "   " + key("a", "allow for session") + "   " + key("n", "deny") + "   " + key("q", "cancel goal"))
}
