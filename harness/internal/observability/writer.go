package observability

import (
	"encoding/json"
	"io"
	"os"
	"sync"
)

// Writer appends JSONL events.
type Writer struct {
	mu  sync.Mutex
	out io.Writer
}

// NewWriter creates a trace writer targeting a file path.
func NewWriter(path string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Writer{out: f}, nil
}

// NewWriterIO wraps an io.Writer (tests).
func NewWriterIO(w io.Writer) *Writer {
	return &Writer{out: w}
}

// Append writes one event as a JSON line.
func (w *Writer) Append(ev Event) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	b, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if _, err := w.out.Write(b); err != nil {
		return err
	}
	if _, err := w.out.Write([]byte{'\n'}); err != nil {
		return err
	}
	return nil
}

// Close closes the underlying file when it is an *os.File.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if f, ok := w.out.(*os.File); ok {
		return f.Close()
	}
	return nil
}
