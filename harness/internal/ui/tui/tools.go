package tui

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/PLACEHOLDER/thee/harness/internal/director"
)

type toolCall struct {
	name   string
	args   toolArgs
	done   bool
	ok     bool
	output string
}

type toolArgs struct {
	Path      string   `json:"path"`
	Pattern   string   `json:"pattern"`
	Glob      string   `json:"glob"`
	Action    string   `json:"action"`
	Text      string   `json:"text"`
	Content   string   `json:"content"`
	Message   string   `json:"message"`
	Filter    string   `json:"filter"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Paths     []string `json:"paths"`
}

type toolMsg struct {
	act director.ToolActivity
}

var hiddenTools = map[string]bool{"ask_user": true, "finish": true}

var groupNouns = map[string][2]string{
	"read":    {"Read", "files"},
	"search":  {"Searched", "patterns"},
	"list":    {"Listed", "directories"},
	"repomap": {"Mapped", "paths"},
}

func newToolCall(act director.ToolActivity) *toolCall {
	tc := &toolCall{name: act.Name}
	_ = json.Unmarshal(act.Args, &tc.args)
	return tc
}

func (tc *toolCall) finish(act director.ToolActivity) {
	tc.done = true
	tc.ok = act.OK
	tc.output = act.Output
}

func (tc *toolCall) title() string {
	a := tc.args
	switch tc.name {
	case "read":
		label := "Read " + a.Path
		if a.StartLine > 0 && a.EndLine > 0 {
			label += fmt.Sprintf(":%d-%d", a.StartLine, a.EndLine)
		}
		return label
	case "search":
		label := fmt.Sprintf("Search %q", a.Pattern)
		if a.Path != "" && a.Path != "." {
			label += " in " + a.Path
		}
		if a.Glob != "" {
			label += " (" + a.Glob + ")"
		}
		return label
	case "list":
		return "List " + orDot(a.Path)
	case "repomap":
		return "Repo map " + orDot(a.Path)
	case "write":
		return "Write " + a.Path
	case "patch":
		return "Edit " + a.Path
	case "delete":
		return "Delete " + a.Path
	case "test":
		return strings.TrimSpace("Test " + a.Filter)
	case "git":
		return strings.TrimSpace("Git " + a.Action + " " + strings.Join(a.Paths, " "))
	case "todo":
		return strings.TrimSpace("Todo " + a.Action + " " + a.Text)
	case "checkpoint":
		return strings.TrimSpace("Checkpoint " + a.Action + " " + a.Path)
	default:
		return tc.name
	}
}

func (tc *toolCall) summary() string {
	if !tc.done {
		return "running…"
	}
	if !tc.ok {
		return firstLine(tc.output)
	}
	n := countLines(tc.output)
	switch tc.name {
	case "read":
		return plural(n, "line")
	case "search":
		if n == 0 {
			return "no matches"
		}
		return plural(n, "match")
	case "list", "repomap":
		return plural(n, "entry")
	case "write":
		return plural(countLines(tc.args.Content), "line") + " written"
	case "patch":
		return "applied"
	default:
		return firstLine(tc.output)
	}
}

func renderToolGroup(group []*toolCall, expanded bool, width int) string {
	if len(group) == 1 || expanded {
		rows := make([]string, len(group))
		for i, tc := range group {
			rows[i] = renderToolRow(tc.title(), tc.summary(), toolBullet(tc.done, tc.ok), !tc.ok && tc.done, width)
		}
		return strings.Join(rows, "\n")
	}
	nouns := groupNouns[group[0].name]
	done, failed := true, 0
	var targets []string
	for _, tc := range group {
		done = done && tc.done
		if tc.done && !tc.ok {
			failed++
		}
		targets = append(targets, tc.target())
	}
	title := fmt.Sprintf("%s %d %s", nouns[0], len(group), nouns[1])
	detail := strings.Join(uniqueTail(targets, 3), " · ")
	if failed > 0 {
		detail = fmt.Sprintf("%d failed · %s", failed, detail)
	}
	return renderToolRow(title, detail, toolBullet(done, failed == 0), false, width-18) + mutedStyle.Render("  (ctrl+o expand)")
}

func renderToolRow(title, detail string, bullet lipgloss.Style, failed bool, width int) string {
	head := "  " + bullet.Render("●") + " " + textStyle.Bold(true).Render(truncateRight(title, width-6))
	if detail == "" {
		return head
	}
	style := mutedStyle
	if failed {
		style = errorStyle
	}
	return head + "\n" + mutedStyle.Render("    ⎿  ") + style.Render(truncateRight(detail, width-8))
}

func toolBullet(done, ok bool) lipgloss.Style {
	switch {
	case !done:
		return accentStyle
	case ok:
		return successStyle
	default:
		return errorStyle
	}
}

func (tc *toolCall) target() string {
	switch tc.name {
	case "search":
		return fmt.Sprintf("%q", tc.args.Pattern)
	default:
		return filepath.Base(orDot(tc.args.Path))
	}
}

func groupable(a, b *toolCall) bool {
	_, ok := groupNouns[a.name]
	return ok && a.name == b.name
}

func uniqueTail(items []string, n int) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	if len(out) <= n {
		return out
	}
	return append(out[:n], fmt.Sprintf("+%d more", len(out)-n))
}

func orDot(p string) string {
	if p == "" {
		return "."
	}
	return p
}

func countLines(s string) int {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i] + " …"
	}
	return s
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	switch {
	case strings.HasSuffix(noun, "ch"):
		noun += "es"
	case strings.HasSuffix(noun, "y"):
		noun = strings.TrimSuffix(noun, "y") + "ies"
	default:
		noun += "s"
	}
	return fmt.Sprintf("%d %s", n, noun)
}

func truncateRight(s string, n int) string {
	r := []rune(s)
	if n < 2 || len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
