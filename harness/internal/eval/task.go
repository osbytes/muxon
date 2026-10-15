package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Check is the hidden grader. Files are copied in only after the agent exits.
type Check struct {
	Cmd   []string `json:"cmd,omitempty"`
	Files string   `json:"files,omitempty"`
}

// Task is a self-graded eval fixture descriptor.
type Task struct {
	ID        string   `json:"id"`
	Prompt    string   `json:"prompt"`
	Fixture   string   `json:"fixture,omitempty"`
	VerifyCmd []string `json:"verify_cmd,omitempty"`
	Directory string   `json:"directory,omitempty"`
	Check     Check    `json:"check,omitempty"`
	TimeoutS  int      `json:"timeout_s,omitempty"`
	Tags      []string `json:"tags,omitempty"`
}

// ResultRow is one line in results.jsonl.
type ResultRow struct {
	RunID              string            `json:"run_id,omitempty"`
	TaskID             string            `json:"task_id"`
	Trial              int               `json:"trial"`
	Reward             int               `json:"reward"`
	ModelID            string            `json:"model_id"`
	Provider           string            `json:"provider,omitempty"`
	HarnessSHA         string            `json:"harness_sha,omitempty"`
	ManifestHash       string            `json:"manifest_hash,omitempty"`
	InputTokens        int               `json:"input_tokens"`
	OutputTokens       int               `json:"output_tokens"`
	CachedTokens       int               `json:"cached_tokens"`
	CostUSD            float64           `json:"cost_usd"`
	ToolRounds         int               `json:"tool_rounds"`
	Escalations        int               `json:"escalations"`
	MalformedToolCalls int               `json:"malformed_tool_calls"`
	SalvagedToolCalls  int               `json:"salvaged_tool_calls,omitempty"`
	PatchApplies       int               `json:"patch_applies,omitempty"`
	WellFormedPatches  int               `json:"well_formed_patches,omitempty"`
	LoopsDetected      int               `json:"loops_detected"`
	TimeoutHit         bool              `json:"timeout_hit,omitempty"`
	Reason             string            `json:"reason,omitempty"`
	Overrides          map[string]string `json:"overrides,omitempty"`
	WallS              float64           `json:"wall_s"`
	RecordedAt         time.Time         `json:"recorded_at"`
}

// PreparedTask is a task plus the directory its check files are relative to.
type PreparedTask struct {
	Task    Task
	TaskDir string
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
	if t.ID == "" {
		return Task{}, fmt.Errorf("task %s: missing id", path)
	}
	if t.Prompt == "" {
		return Task{}, fmt.Errorf("task %s: missing prompt", path)
	}
	return t, nil
}

// LoadTasks reads every JSON task matched by pattern.
func LoadTasks(pattern string) ([]PreparedTask, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no tasks match %s", pattern)
	}
	var tasks []PreparedTask
	for _, match := range matches {
		task, err := LoadPrepared(match)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// LoadPrepared reads a task and remembers its directory.
func LoadPrepared(path string) (PreparedTask, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return PreparedTask{}, err
	}
	t, err := LoadTask(abs)
	if err != nil {
		return PreparedTask{}, err
	}
	return PreparedTask{Task: t, TaskDir: filepath.Dir(abs)}, nil
}

// AppendResult appends one eval row to a results.jsonl file.
func AppendResult(path string, row ResultRow) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
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
