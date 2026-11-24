package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/director"
	"github.com/PLACEHOLDER/muxon/harness/internal/observability"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/security"
	"github.com/PLACEHOLDER/muxon/harness/internal/session"
)

// Options configures an interactive session.
type Options struct {
	Workspace   string
	SessionID   string
	Trusted     bool
	Provider    provider.Provider
	ProviderID  string
	Manifest    *config.Manifest
	Version     string
	WorkerModel string
	Permission  string
	SetupHint   string
	AutoConnect string
}

// Model is the Bubble Tea session UI.
type Model struct {
	opt                 Options
	input               textarea.Model
	keyInput            textinput.Model
	awaitingKey         string
	pickerOpen          bool
	pickerIndex         int
	connecting          string
	pendingGoal         string
	asking              *askUserMsg
	startCmd            tea.Cmd
	viewport            viewport.Model
	transcript          []entry
	history             []provider.Message
	expandTools         bool
	completeIndex       int
	completeDismissed   bool
	choice              ProviderChoice
	modelChoices        []string
	modelsLoading       bool
	modelsAuthoritative bool
	modelEpoch          int
	modelPicker         bool
	modelPickerIndex    int
	state               string
	taskType            string
	rounds              int
	verifies            int
	inputTokens         int
	outputTokens        int
	cachedTokens        int
	costUSD             float64
	cost                string
	modal               *Modal
	forceEscalate       bool
	lastPatch           string
	lastOutcome         director.Outcome
	lastInterrupt       time.Time
	notice              string
	width               int
	height              int
	run                 *runCtrl
	events              <-chan tea.Msg
}

type runCtrl struct {
	mu     sync.Mutex
	cancel context.CancelFunc
}

type liveEvent struct {
	ev observability.Event
}

type assistantLine struct {
	text string
}

type askUserMsg struct {
	question string
	choices  []string
	reply    chan<- string
}

func formatQuestion(question string, choices []string) string {
	var b strings.Builder
	b.WriteString(question)
	for i, c := range choices {
		fmt.Fprintf(&b, "\n%d. %s", i+1, c)
	}
	return b.String()
}

func (m *Model) answerQuestion(line string) {
	answer := line
	if n, err := strconv.Atoi(line); err == nil && n >= 1 && n <= len(m.asking.choices) {
		answer = m.asking.choices[n-1]
	}
	m.appendLine("> " + line)
	m.asking.reply <- answer
	m.asking = nil
	m.state = director.StateActing
	m.input.Blur()
}

type goalDone struct {
	out director.Outcome
	err error
}

// NewModel builds the TUI model.
func NewModel(opt Options) Model {
	if opt.SessionID == "" {
		opt.SessionID = session.NewID()
	}
	if opt.Permission == "" {
		opt.Permission = "edit"
	}
	if opt.ProviderID == "" {
		opt.ProviderID = "deepinfra"
	}
	ti := textarea.New()
	ti.Placeholder = "Ask muxon to change something  (/help for commands)"
	ti.Prompt = ""
	ti.ShowLineNumbers = false
	ti.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("alt+enter", "shift+enter"))
	inputStyle := textarea.Style{
		Base:        lipgloss.NewStyle(),
		CursorLine:  lipgloss.NewStyle(),
		EndOfBuffer: lipgloss.NewStyle().Foreground(colorBorder),
		Placeholder: mutedStyle,
		Prompt:      accentStyle,
		Text:        textStyle,
	}
	ti.FocusedStyle = inputStyle
	ti.BlurredStyle = inputStyle
	ti.Focus()
	keyIn := textinput.New()
	keyIn.Prompt = ""
	keyIn.Placeholder = "paste your API key"
	keyIn.PlaceholderStyle = mutedStyle
	keyIn.EchoMode = textinput.EchoPassword
	keyIn.EchoCharacter = '•'
	m := Model{
		opt:      opt,
		input:    ti,
		keyInput: keyIn,
		viewport: viewport.New(76, 16),
		state:    "idle",
		cost:     "$0.00",
		width:    80,
		height:   24,
		run:      &runCtrl{},
	}
	m.layout()
	if opt.Provider == nil && opt.AutoConnect != "" {
		m.startCmd = m.autoConnect(opt.AutoConnect)
	}
	if !m.opt.Trusted && security.IsTrustedRoot(opt.Workspace) {
		m.opt.Trusted = true
	}
	if !m.opt.Trusted {
		m.modal = NewTrustModal(opt.Workspace)
	}
	return m
}

