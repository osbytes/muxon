package observability

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriterRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriterIO(&buf)
	if err := w.Append(SessionStartEvent("dev", "/tmp/ws")); err != nil {
		t.Fatal(err)
	}
	if err := w.Append(SessionEndEvent("done", "verified", UsageTotals{}, 0, 0, 0, 0, 0)); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines: %d", len(lines))
	}
	var ev Event
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.SchemaVersion != SchemaVersion || ev.Workspace != "/tmp/ws" {
		t.Fatalf("first event: %+v", ev)
	}
	if err := json.Unmarshal([]byte(lines[1]), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Outcome != "done" {
		t.Fatalf("second event: %+v", ev)
	}
}
