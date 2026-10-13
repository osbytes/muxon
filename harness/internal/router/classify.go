package router

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// Classification is the router task label for a goal.
type Classification struct {
	TaskType   string
	Plan       string
	Confidence float64
	Tags       []string
}

// ClassifyGoal labels the goal using router model or heuristics (D13).
func ClassifyGoal(ctx context.Context, goal string, m *config.Manifest, routerProv provider.Provider) Classification {
	workerID := ""
	if w, err := m.ResolveRole("worker"); err == nil {
		workerID = w.ID
	}
	routerRole, err := m.ResolveRole("router")
	if err != nil || routerProv == nil || routerRole.ID == workerID {
		return HeuristicClassify(goal)
	}
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_type":  map[string]string{"type": "string"},
			"plan":       map[string]string{"type": "string"},
			"confidence": map[string]string{"type": "number"},
			"tags":       map[string]interface{}{"type": "array", "items": map[string]string{"type": "string"}},
		},
		"required": []string{"task_type", "plan", "confidence"},
	}
	req := provider.Request{
		ModelID: routerRole.Model,
		Messages: []provider.Message{{
			Role:    "user",
			Content: "Classify this coding goal:\n" + goal,
		}},
		ResponseFormat: provider.ResponseFormat{
			Kind:   provider.FormatJSONSchema,
			Schema: schema,
			Name:   "classification",
		},
		MaxOutputTokens: 256,
	}
	resp, err := routerProv.Complete(ctx, req)
	if err != nil {
		return HeuristicClassify(goal)
	}
	var c Classification
	if err := json.Unmarshal([]byte(resp.Text), &c); err != nil {
		return HeuristicClassify(goal)
	}
	if c.TaskType == "" {
		c.TaskType = "change"
	}
	if c.Confidence == 0 {
		c.Confidence = 0.5
	}
	if len(c.Tags) == 0 {
		c.Tags = goalTags(goal)
	}
	return c
}

// HeuristicClassify applies goal-text rules when no router model is used.
func HeuristicClassify(goal string) Classification {
	trim := strings.TrimSpace(goal)
	lower := strings.ToLower(trim)
	task := "change"
	if strings.Contains(trim, "?") || strings.HasPrefix(lower, "where ") || strings.HasPrefix(lower, "what ") ||
		strings.HasPrefix(lower, "why ") || strings.HasPrefix(lower, "how ") || strings.HasPrefix(lower, "explain ") {
		task = "question"
	}
	return Classification{
		TaskType:   task,
		Plan:       "skip",
		Confidence: 0.9,
		Tags:       goalTags(goal),
	}
}

func goalTags(goal string) []string {
	var tags []string
	lower := strings.ToLower(goal)
	if strings.Contains(lower, "test") {
		tags = append(tags, "test-fix")
	}
	return tags
}

// LowConfidence reports whether classification should trigger escalation.
func (c Classification) LowConfidence() bool {
	return c.Confidence > 0 && c.Confidence < 0.5
}