func (m Model) trusted() bool {
	return m.opt.Trusted
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.input.Focus(), m.startCmd)
}

// LastOutcome is the most recent goal result.
func (m Model) LastOutcome() director.Outcome {
	return m.lastOutcome
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout()
		return m, nil
	case providerConnected:
		m.finishConnect(msg)
		return m, nil
	case modelsLoaded:
		m.finishModels(msg)
		m.layout()
		return m, nil
	case tea.KeyMsg:
		if m.awaitingKey != "" {
			return m.updateKeyPrompt(msg)
		}
		if m.pickerOpen {
			return m.updatePicker(msg)
		}
		if m.modelPicker {
			return m.updateModelPicker(msg)
		}
		if m.modal != nil && !m.modal.Done {
			m.modal.Press(msg.String())
			if m.modal.Done {
				cmd := m.applyModal()
				return m, cmd
			}
			return m, nil
		}
		if msg.String() == "ctrl+c" {
			m.Interrupt(time.Now())
			return m, nil
		}
		if msg.String() == "ctrl+o" {
			m.expandTools = !m.expandTools
			m.syncViewport()
			return m, nil
		}
		if m.inputBusy() {
			return m, nil
		}
		if model, cmd, ok := m.updateComplete(msg); ok {
			return model, cmd
		}
		if msg.String() == "enter" {
			line := strings.TrimSpace(m.input.Value())
			m.input.Reset()
			m.completeIndex = 0
			m.completeDismissed = false
			return m.submit(line)
		}
		prev := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != prev {
			cmd = tea.Batch(cmd, m.noteInputChange())
		}
		return m, cmd
	case liveEvent:
		m.applyEvent(msg.ev)
		return m, listen(m.events)
	case assistantLine:
		m.appendLine("[assistant] " + msg.text)
		return m, listen(m.events)
	case toolMsg:
		m.applyTool(msg.act)
		return m, listen(m.events)
	case askUserMsg:
		m.asking = &msg
		m.state = director.StateBlocked
		m.appendLine("[question] " + formatQuestion(msg.question, msg.choices))
		return m, tea.Batch(m.input.Focus(), listen(m.events))
	case goalDone:
		m.asking = nil
		if len(msg.out.History) > 0 {
			m.history = msg.out.History
		}
		m.lastOutcome = msg.out
		if msg.out.State != "" {
			m.state = msg.out.State
		}
		if msg.out.State == director.StateDone || msg.out.State == director.StateFailed || msg.out.State == director.StatePaused || msg.out.State == director.StateCancelled {
			m.appendLine(fmt.Sprintf("[%s] %s", msg.out.State, msg.out.Reason))
		}
		if msg.err != nil {
			m.appendLine("[error] " + shortError(msg.err))
		}
		m.events = nil
		cmd := m.input.Focus()
		return m, cmd
	}
	return m, nil
}

func (m Model) updateKeyPrompt(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "ctrl+c" || msg.String() == "esc" {
		m.awaitingKey = ""
		m.keyInput.SetValue("")
		m.appendLine("provider setup cancelled")
		cmd := m.input.Focus()
		return m, cmd
	}
	if msg.String() == "enter" {
		secret := strings.TrimSpace(m.keyInput.Value())
		choice, ok := choiceByArg(m.awaitingKey)
		m.keyInput.SetValue("")
		m.awaitingKey = ""
		if !ok {
			m.appendLine("[error] unknown provider")
			cmd := m.input.Focus()
			return m, cmd
		}
		cmd := m.beginConnect(choice, secret)
		return m, tea.Batch(m.input.Focus(), cmd)
	}
	var cmd tea.Cmd
	m.keyInput, cmd = m.keyInput.Update(msg)
	return m, cmd
}

func (m *Model) applyModal() tea.Cmd {
	if m.modal == nil {
		return nil
	}
	decision := m.modal.Decision
	m.notice = decision
	m.modal = nil
	switch decision {
	case "trust":
		root := m.opt.Workspace
		s, err := security.LoadTrust()
		if err == nil {
			s.AddRootWithConfigHash(root, security.HashMuxonToml(root))
			err = security.SaveTrust(s)
		}
		m.opt.Trusted = true
		if err != nil {
			m.appendLine("[error] trusted for this session only; could not save trust: " + err.Error())
		} else {
			m.appendLine("[harness] trusted " + shortPath(root))
		}
	case "trust_once":
		m.opt.Trusted = true
		m.appendLine("[harness] trusted " + shortPath(m.opt.Workspace) + " for this session")
	case "dont_trust":
		m.pendingGoal = ""
		m.appendLine("Workspace not trusted · goals need trust · /trust to change")
		return nil
	default:
		return nil
	}
	if m.pendingGoal == "" {
		return nil
	}
	goal := m.pendingGoal
	m.pendingGoal = ""
	return m.runGoal(goal)
}

