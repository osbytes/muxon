package contextengine

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/assets"
	"github.com/PLACEHOLDER/thee/harness/internal/tools"
)

const (
	VariantCompact  = "compact"
	VariantStandard = "standard"
)

// RenderedPrompt is the stable prefix text for sections 1–2 (and compact example in section 2 tail).
type RenderedPrompt struct {
	SystemRules string
	ToolCatalog string
	Example     string
}

// RenderVariant loads embedded templates and builds the tool catalog.
func RenderVariant(variant string, specs []tools.Spec) (RenderedPrompt, error) {
	if variant == "" {
		variant = VariantCompact
	}
	rulesPath := fmt.Sprintf("prompts/%s/rules.tmpl", variant)
	rules, err := fs.ReadFile(assets.PromptFS, rulesPath)
	if err != nil {
		return RenderedPrompt{}, err
	}
	out := RenderedPrompt{
		SystemRules: strings.TrimSpace(string(rules)),
		ToolCatalog: renderCatalog(specs, variant),
	}
	if variant == VariantCompact {
		ex, err := fs.ReadFile(assets.PromptFS, "prompts/compact/example_two_step.tmpl")
		if err != nil {
			return RenderedPrompt{}, err
		}
		out.Example = strings.TrimSpace(string(ex))
	}
	return out, nil
}

func renderCatalog(specs []tools.Spec, variant string) string {
	var b strings.Builder
	b.WriteString("Tool catalog:\n")
	for _, s := range specs {
		line := s.Name + ": " + s.Description
		if variant == VariantStandard {
			line += " — use only when needed for the goal"
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	b.WriteString("Protocol: call one tool per turn with valid JSON arguments.\n")
	return b.String()
}

// SystemBlock returns sections 1–2 combined for golden hashing.
func SystemBlock(r RenderedPrompt) string {
	var b strings.Builder
	b.WriteString(r.SystemRules)
	b.WriteByte('\n')
	b.WriteString(r.ToolCatalog)
	if r.Example != "" {
		b.WriteByte('\n')
		b.WriteString(r.Example)
	}
	return b.String()
}
