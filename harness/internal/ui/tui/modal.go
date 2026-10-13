package tui

// Modal is an approval or trust prompt.
type Modal struct {
	Kind     string
	Decision string
	Done     bool
}

// NewApprovalModal starts a tool-approval prompt.
func NewApprovalModal() *Modal {
	return &Modal{Kind: "approval"}
}

// NewTrustModal starts a first-run trust prompt.
func NewTrustModal() *Modal {
	return &Modal{Kind: "trust"}
}

// Press applies one key. Approval: y, a, n, q. Trust: t, o, d.
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
		case "d":
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
	if m.Kind == "trust" {
		return "trust this workspace? [t] trust [o] once [d] don't"
	}
	return "approval required [y] once [a] session [n] deny [q] cancel"
}
