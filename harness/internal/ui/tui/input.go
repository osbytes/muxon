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
	case "provider", "connect":
		return Slash{Name: "providers", Rest: rest}, true
	case "models":
		return Slash{Name: "model", Rest: rest}, true
	case "quit":
		return Slash{Name: "exit", Rest: rest}, true
	case "help", "abort", "undo", "escalate", "cost", "providers", "trust", "model", "exit":
		return Slash{Name: name, Rest: rest}, true
	default:
		return Slash{}, false
	}
}

// HelpText lists the slash commands with one-line descriptions.
func HelpText() string {
	rows := make([]string, len(commandDocs))
	for i, c := range commandDocs {
		rows[i] = padRight(c.name, 12) + c.desc
	}
	return strings.Join(rows, "\n")
}
