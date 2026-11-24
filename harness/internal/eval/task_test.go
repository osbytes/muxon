package eval

import (
	"path/filepath"
	"testing"
)

func TestRepoEvalTasksResolve(t *testing.T) {
	tasks, err := LoadTasks(filepath.Join("..", "..", "..", "evals", "tasks", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, pt := range tasks {
		t.Run(pt.Task.ID, func(t *testing.T) {
			if _, err := taskSource(pt.Task, TrialOptions{TaskDir: pt.TaskDir}); err != nil {
				t.Errorf("fixture: %v", err)
			}
			if pt.Task.Check.Files == "" {
				return
			}
			if _, err := findDir(pt.Task.Check.Files, []string{pt.TaskDir, filepath.Join(pt.TaskDir, "..")}); err != nil {
				t.Errorf("check files: %v", err)
			}
		})
	}
}
