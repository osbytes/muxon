package tui

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/openai"
	"github.com/PLACEHOLDER/thee/harness/internal/router"
)

var openCodeGoBaseURL = "https://opencode.ai/zen/go/v1"

// ProviderChoice is one entry in /providers.
type ProviderChoice struct {
	ID        string
	Title     string
	BaseURL   string
	APIKeyEnv string
	ModelID   string
	ModelName string
}

// ProviderChoices lists providers the session can connect, OpenCode Go first.
func ProviderChoices() []ProviderChoice {
	return []ProviderChoice{
		{
			ID:        "opencode-go",
			Title:     "OpenCode Go",
			BaseURL:   openCodeGoBaseURL,
			APIKeyEnv: "OPENCODE_API_KEY",
			ModelID:   "opencode-go-kimi",
			ModelName: "kimi-k2.7-code",
		},
		{
			ID:        "deepinfra",
			Title:     "DeepInfra",
			BaseURL:   "https://api.deepinfra.com/v1/openai",
			APIKeyEnv: "DEEPINFRA_API_KEY",
			ModelID:   "m0-worker",
			ModelName: "test/model",
		},
	}
}

type providerConnected struct {
	choice   ProviderChoice
	provider provider.Provider
	manifest *config.Manifest
	err      error
}

func choiceByArg(arg string) (ProviderChoice, bool) {
	arg = strings.TrimSpace(arg)
	choices := ProviderChoices()
	if n, err := strconv.Atoi(arg); err == nil && n >= 1 && n <= len(choices) {
		return choices[n-1], true
	}
	for _, c := range choices {
		if c.ID == arg || strings.EqualFold(c.Title, arg) {
			return c, true
		}
	}
	return ProviderChoice{}, false
}

func knownProviderKey(c ProviderChoice) string {
	if c.APIKeyEnv != "" {
		if v := os.Getenv(c.APIKeyEnv); v != "" {
			return v
		}
	}
	if v, err := config.ResolveProviderKey(c.APIKeyEnv, c.ID); err == nil && v != "" {
		return v
	}
	if v, err := config.OpenCodeAuthKey(c.ID); err == nil && v != "" {
		return v
	}
	return ""
}

func (m *Model) providersCommand(rest string) tea.Cmd {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		m.pickerOpen = true
		m.pickerIndex = 0
		return nil
	}
	choice, ok := choiceByArg(rest)
	if !ok {
		m.appendLine("[error] unknown provider " + rest + " · try /providers")
		return nil
	}
	return m.selectProvider(choice)
}

func (m *Model) selectProvider(c ProviderChoice) tea.Cmd {
	m.pickerOpen = false
	secret := knownProviderKey(c)
	if secret == "" {
		m.awaitingKey = c.ID
		m.keyInput.SetValue("")
		return m.keyInput.Focus()
	}
	return m.beginConnect(c, secret)
}

func (m Model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	choices := ProviderChoices()
	switch msg.String() {
	case "esc", "ctrl+c", "q":
		m.pickerOpen = false
		return m, nil
	case "up", "k", "shift+tab":
		m.pickerIndex = (m.pickerIndex + len(choices) - 1) % len(choices)
		return m, nil
	case "down", "j", "tab":
		m.pickerIndex = (m.pickerIndex + 1) % len(choices)
		return m, nil
	case "enter":
		cmd := m.selectProvider(choices[m.pickerIndex])
		return m, cmd
	}
	if c, ok := choiceByArg(msg.String()); ok {
		cmd := m.selectProvider(c)
		return m, cmd
	}
	return m, nil
}

func (m Model) pickerView() string {
	rows := []string{textStyle.Bold(true).Render("Connect a provider"), ""}
	for i, c := range ProviderChoices() {
		cursor := "  "
		title := textStyle.Render(c.Title)
		if i == m.pickerIndex {
			cursor = accentStyle.Render("› ")
			title = accentStyle.Bold(true).Render(c.Title)
		}
		status := mutedStyle.Render("needs key")
		if knownProviderKey(c) != "" {
			status = successStyle.Render("key found")
		}
		rows = append(rows, fmt.Sprintf("%s%s  %s  %s", cursor, padRight(title, 14), mutedStyle.Render(padRight(c.ModelName, 16)), status))
	}
	rows = append(rows, "", mutedStyle.Render("↑↓ select · enter connect · esc close"))
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorAccent).
		Padding(1, 3).
		Render(strings.Join(rows, "\n"))
}

