package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/director"
	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/session"
)

// Options configures an interactive session.
type Options struct {
	Workspace   string
	SessionID   string
	Trusted     bool
	Provider    provider.Provider
	Manifest    *config.Manifest
	Version     string
	WorkerModel string
	Permission  string
}

// Model is the Bubble Tea session UI.
type Model struct {
	opt           Options
	input         textinput.Model
	transcript    []string
	state         string
	rounds        int
	cost          string
	modal         *Modal
	forceEscalate bool
	lastPatch     string
	lastOutcome   director.Outcome
	lastInterrupt time.Time
	notice        string
	run           *runCtrl
	headerStyle   lipgloss.Style
}

type runCtrl struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

type goalDone struct {
	out   director.Outcome
	err   error
	lines []string
	patch string
}

// NewModel builds the TUI model.
func NewModel(opt Options) Model {
	if opt.SessionID == "" {
		opt.SessionID = session.NewID()
	}
	if opt.Permission == "" {
		opt.Permission = "edit"
	}
	ti := textinput.New()
	ti.Placeholder = "goal"
	ti.Focus()
	m := Model{
		opt:         opt,
		input:       ti,
		state:       "idle",
		cost:        "$—",
		run:         &runCtrl{},
		headerStyle: lipgloss.NewStyle().Bold(true),
	}
	if !opt.Trusted {
		if _, err := os.Stat(filepath.Join(opt.Workspace, "thee.toml")); err == nil {
			m.modal = NewTrustModal()
		}
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// LastOutcome is the most recent goal result.
func (m Model) LastOutcome() director.Outcome {
	return m.lastOutcome
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.modal != nil && !m.modal.Done {
			m.modal.Press(msg.String())
			if m.modal.Done {
				m.applyModal()
			}
			return m, nil
		}
		if msg.String() == "ctrl+c" {
			m.Interrupt(time.Now())
			return m, nil
		}
		if msg.Type == tea.KeyEnter {
			line := m.input.Value()
			m.input.SetValue("")
			return m.submit(line)
		}
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	case goalDone:
		m.state = msg.out.State
		m.lastOutcome = msg.out
		m.transcript = append(m.transcript, msg.lines...)
		if msg.patch != "" {
			m.lastPatch = msg.patch
		}
		if msg.out.State == director.StateDone || msg.out.State == director.StateFailed || msg.out.State == director.StateCancelled {
			m.state = msg.out.State
			m.transcript = append(m.transcript, fmt.Sprintf("[%s] %s", msg.out.State, msg.out.Reason))
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) applyModal() {
	if m.modal == nil {
		return
	}
	switch m.modal.Decision {
	case "trust", "trust_once":
		root := m.opt.Workspace
		s, err := security.LoadTrust()
		if err == nil {
			s.AddRootWithConfigHash(root, security.HashTheeToml(root))
			_ = security.SaveTrust(s)
		}
		if m.modal.Decision == "trust" {
			m.opt.Trusted = true
		}
		m.notice = m.modal.Decision
	case "allow_once", "allow_session":
		m.notice = m.modal.Decision
	case "deny", "cancel", "dont_trust":
		m.notice = m.modal.Decision
	}
	m.modal = nil
}

func (m Model) submit(line string) (tea.Model, tea.Cmd) {
	if slash, ok := ParseSlash(line); ok {
		switch slash.Name {
		case "help":
			m.transcript = append(m.transcript, HelpText())
		case "abort":
			m.transcript = append(m.transcript, "cancelled")
			m.lastOutcome = director.Outcome{State: director.StateCancelled, Reason: "user_cancel", ExitCode: 130}
			m.state = director.StateCancelled
			if m.run != nil {
				m.run.mu.Lock()
				if m.run.cancel != nil {
					m.run.cancel()
				}
				m.run.mu.Unlock()
			}
		case "undo":
			if m.lastPatch == "" {
				m.transcript = append(m.transcript, "nothing to undo")
				break
			}
			if err := director.RestoreNewest(m.opt.Workspace, m.opt.SessionID, m.lastPatch); err != nil {
				m.transcript = append(m.transcript, "undo failed")
			} else {
				m.transcript = append(m.transcript, "undid "+m.lastPatch)
			}
		case "escalate":
			m.forceEscalate = true
			m.transcript = append(m.transcript, "escalate on next goal")
		case "cost":
			m.transcript = append(m.transcript, m.costLine())
		}
		return m, nil
	}
	if line == "" {
		return m, nil
	}
	m.transcript = append(m.transcript, "> "+line)
	m.state = "acting"
	goal := line
	force := m.forceEscalate
	m.forceEscalate = false
	return m, m.runGoal(goal, force)
}

func (m Model) runGoal(goal string, force bool) tea.Cmd {
	opt := m.opt
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.run.mu.Lock()
		m.run.cancel = cancel
		m.run.mu.Unlock()
		defer cancel()
		var lines []string
		var patch string
		out, err := director.Run(ctx, director.Config{
			Goal:             goal,
			Workspace:        opt.Workspace,
			Trusted:          opt.Trusted || security.IsTrustedRoot(opt.Workspace),
			TheeVersion:      opt.Version,
			Provider:         opt.Provider,
			Manifest:         opt.Manifest,
			SessionID:        opt.SessionID,
			ForceEscalate:    force,
			Interactive:      true,
			RegisterAllTools: true,
			OnEvent: func(ev observability.Event) {
				if line, ok := FormatEvent(ev); ok {
					lines = append(lines, line)
				}
				if ev.Type == "tool.call" && ev.Name == "patch" {
					patch = "sum.go"
				}
			},
		})
		return goalDone{out: out, err: err, lines: lines, patch: patch}
	}
}

// Interrupt cancels the in-flight goal. A second call within 2s aborts.
func (m *Model) Interrupt(now time.Time) {
	if !m.lastInterrupt.IsZero() && now.Sub(m.lastInterrupt) < 2*time.Second {
		m.lastOutcome = director.Outcome{State: director.StateCancelled, Reason: "user_cancel", ExitCode: 130}
		m.state = director.StateCancelled
		m.transcript = append(m.transcript, "cancelled")
	}
	m.lastInterrupt = now
	if m.run != nil {
		m.run.mu.Lock()
		if m.run.cancel != nil {
			m.run.cancel()
		}
		m.run.mu.Unlock()
	}
}

func (m Model) costLine() string {
	return fmt.Sprintf("status cost %s rounds %d", m.cost, m.rounds)
}

// View implements tea.Model.
func (m Model) View() string {
	header := m.headerStyle.Render(fmt.Sprintf("header %s worker:%s %s trusted:%t", m.opt.Workspace, m.opt.WorkerModel, m.opt.Permission, m.opt.Trusted))
	var body string
	if m.modal != nil && !m.modal.Done {
		body = m.modal.View()
	} else {
		body = "transcript\n" + joinLines(m.transcript)
	}
	status := fmt.Sprintf("status state:%s rounds:%d %s", m.state, m.rounds, m.cost)
	return header + "\n" + body + "\n" + status + "\ninput " + m.input.View() + "\n"
}

func joinLines(lines []string) string {
	out := ""
	for _, l := range lines {
		out += l + "\n"
	}
	return out
}

// Run starts the interactive program.
func Run(opt Options) error {
	p := tea.NewProgram(NewModel(opt))
	_, err := p.Run()
	return err
}
