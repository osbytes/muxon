package contextengine

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/index"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

const wrapUpNudge = "You are near the tool round limit. Finish the current change, then call finish with a summary of what is done and what remains."

// TurnInput builds one model turn.
type TurnInput struct {
	Goal          string
	History       []provider.Message
	ToolProtocol  string
	PromptVariant string
	ToolSpecs     []tools.Spec
	WorkspaceRoot string
	Trusted       bool
	Phase         string
	ContextWindow int
	Estimator     *Estimator
	Policy        *security.Policy
	Ignore        *security.IgnoreRules
	ToolRounds    int
	MaxToolRounds int
	Parallel      bool
}

// TurnOutput is assembled messages and optional native tools.
type TurnOutput struct {
	Messages             []provider.Message
	Tools                []provider.ToolDef
	PrefixHash           string
	EstimatedInputTokens int
}

// BuildTurn assembles the 12-section layout into provider messages.
func BuildTurn(in TurnInput) TurnOutput { //nolint:gocyclo // linear assembly of prompt sections; splitting hides the order
	if in.Estimator == nil {
		in.Estimator = NewEstimator()
	}
	if in.Phase == "" {
		in.Phase = tools.PhaseEditing
	}
	variant := in.PromptVariant
	if variant == "" {
		variant = VariantCompact
	}
	rendered, err := RenderVariant(variant, in.ToolSpecs, in.Parallel)
	if err != nil {
		rendered = RenderedPrompt{SystemRules: "You are thee.", ToolCatalog: "Tools available."}
	}

	instructions, _ := LoadProjectInstructions(in.WorkspaceRoot, in.Trusted)
	var repoText string
	if in.WorkspaceRoot != "" {
		pol := in.Policy
		if pol == nil {
			pol = security.NewPolicy()
		}
		ign := in.Ignore
		if ign == nil {
			ign, _ = security.LoadIgnore(in.WorkspaceRoot, pol)
		}
		syms, _ := index.LoadOrBuild(in.WorkspaceRoot, in.Goal, ign, pol)
		repoText = index.FormatSymbols(syms, 0)
	}

	budget := TurnInputBudget(in.ContextWindow)
	pb := PrefixBudget{TurnInputBudget: budget}
	instructions, repoText, note := pb.Apply(instructions, repoText)

	sec3 := "Skills: (none)\n"
	sec4 := instructions
	if note != "" {
		sec4 = note + sec4
	}
	sec5 := repoText

	prefixHash := hashPrefixSections(rendered.SystemRules, rendered.ToolCatalog, sec3, sec4, sec5)

	system := strings.Join([]string{
		rendered.SystemRules,
		rendered.ToolCatalog,
		sec3,
		sec4,
		sec5,
	}, "\n\n")

	history := CompactHistory(in.History, budget, in.ContextWindow, in.Estimator)

	phaseLine := "Active: " + activeToolsForPhase(in.Phase, in.ToolSpecs) + "\n"
	if in.MaxToolRounds > 0 {
		phaseLine += fmt.Sprintf("Tool rounds: %d/%d\n", in.ToolRounds, in.MaxToolRounds)
		if in.ToolRounds >= wrapUpAt(in.MaxToolRounds) {
			phaseLine += wrapUpNudge + "\n"
		}
	}
	var step2 string
	if in.ToolProtocol == "two_step" {
		step2 = "Step-2: emit flat JSON tool call for one tool.\n"
		if rendered.Example != "" {
			step2 += rendered.Example + "\n"
		}
	}

	msgs := []provider.Message{{Role: "system", Content: system}}
	msgs = append(msgs, history...)
	tail := phaseLine + step2
	if tail != "" {
		msgs = append(msgs, provider.Message{Role: "user", Content: tail})
	}
	if in.Goal != "" && !historyEndsWithGoal(history, in.Goal) {
		msgs = append(msgs, provider.Message{Role: "user", Content: in.Goal})
	}

	out := TurnOutput{
		Messages:             msgs,
		PrefixHash:           prefixHash,
		EstimatedInputTokens: in.Estimator.EstimateMessages(msgs),
	}
	if in.ToolProtocol == "native" {
		for _, s := range in.ToolSpecs {
			out.Tools = append(out.Tools, provider.ToolDef{
				Name:        s.Name,
				Description: s.Description,
				Parameters:  s.Parameters,
			})
		}
	}
	return out
}

// TurnInputBudget is the prompt budget for a model window: the window minus a
// 10% margin and reserved output tokens.
func TurnInputBudget(contextWindow int) int {
	if contextWindow <= 0 {
		contextWindow = 32768
	}
	margin := int(float64(contextWindow) * 0.1)
	return contextWindow - margin - 4096
}

func hashPrefixSections(s1, s2, s3, s4, s5 string) string {
	h := sha256.Sum256([]byte(strings.Join([]string{s1, s2, s3, s4, s5}, "\x1e")))
	return hex.EncodeToString(h[:16])
}

func historyEndsWithGoal(history []provider.Message, goal string) bool {
	if len(history) == 0 {
		return false
	}
	last := history[len(history)-1]
	return last.Role == "user" && last.Content == goal
}

func activeToolsForPhase(phase string, specs []tools.Spec) string {
	if phase == "" {
		phase = tools.PhaseEditing
	}
	names := make([]string, 0, len(specs))
	for _, s := range specs {
		if tools.CheckPhase(phase, s.Name) == nil {
			names = append(names, s.Name)
		}
	}
	return strings.Join(names, ", ")
}

// wrapUpAt is the completed-round count where the model is told to finish.
// That is the earlier of 80% of max and the last five rounds.
func wrapUpAt(max int) int {
	if max <= 0 {
		return 0
	}
	atPct := max * 80 / 100
	last := max - 5
	if last < 0 {
		last = 0
	}
	if last < atPct {
		return last
	}
	return atPct
}

// PrefixHashForSession returns hash for unchanged prefix fields (tests).
func PrefixHashForSession(in TurnInput) string {
	out := BuildTurn(in)
	return out.PrefixHash
}