// beginConnect saves the key, builds the client, and checks it with one request.
func (m *Model) beginConnect(c ProviderChoice, secret string) tea.Cmd {
	m.awaitingKey = ""
	m.keyInput.SetValue("")
	if secret == "" {
		m.appendLine("[error] an API key is required for " + c.Title)
		return nil
	}
	if err := config.StoreProviderKey(c.ID, secret); err != nil {
		m.appendLine("[error] save key: " + err.Error())
		return nil
	}
	if err := config.RememberProvider(c.ID, c.BaseURL, c.APIKeyEnv); err != nil {
		m.appendLine("[error] save provider: " + err.Error())
		return nil
	}
	base := m.opt.Manifest
	if base == nil {
		embedded, err := config.LoadEmbeddedManifest()
		if err != nil {
			m.appendLine("[error] " + err.Error())
			return nil
		}
		base = embedded
	}
	manifest := cloneManifest(base)
	installChoice(manifest, c)
	bind, err := router.ResolveWorker(manifest, "")
	if err != nil {
		m.appendLine("[error] " + err.Error())
		return nil
	}
	p, err := router.NewProvider(bind)
	if err != nil {
		m.appendLine("[error] " + err.Error())
		return nil
	}
	client, isOpenAI := p.(*openai.Client)
	if isOpenAI {
		if client.Headers == nil {
			client.Headers = map[string]string{}
		}
		client.Headers["User-Agent"] = "thee/" + displayVersion(m.opt.Version)
		client.Headers["x-opencode-session"] = m.opt.SessionID
	}
	m.connecting = c.Title
	return func() tea.Msg {
		if !isOpenAI {
			return providerConnected{choice: c, provider: p, manifest: manifest}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		err := client.Ping(ctx, c.ModelName)
		return providerConnected{choice: c, provider: p, manifest: manifest, err: err}
	}
}

func (m *Model) finishConnect(msg providerConnected) {
	m.connecting = ""
	if msg.err != nil {
		m.appendLine("[error] could not connect to " + msg.choice.Title + ": " + msg.err.Error())
		return
	}
	m.opt.Provider = msg.provider
	m.opt.ProviderID = msg.choice.ID
	m.opt.Manifest = msg.manifest
	m.choice = msg.choice
	m.modelEpoch++
	m.modelsLoading = false
	m.modelsAuthoritative = false
	m.modelChoices = nil
	m.modelPicker = false
	model := msg.choice.ModelName
	if config.LastProvider() == msg.choice.ID {
		if saved := config.LastModel(); saved != "" {
			model = saved
		}
	}
	if err := m.applyWorkerModel(model); err != nil {
		m.opt.WorkerModel = model
		m.appendLine("[error] " + err.Error())
	}
	_ = config.SaveLastProvider(msg.choice.ID)
	_ = config.SaveLastModel(m.opt.WorkerModel)
	m.appendLine("[connected] " + msg.choice.Title + " · " + m.opt.WorkerModel)
}

// autoConnect reconnects the provider saved by the last session when its key is available.
func (m *Model) autoConnect(id string) tea.Cmd {
	c, ok := choiceByArg(id)
	if !ok {
		return nil
	}
	secret := knownProviderKey(c)
	if secret == "" {
		return nil
	}
	return m.beginConnect(c, secret)
}

func cloneManifest(src *config.Manifest) *config.Manifest {
	out := *src
	out.Providers = make(map[string]config.ProviderDef, len(src.Providers))
	for k, v := range src.Providers {
		out.Providers[k] = v
	}
	out.Roles = make(map[string]config.RoleDef, len(src.Roles))
	for k, v := range src.Roles {
		out.Roles[k] = v
	}
	out.Models = append([]config.ModelDef(nil), src.Models...)
	return &out
}

func installChoice(manifest *config.Manifest, c ProviderChoice) {
	manifest.Providers[c.ID] = config.ProviderDef{
		API:       "openai_chat",
		BaseURL:   c.BaseURL,
		APIKeyEnv: c.APIKeyEnv,
	}
	found := false
	for i := range manifest.Models {
		if manifest.Models[i].ID == c.ModelID {
			manifest.Models[i].Provider = c.ID
			manifest.Models[i].Model = c.ModelName
			found = true
			break
		}
	}
	if !found {
		manifest.Models = append(manifest.Models, config.ModelDef{
			ID:       c.ModelID,
			Provider: c.ID,
			Model:    c.ModelName,
			Capabilities: config.Capabilities{
				ContextWindow:     131072,
				ToolProtocol:      "native",
				EditFormat:        "search_replace",
				PromptVariant:     "compact",
				ParallelToolCalls: true,
			},
		})
		config.ApplyDefaults(&manifest.Models[len(manifest.Models)-1].Capabilities)
	}
	manifest.Roles["worker"] = config.RoleDef{ModelID: c.ModelID}
}
