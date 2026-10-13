package verifier

import (
	"context"
	"os"
	"path/filepath"
	"sync"
)

// BaselineCache runs verify once before the first model turn.
type BaselineCache struct {
	mu     sync.Mutex
	stamp  string
	result *Result
	done   chan struct{}
}

// StartBaseline kicks off baseline verify when enabled.
func (b *BaselineCache) StartBaseline(ctx context.Context, root string, argv []string, enabled bool) {
	if !enabled || len(argv) == 0 {
		b.done = closedChan()
		return
	}
	b.stamp = treeStamp(root)
	b.done = make(chan struct{})
	go func() {
		res, err := Run(ctx, root, argv)
		if err == nil {
			b.mu.Lock()
			b.result = &res
			b.mu.Unlock()
		}
		close(b.done)
	}()
}

func closedChan() chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}

// Note returns a one-line baseline message for the first turn.
func (b *BaselineCache) Note() string {
	if b.done != nil {
		<-b.done
	}
	b.mu.Lock()
	res := b.result
	b.mu.Unlock()
	if res == nil {
		return ""
	}
	if res.Passed {
		return ""
	}
	return "Baseline verify failed before edits; fix without adding new failures.\n"
}

func treeStamp(root string) string {
	gm := filepath.Join(root, "go.mod")
	if st, err := os.Stat(gm); err == nil {
		return gm + ":" + st.ModTime().String()
	}
	return root
}