const continueGoal = "Continue the previous task from where you left off."

func (m *Model) runGoal(goal string) tea.Cmd {
	m.state = director.StateActing
	force := m.forceEscalate
	m.forceEscalate = false
	m.input.Blur()
	return m.startGoal(goal, force)
}

func (m Model) submit(line string) (tea.Model, tea.Cmd) {
	if slash, ok := ParseSlash(line); ok {
		switch slash.Name {
		case "help":
			m.appendLine(HelpText())
		case "clear":
			m.transcript = nil
			m.history = nil
			m.syncViewport()
		case "providers":
			cmd := m.providersCommand(slash.Rest)
			return m, cmd
		case "model":
			cmd := m.modelCommand(slash.Rest)
			return m, cmd
		case "abort":
			m.appendLine("cancelled")
			m.lastOutcome = director.Outcome{State: director.StateCancelled, Reason: "user_cancel", ExitCode: 130}
			m.state = director.StateCancelled
			m.cancelRun()
		case "exit":
			m.cancelRun()
			return m, func() tea.Msg { return tea.Quit() }
		case "undo":
			if m.lastPatch == "" {
				m.appendLine("nothing to undo")
				break
			}
			if err := director.RestoreNewest(m.opt.Workspace, m.opt.SessionID, m.lastPatch); err != nil {
				m.appendLine("undo failed")
			} else {
				m.appendLine("undid " + m.lastPatch)
			}
		case "escalate":
			m.forceEscalate = true
			m.appendLine("escalate on next goal")
		case "continue":
			if m.lastOutcome.State != director.StatePaused && m.lastOutcome.State != director.StateFailed {
				m.appendLine("nothing to continue")
				break
			}
			if m.opt.Provider == nil {
				m.appendLine("No provider connected · run /providers")
				break
			}
			m.appendLine("> " + continueGoal)
			return m, m.runGoal(continueGoal)
		case "cost":
			m.appendLine(m.statusText())
		case "trust":
			m.modal = NewTrustModal(m.opt.Workspace)
		}
		return m, nil
	}
	if line == "" {
		return m, nil
	}
	if m.asking != nil {
		m.answerQuestion(line)
		return m, nil
	}
	if m.opt.Provider == nil {
		m.appendLine("> " + line)
		if m.connecting != "" {
			m.appendLine("Still connecting to " + m.connecting + " · try again in a moment")
		} else {
			m.appendLine("No provider connected · run /providers")
		}
		return m, nil
	}
	m.appendLine("> " + line)
	if !m.trusted() {
		m.pendingGoal = line
		m.modal = NewTrustModal(m.opt.Workspace)
		return m, nil
	}
	cmd := m.runGoal(line)
	return m, cmd
}

func (m *Model) startGoal(goal string, force bool) tea.Cmd {
	opt := m.opt
	trusted := m.trusted()
	prior := append([]provider.Message(nil), m.history...)
	events := make(chan tea.Msg, 32)
	m.events = events
	go func() {
		defer close(events)
		ctx, cancel := context.WithCancel(context.Background())
		m.run.mu.Lock()
		m.run.cancel = cancel
		m.run.mu.Unlock()
		defer cancel()
		out, err := director.Run(ctx, director.Config{
			Goal:             goal,
			Workspace:        opt.Workspace,
			Trusted:          trusted,
			MuxonVersion:     opt.Version,
			Provider:         opt.Provider,
			Manifest:         opt.Manifest,
			SessionID:        opt.SessionID,
			ForceEscalate:    force,
			Interactive:      true,
			RegisterAllTools: true,
			History:          prior,
			OnEvent: func(ev observability.Event) {
				events <- liveEvent{ev: ev}
			},
			OnAssistant: func(text string) {
				events <- assistantLine{text: text}
			},
			OnTool: func(act director.ToolActivity) {
				events <- toolMsg{act: act}
			},
			AskUser: func(ctx context.Context, question string, choices []string) (string, error) {
				reply := make(chan string, 1)
				select {
				case events <- askUserMsg{question: question, choices: choices, reply: reply}:
				case <-ctx.Done():
					return "", ctx.Err()
				}
				select {
				case answer := <-reply:
					return answer, nil
				case <-ctx.Done():
					return "", ctx.Err()
				}
			},
		})
		events <- goalDone{out: out, err: err}
	}()
	return listen(events)
}

