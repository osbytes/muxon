package tui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/openai"
)

const modelPickerRows = 8

type modelsLoaded struct {
	names []string
	err   error
	epoch int
}

func wantsModelList(raw string) bool {
	token, _, _ := strings.Cut(strings.TrimPrefix(raw, "/"), " ")
	token = strings.ToLower(token)
	return token == "model" || token == "models"
}

func (m *Model) noteInputChange() tea.Cmd {
	m.completeDismissed = false
	m.completeIndex = 0
	m.layout()
	return m.ensureModels()
}

func (m *Model) ensureModels() tea.Cmd {
	if !wantsModelList(m.input.Value()) && !m.modelPicker {
		return nil
	}
	return m.loadModels()
}

func (m *Model) loadModels() tea.Cmd {
	if m.modelsLoading || m.modelsAuthoritative {
		return nil
	}
	client, ok := m.opt.Provider.(*openai.Client)
	if !ok {
		if m.opt.WorkerModel != "" && len(m.modelChoices) == 0 {
			m.modelChoices = []string{m.opt.WorkerModel}
		}
		return nil
	}
	m.modelsLoading = true
	epoch := m.modelEpoch
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		names, err := client.ListModels(ctx)
		return modelsLoaded{names: names, err: err, epoch: epoch}
	}
}

func (m *Model) finishModels(msg modelsLoaded) {
	if msg.epoch != m.modelEpoch {
		return
	}
	m.modelsLoading = false
	if msg.err != nil {
		m.appendLine("[error] could not list models: " + msg.err.Error())
		if m.opt.WorkerModel != "" {
			m.modelChoices = []string{m.opt.WorkerModel}
		}
		return
	}
	m.modelsAuthoritative = true
	m.modelChoices = orderModels(msg.names, m.opt.WorkerModel)
	m.modelPickerIndex = 0
}

func orderModels(names []string, current string) []string {
	if current == "" {
		return names
	}
	out := []string{current}
	for _, name := range names {
		if name != current {
			out = append(out, name)
		}
	}
	return out
}

func (m *Model) modelCommand(rest string) tea.Cmd {
	if m.opt.Provider == nil {
		m.appendLine("[error] no provider connected · run /providers")
		return nil
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		m.modelPicker = true
		m.modelPickerIndex = 0
		for i, name := range m.modelChoices {
			if name == m.opt.WorkerModel {
				m.modelPickerIndex = i
				break
			}
		}
		return m.loadModels()
	}
	m.selectModel(rest)
	return nil
}

func (m *Model) selectModel(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	if matched, ok := matchModelName(m.modelChoices, name); ok {
		name = matched
	} else if m.modelsAuthoritative {
		m.appendLine("[error] unknown model " + name)
		return
	}
	if err := m.applyWorkerModel(name); err != nil {
		m.appendLine("[error] " + err.Error())
		return
	}
	m.modelPicker = false
	_ = config.SaveLastModel(name)
	m.appendLine("[harness] model " + name)
}

func (m *Model) applyWorkerModel(name string) error {
	if m.opt.Manifest == nil {
		embedded, err := config.LoadEmbeddedManifest()
		if err != nil {
			return err
		}
		m.opt.Manifest = cloneManifest(embedded)
	}
	providerID := m.opt.ProviderID
	if providerID == "" {
		providerID = "deepinfra"
	}
	if _, ok := m.opt.Manifest.Providers[providerID]; !ok && m.choice.ID == providerID {
		installChoice(m.opt.Manifest, m.choice)
	}
	installWorkerModel(m.opt.Manifest, providerID, name)
	m.opt.WorkerModel = name
	return nil
}

