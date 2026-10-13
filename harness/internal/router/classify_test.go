package router

import (
	"testing"
)

func TestHeuristicQuestionGoal(t *testing.T) {
	c := HeuristicClassify("where is X configured?")
	if c.TaskType != "question" {
		t.Fatalf("task_type %q", c.TaskType)
	}
}

func TestHeuristicChangeGoal(t *testing.T) {
	c := HeuristicClassify("fix the failing test")
	if c.TaskType != "change" {
		t.Fatalf("task_type %q", c.TaskType)
	}
}