func listen(ch <-chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) applyEvent(ev observability.Event) {
	switch ev.Type {
	case "state.transition":
		if ev.To != "" {
			m.state = ev.To
		}
	case "goal.start":
		if ev.TaskType != "" {
			m.taskType = ev.TaskType
		}
	case "tool.call":
		m.rounds++
		if ev.Name == "patch" {
			m.lastPatch = "sum.go"
		}
	case "verify.run":
		m.verifies++
	case "model.response":
		m.inputTokens += ev.InputTokens
		m.outputTokens += ev.OutputTokens
		m.cachedTokens += ev.CachedTokens
		m.costUSD += ev.CostUSD
		if m.costUSD > 0 {
			m.cost = fmt.Sprintf("$%.2f", m.costUSD)
		}
	}
	if line, ok := FormatEvent(ev); ok {
		m.appendLine(line)
	}
}

func (m Model) inputBusy() bool {
	switch m.state {
	case director.StateActing, director.StateTooling, director.StateVerifying:
		return true
	default:
		return false
	}
}

func (m *Model) cancelRun() {
	if m.run == nil {
		return
	}
	m.run.mu.Lock()
	if m.run.cancel != nil {
		m.run.cancel()
	}
	m.run.mu.Unlock()
}

// Interrupt cancels the in-flight goal. A second call within 2s aborts.
func (m *Model) Interrupt(now time.Time) {
	if !m.lastInterrupt.IsZero() && now.Sub(m.lastInterrupt) < 2*time.Second {
		m.lastOutcome = director.Outcome{State: director.StateCancelled, Reason: "user_cancel", ExitCode: 130}
		m.state = director.StateCancelled
		m.appendLine("cancelled")
	}
	m.lastInterrupt = now
	m.cancelRun()
}

func (m *Model) appendLine(line string) {
	m.transcript = append(m.transcript, entry{line: line})
	m.syncViewport()
}

func (m *Model) applyTool(act director.ToolActivity) {
	if hiddenTools[act.Name] {
		return
	}
	if act.Done {
		for i := len(m.transcript) - 1; i >= 0; i-- {
			if tc := m.transcript[i].tool; tc != nil && tc.name == act.Name && !tc.done {
				tc.finish(act)
				m.syncViewport()
				return
			}
		}
	}
	tc := newToolCall(act)
	if act.Done {
		tc.finish(act)
	}
	m.transcript = append(m.transcript, entry{tool: tc})
	m.syncViewport()
}

func (m Model) transcriptText() string {
	lines := make([]string, len(m.transcript))
	for i, e := range m.transcript {
		lines[i] = e.line
		if e.tool != nil {
			lines[i] = e.tool.title()
		}
	}
	return strings.Join(lines, "\n")
}

const (
	sideMargin  = 2
	inputLines  = 3
	chromeLines = inputLines + 2 + 1 + 1 + 1
)

func (m *Model) syncViewport() {
	m.viewport.SetContent(renderTranscript(m.transcript, m.viewport.Width, m.expandTools))
	m.viewport.GotoBottom()
}

func (m Model) contentWidth() int {
	return max(m.width-2*sideMargin, 20)
}

func (m *Model) layout() {
	width := m.contentWidth()
	height := max(m.height, 12)
	m.viewport.Width = width
	m.viewport.Height = max(height-chromeLines-m.suggestionRows(), 3)
	m.input.SetWidth(width - 4)
	m.input.SetHeight(inputLines)
	m.keyInput.Width = width - 6
	m.syncViewport()
}

func (m Model) statusText() string {
	task := m.taskType
	if task == "" {
		task = "—"
	}
	return fmt.Sprintf("goal %s · state %s · rounds %d · verify %d · %s · %s in (%s cached) / %s out",
		task, m.state, m.rounds, m.verifies, m.cost,
		formatTokens(m.inputTokens), formatTokens(m.cachedTokens), formatTokens(m.outputTokens))
}

func shortError(err error) string {
	return truncateRight(strings.Join(strings.Fields(err.Error()), " "), 240)
}