// installWorkerModel points the worker role at modelName.
// Capabilities are copied from the current worker on the same provider so the
// new model keeps that endpoint's tool protocol.
func installWorkerModel(manifest *config.Manifest, providerID, modelName string) {
	if manifest.Roles == nil {
		manifest.Roles = map[string]config.RoleDef{}
	}
	if manifest.Providers == nil {
		manifest.Providers = map[string]config.ProviderDef{}
	}
	if _, ok := manifest.Providers[providerID]; !ok {
		manifest.Providers[providerID] = config.ProviderDef{API: "openai_chat"}
	}
	caps := config.Capabilities{}
	if cur, err := manifest.ResolveRole("worker"); err == nil && cur.Provider == providerID {
		caps = cur.Capabilities
	}
	config.ApplyDefaults(&caps)
	for i := range manifest.Models {
		if manifest.Models[i].Provider == providerID && manifest.Models[i].Model == modelName {
			manifest.Roles["worker"] = config.RoleDef{ModelID: manifest.Models[i].ID}
			return
		}
	}
	id := providerID + "/" + modelName
	if _, err := manifest.ResolveModel(id); err == nil {
		id += "-worker"
	}
	manifest.Models = append(manifest.Models, config.ModelDef{
		ID:           id,
		Provider:     providerID,
		Model:        modelName,
		Capabilities: caps,
	})
	manifest.Roles["worker"] = config.RoleDef{ModelID: id}
}

func matchModelName(choices []string, query string) (string, bool) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return "", false
	}
	var hits []string
	for _, name := range choices {
		if strings.EqualFold(name, query) {
			return name, true
		}
		if strings.Contains(strings.ToLower(name), query) {
			hits = append(hits, name)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	return "", false
}

func (m Model) modelSuggestions(raw string) ([]suggestion, bool) {
	if !strings.HasPrefix(raw, "/") || strings.Contains(raw, "\n") {
		return nil, false
	}
	token, rest, hasSpace := strings.Cut(strings.TrimPrefix(raw, "/"), " ")
	token = strings.ToLower(token)
	if !hasSpace || (token != "model" && token != "models") {
		return nil, false
	}
	if strings.Contains(rest, " ") {
		return nil, true
	}
	query := strings.ToLower(rest)
	var out []suggestion
	for _, name := range m.modelChoices {
		if query != "" && !strings.Contains(strings.ToLower(name), query) {
			continue
		}
		item := suggestion{label: name, insert: "/model " + name}
		if name == m.opt.WorkerModel {
			item.desc = "current"
		}
		out = append(out, item)
	}
	return out, true
}

func (m Model) updateModelPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c", "q":
		m.modelPicker = false
		return m, nil
	case "up", "k", "shift+tab":
		if len(m.modelChoices) == 0 {
			return m, nil
		}
		m.modelPickerIndex = (m.modelPickerIndex + len(m.modelChoices) - 1) % len(m.modelChoices)
		return m, nil
	case "down", "j", "tab":
		if len(m.modelChoices) == 0 {
			return m, nil
		}
		m.modelPickerIndex = (m.modelPickerIndex + 1) % len(m.modelChoices)
		return m, nil
	case "enter":
		if m.modelsLoading || len(m.modelChoices) == 0 {
			return m, nil
		}
		idx := m.modelPickerIndex
		if idx < 0 || idx >= len(m.modelChoices) {
			idx = 0
		}
		m.selectModel(m.modelChoices[idx])
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) modelPickerView() string {
	title := "Switch model"
	if m.choice.Title != "" {
		title += " · " + m.choice.Title
	}
	rows := []string{textStyle.Bold(true).Render(title), ""}
	switch {
	case m.modelsLoading && len(m.modelChoices) == 0:
		rows = append(rows, mutedStyle.Render("Loading models…"))
	case len(m.modelChoices) == 0:
		rows = append(rows, warnStyle.Render("No models returned"))
	default:
		start, selected := sliceWindow(len(m.modelChoices), m.modelPickerIndex, modelPickerRows)
		for i, name := range m.modelChoices[start:min(start+modelPickerRows, len(m.modelChoices))] {
			cursor := "  "
			label := textStyle.Render(name)
			if i == selected {
				cursor = accentStyle.Render("› ")
				label = accentStyle.Bold(true).Render(name)
			}
			line := cursor + label
			if name == m.opt.WorkerModel {
				line += mutedStyle.Render("  current")
			}
			rows = append(rows, line)
		}
	}
	rows = append(rows, "", mutedStyle.Render("↑↓ select · enter switch · esc close"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 3).
		Render(strings.Join(rows, "\n"))
}

func sliceWindow(n, idx, limit int) (start, selected int) {
	if n == 0 {
		return 0, 0
	}
	if idx < 0 || idx >= n {
		idx = 0
	}
	if n <= limit {
		return 0, idx
	}
	start = idx - limit/2
	if start < 0 {
		start = 0
	}
	if start+limit > n {
		start = n - limit
	}
	return start, idx - start
}
