package tui

import "strings"

// Slash is a parsed interactive command.
type Slash struct {
	Name string
	Rest string
}

// ParseSlash reports whether line is a known slash command.
// Unknown /foo is not a command so it can be sent as a goal.
func ParseSlash(line string) (Slash, bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "/") {
		return Slash{}, false
	}
	fields := strings.SplitN(line, " ", 2)
	name := strings.TrimPrefix(fields[0], "/")
	rest := ""
	if len(fields) == 2 {
		rest = fields[1]
	}
	switch name {
	case "help", "abort", "undo", "escalate", "cost":
		return Slash{Name: name, Rest: rest}, true
	default:
		return Slash{}, false
	}
}

// HelpText is the short slash list.
func HelpText() string {
	return "/help /abort /undo /escalate /cost"
}
