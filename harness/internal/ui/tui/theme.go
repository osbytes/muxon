package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorAccent  = lipgloss.Color("#FAB283")
	colorBlue    = lipgloss.Color("#5C9CF5")
	colorText    = lipgloss.Color("#EEEEEE")
	colorMuted   = lipgloss.Color("#808080")
	colorBorder  = lipgloss.Color("#4B4C5C")
	colorSuccess = lipgloss.Color("#7FD88F")
	colorError   = lipgloss.Color("#E06C75")
	colorWarn    = lipgloss.Color("#F5A742")
)

const logo = "▀█▀ █ █ █▀▀ █▀▀\n █  █▀█ ██▄ ██▄"

var (
	textStyle    = lipgloss.NewStyle().Foreground(colorText)
	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted)
	accentStyle  = lipgloss.NewStyle().Foreground(colorAccent)
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	errorStyle   = lipgloss.NewStyle().Foreground(colorError)
	warnStyle    = lipgloss.NewStyle().Foreground(colorWarn)
	userStyle    = lipgloss.NewStyle().
			Border(lipgloss.ThickBorder(), false, false, false, true).
			BorderForeground(colorBlue).
			Foreground(colorText).
			PaddingLeft(1)
	assistantStyle = lipgloss.NewStyle().Foreground(colorText).PaddingLeft(2)
	infoStyle      = lipgloss.NewStyle().Foreground(colorMuted).PaddingLeft(2)
)

type commandDoc struct {
	name string
	desc string
}

var commandDocs = []commandDoc{
	{"/providers", "connect a model provider (OpenCode Go, DeepInfra)"},
	{"/help", "show commands"},
	{"/trust", "trust this workspace so goals can run"},
	{"/cost", "session cost and tokens"},
	{"/escalate", "use the stronger model on the next goal"},
	{"/undo", "revert the last patch"},
	{"/abort", "cancel the running goal"},
}

func commandList(nameStyle, descStyle lipgloss.Style) string {
	rows := make([]string, len(commandDocs))
	widest := 0
	for i, c := range commandDocs {
		rows[i] = nameStyle.Render(padRight(c.name, 12)) + descStyle.Render(c.desc)
		widest = max(widest, lipgloss.Width(rows[i]))
	}
	for i := range rows {
		rows[i] = padRight(rows[i], widest)
	}
	return strings.Join(rows, "\n")
}

func truncateLeft(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return "…" + string(r[len(r)-n+1:])
}

func displayVersion(v string) string {
	if v == "" || v == "dev" {
		return "dev"
	}
	return "v" + strings.TrimPrefix(v, "v")
}

func padRight(s string, n int) string {
	if w := lipgloss.Width(s); w < n {
		return s + strings.Repeat(" ", n-w)
	}
	return s
}

func shortPath(p string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

func renderTranscript(lines []string, width int) string {
	if width < 10 {
		width = 10
	}
	var blocks []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		kind := lineKind(line)
		if kind == "tool" && i+1 < len(lines) && strings.HasPrefix(lines[i+1], "▸ result ") {
			blocks = append(blocks, renderTool(line, lines[i+1]))
			i++
			continue
		}
		if len(blocks) > 0 && kind != "tool" {
			blocks = append(blocks, "")
		}
		blocks = append(blocks, renderLine(line, kind, width))
	}
	return strings.Join(blocks, "\n")
}

func lineKind(line string) string {
	switch {
	case strings.HasPrefix(line, "> "):
		return "user"
	case strings.HasPrefix(line, "[assistant] "):
		return "assistant"
	case strings.HasPrefix(line, "▸ verify "):
		return "verify"
	case strings.HasPrefix(line, "▸ result "):
		return "result"
	case strings.HasPrefix(line, "▸ "):
		return "tool"
	case strings.HasPrefix(line, "[harness] "):
		return "harness"
	case strings.HasPrefix(line, "[done]"), strings.HasPrefix(line, "[failed]"), strings.HasPrefix(line, "[cancelled]"):
		return "outcome"
	case strings.HasPrefix(line, "No provider"), strings.HasPrefix(line, "Still connecting"):
		return "warn"
	case strings.HasPrefix(line, "[error] "):
		return "error"
	case strings.HasPrefix(line, "[connected] "):
		return "connected"
	case strings.HasPrefix(line, "[question] "):
		return "question"
	default:
		return "info"
	}
}

func renderLine(line, kind string, width int) string {
	switch kind {
	case "user":
		return userStyle.Width(width - 1).Render(strings.TrimPrefix(line, "> "))
	case "assistant":
		return assistantStyle.Width(width).Render(strings.TrimPrefix(line, "[assistant] "))
	case "tool":
		return renderTool(line, "")
	case "result":
		return "  " + resultMark(line)
	case "verify":
		if strings.HasSuffix(line, "pass") {
			return "  " + successStyle.Render("✓ verify passed")
		}
		return "  " + errorStyle.Render("✗ verify failed")
	case "harness":
		return infoStyle.Italic(true).Render(strings.TrimPrefix(line, "[harness] "))
	case "outcome":
		return "  " + renderOutcome(line)
	case "warn":
		return "  " + warnStyle.Render(line)
	case "error":
		return lipgloss.NewStyle().Foreground(colorError).PaddingLeft(2).Width(width).Render("✗ " + strings.TrimPrefix(line, "[error] "))
	case "connected":
		return "  " + successStyle.Render("● connected ") + textStyle.Render(strings.TrimPrefix(line, "[connected] "))
	case "question":
		body := strings.TrimPrefix(line, "[question] ")
		return lipgloss.NewStyle().Foreground(colorAccent).PaddingLeft(2).Width(width).Render("? " + body)
	default:
		return infoStyle.Width(width).Render(line)
	}
}

func renderTool(call, result string) string {
	name := strings.TrimPrefix(call, "▸ ")
	out := "  " + mutedStyle.Render("⚙ ") + textStyle.Render(name)
	if result != "" {
		out += "  " + resultMark(result)
	}
	return out
}

func resultMark(result string) string {
	if strings.HasSuffix(result, "ok") {
		return successStyle.Render("✓")
	}
	return errorStyle.Render("✗")
}

func renderOutcome(line string) string {
	end := strings.Index(line, "]")
	if end < 0 {
		return line
	}
	state := line[1:end]
	reason := strings.ReplaceAll(strings.TrimSpace(line[end+1:]), "_", " ")
	label := state
	if reason != "" {
		label += " · " + reason
	}
	switch state {
	case "done":
		return successStyle.Render("● " + label)
	case "failed":
		return errorStyle.Render("● " + label)
	default:
		return warnStyle.Render("● " + label)
	}
}
