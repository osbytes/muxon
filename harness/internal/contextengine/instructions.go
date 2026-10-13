package contextengine

import (
	"os"
	"path/filepath"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

var projectInstructionFiles = []string{
	"AGENTS.md",
	"CONVENTIONS.md",
	"CLAUDE.md",
	".cursorrules",
}

// LoadProjectInstructions loads root project guidance per spec precedence.
func LoadProjectInstructions(workspaceRoot string, trusted bool) (string, error) {
	var parts []string
	home, err := config.TheeHome()
	if err == nil {
		user := filepath.Join(home, "AGENTS.md")
		if b, err := os.ReadFile(user); err == nil {
			parts = append(parts, string(b))
		}
	}
	for _, name := range projectInstructionFiles {
		p := filepath.Join(workspaceRoot, name)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		body := string(b)
		if !trusted {
			body = tools.FormatResult("instructions", false, body, 0, 0)
		}
		parts = append(parts, body)
		break
	}
	if len(parts) == 0 {
		return "", nil
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "\n\n"
		}
		out += p
	}
	return out, nil
}