func formatTokens(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// View implements tea.Model.
func (m Model) View() string {
	width := m.contentWidth()
	body := m.viewport.View()
	switch {
	case m.pickerOpen:
		body = lipgloss.Place(width, m.viewport.Height, lipgloss.Center, lipgloss.Center, m.pickerView())
	case m.modelPicker:
		body = lipgloss.Place(width, m.viewport.Height, lipgloss.Center, lipgloss.Center, m.modelPickerView())
	case m.modal != nil && !m.modal.Done:
		body = lipgloss.Place(width, m.viewport.Height, lipgloss.Center, lipgloss.Center, m.modal.View())
	case len(m.transcript) == 0:
		body = lipgloss.Place(width, m.viewport.Height, lipgloss.Center, lipgloss.Center, m.welcomeView())
	}
	pad := lipgloss.NewStyle().PaddingLeft(sideMargin)
	parts := []string{
		"",
		pad.Render(body),
		pad.Render(m.activityView()),
	}
	if menu := m.suggestionView(width); menu != "" {
		parts = append(parts, pad.Render(menu))
	}
	parts = append(parts, pad.Render(m.inputView(width)), pad.Render(m.footerView(width)))
	return strings.Join(parts, "\n")
}

func (m Model) welcomeView() string {
	parts := []string{
		accentStyle.Bold(true).Render(logo),
		mutedStyle.Render(displayVersion(m.opt.Version)),
		"",
		commandList(textStyle.Bold(true), mutedStyle),
	}
	switch {
	case m.connecting != "":
		parts = append(parts, "", accentStyle.Render("Connecting to "+m.connecting+"…"))
	case m.opt.Provider == nil:
		parts = append(parts, "", warnStyle.Render("No provider connected · run /providers"))
	}
	return lipgloss.JoinVertical(lipgloss.Center, parts...)
}

func (m Model) activityView() string {
	if m.connecting != "" {
		return accentStyle.Render("● ") + textStyle.Render("connecting to "+m.connecting+"…")
	}
	if m.asking != nil {
		return accentStyle.Render("? ") + textStyle.Render("waiting for your answer") + mutedStyle.Render("   enter reply · ctrl+c cancel goal")
	}
	if m.inputBusy() {
		return accentStyle.Render("● ") + textStyle.Render(m.state+"…") + mutedStyle.Render("   ctrl+c interrupt")
	}
	if m.modelsLoading && wantsModelList(m.input.Value()) {
		return mutedStyle.Render("loading models…")
	}
	if m.suggestionRows() > 0 {
		return mutedStyle.Render("↑↓ select · tab complete · enter run · esc close")
	}
	return mutedStyle.Render("enter send · alt+enter newline · ctrl+o tool details · /help commands")
}

func (m Model) inputView(width int) string {
	border := colorAccent
	if m.inputBusy() {
		border = colorBorder
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1).
		Width(width - 2)
	if m.awaitingKey != "" {
		title := m.awaitingKey
		if c, ok := choiceByArg(m.awaitingKey); ok {
			title = c.Title
		}
		label := accentStyle.Render("API key · "+title) + mutedStyle.Render("   enter save · esc cancel")
		return box.Render(label + "\n" + m.keyInput.View())
	}
	return box.Render(m.input.View())
}

func (m Model) footerView(width int) string {
	brand := accentStyle.Bold(true).Render("muxon") + " " + mutedStyle.Render(displayVersion(m.opt.Version)) + "  "
	model := m.opt.WorkerModel
	if model == "" {
		model = "no model"
	}
	right := textStyle.Render(model)
	if m.opt.Provider != nil && m.opt.ProviderID != "" {
		right += mutedStyle.Render(" · " + m.opt.ProviderID)
	}
	right += mutedStyle.Render(" · " + m.opt.Permission + " · " + m.cost)
	if !m.opt.Trusted {
		right += mutedStyle.Render(" · ") + warnStyle.Render("untrusted")
	}
	room := width - lipgloss.Width(brand) - lipgloss.Width(right) - 2
	if room < 8 {
		return brand + textStyle.Render(model)
	}
	left := brand + mutedStyle.Render(truncateLeft(shortPath(m.opt.Workspace), room))
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	return left + strings.Repeat(" ", max(gap, 2)) + right
}

// Run starts the interactive program.
func Run(opt Options) error {
	p := tea.NewProgram(NewModel(opt), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
