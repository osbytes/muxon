package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Task is a self-graded eval fixture descriptor.
type Task struct {
	ID        string   `json:"id"`
	Prompt    string   `json:"prompt"`
	Fixture   string   `json:"fixture"`
	VerifyCmd []string `json:"verify_cmd"`
	Directory string   `json:"directory"`
}

// ResultRow is one line in results.jsonl.
type ResultRow struct {
	TaskID             string    `json:"task_id"`
	Trial              int       `json:"trial"`
	Reward             int       `json:"reward"`
	ModelID            string    `json:"model_id"`
	InputTokens        int       `json:"input_tokens"`
	OutputTokens       int       `json:"output_tokens"`
	CachedTokens       int       `json:"cached_tokens"`
	CostUSD            float64   `json:"cost_usd"`
	ToolRounds         int       `json:"tool_rounds"`
	Escalations        int       `json:"escalations"`
	MalformedToolCalls int       `json:"malformed_tool_calls"`
	LoopsDetected      int       `json:"loops_detected"`
	WallS              float64   `json:"wall_s"`
	RecordedAt         time.Time `json:"recorded_at"`
}

// LoadTask reads a task JSON file.
func LoadTask(path string) (Task, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Task{}, err
	}
	var t Task
	if err := json.Unmarshal(b, &t); err != nil {
		return Task{}, err
	}
	return t, nil
}

// AppendResult appends one eval row under workspace/.thee/eval/results.jsonl.
func AppendResult(workspaceRoot string, row ResultRow) error {
	dir := filepath.Join(workspaceRoot, ".thee", "eval")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "results.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if row.RecordedAt.IsZero() {
		row.RecordedAt = time.Now().UTC()
	}
	b, err := json.Marshal(row)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}
