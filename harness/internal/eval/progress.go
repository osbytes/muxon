package eval

import (
	"fmt"
	"io"
	"sync"
)

// TrialProgress writes one line when a trial starts and one when it finishes.
type TrialProgress struct {
	w     io.Writer
	mu    sync.Mutex
	next  int
	total int
	spent float64
}

// NewTrialProgress returns a reporter. A nil writer disables reporting.
func NewTrialProgress(w io.Writer) *TrialProgress {
	if w == nil {
		return nil
	}
	return &TrialProgress{w: w}
}

// SetTotal records how many trials this run will attempt.
func (p *TrialProgress) SetTotal(n int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.total = n
	p.mu.Unlock()
}

// Banner prints the run id and how many trials are scheduled.
func (p *TrialProgress) Banner(runID string, trials int) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "run %s  %d trials\n", runID, trials)
}

// Start prints the trial that is about to run and returns its 1-based index.
func (p *TrialProgress) Start(taskID, modelID string, trial int) int {
	if p == nil {
		return 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.next++
	fmt.Fprintf(p.w, "start  %s  %s  %s  trial %d\n", p.pos(p.next), taskID, modelID, trial)
	return p.next
}

// Done prints the reward, wall time, and cost for a finished trial.
func (p *TrialProgress) Done(idx int, row ResultRow) {
	if p == nil || row.TaskID == "" {
		return
	}
	mark := "fail"
	if row.Reward == 1 {
		mark = "pass"
	}
	reason := ""
	if row.Reward == 0 && row.Reason != "" {
		reason = "  " + row.Reason
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spent += row.CostUSD
	fmt.Fprintf(p.w, "%-5s %s  %s  %s  trial %d  %.1fs  $%.3f  total $%.3f%s\n",
		mark, p.pos(idx), row.TaskID, row.ModelID, row.Trial, row.WallS, row.CostUSD, p.spent, reason)
}

// Error prints a trial that failed before it could be graded.
func (p *TrialProgress) Error(idx int, taskID, modelID string, trial int, err error) {
	if p == nil || err == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.w, "fail   %s  %s  %s  trial %d  %s\n", p.pos(idx), taskID, modelID, trial, err)
}

func (p *TrialProgress) pos(idx int) string {
	if p.total <= 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", idx, p.total)
}
