package toolcall

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/tools"
	"github.com/PLACEHOLDER/muxon/harness/internal/tools/schema"
)

var fenceRE = regexp.MustCompile("(?s)```(?:json)?\\s*([\\s\\S]*?)```")

// SalvageResult is a recovered tool call from text.
type SalvageResult struct {
	Call     provider.ToolCall
	Salvaged bool
	Step     string
}

// SalvageText finds and validates a tool call in free text.
func SalvageText(text string, toolName string, spec tools.Spec) (SalvageResult, error) {
	text = StripReasoning(text)
	raw := extractJSON(text)
	if raw == "" {
		return SalvageResult{}, errNoCall
	}
	raw = fixJSONSyntax(raw)
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return SalvageResult{}, err
	}
	coerceTypes(args, spec.Parameters)
	out, err := json.Marshal(args)
	if err != nil {
		return SalvageResult{}, err
	}
	if err := schema.ValidateRequired(out, spec.Parameters); err != nil {
		return SalvageResult{}, err
	}
	return SalvageResult{
		Call: provider.ToolCall{
			ID:        "salvaged",
			Name:      toolName,
			Arguments: out,
		},
		Salvaged: true,
		Step:     "syntax",
	}, nil
}

// SalvageStep1 picks tool name from {"tool":"..."} JSON.
func SalvageStep1(text string, allowed map[string]tools.Spec) (string, json.RawMessage, error) {
	text = StripReasoning(text)
	raw := extractJSON(text)
	if raw == "" {
		return "", nil, errNoCall
	}
	raw = fixJSONSyntax(raw)
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return "", nil, err
	}
	name, _ := m["tool"].(string)
	if name == "" {
		return "", nil, errNoCall
	}
	if _, ok := allowed[name]; !ok {
		return "", nil, errNoCall
	}
	return name, json.RawMessage(raw), nil
}

func extractJSON(text string) string {
	if m := fenceRE.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "{") {
		return text
	}
	i := strings.Index(text, "{")
	if i >= 0 {
		return balanceJSON(text[i:])
	}
	return ""
}

func balanceJSON(s string) string {
	depth := 0
	for i, r := range s {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[:i+1]
			}
		}
	}
	return s
}

func fixJSONSyntax(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, ",")
	re := regexp.MustCompile(`,\s*}`)
	s = re.ReplaceAllString(s, "}")
	re2 := regexp.MustCompile(`,\s*]`)
	s = re2.ReplaceAllString(s, "]")
	return s
}

func coerceTypes(args map[string]interface{}, params json.RawMessage) {
	var sm map[string]interface{}
	if err := json.Unmarshal(params, &sm); err != nil {
		return
	}
	props, _ := sm["properties"].(map[string]interface{})
	for k, v := range args {
		if sv, ok := v.(string); ok {
			p, _ := props[k].(map[string]interface{})
			if p != nil && p["type"] == "integer" {
				if n, err := strconv.Atoi(sv); err == nil {
					args[k] = n
				}
			}
		}
	}
}

var errNoCall = &salvageErr{"no tool call found"}

type salvageErr struct{ msg string }

func (e *salvageErr) Error() string { return e.msg }
