package director

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PLACEHOLDER/muxon/harness/internal/observability"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

const (
	loopRepeatThreshold = 3
	loopRereadThreshold = 3
	loopPatchThrash     = 2
	loopWindow          = 8
)

type loopDetector struct {
	history []loopEntry
	fires   int
}

type loopEntry struct {
	tool      string
	argsHash  string
	revision  int
	path      string
	start     int
	end       int
	oldHash   string
	ok        bool
	resultSet bool
}

// RecordToolCall tracks one call at the workspace revision it was issued.
func (d *loopDetector) RecordToolCall(name string, args []byte, revision int) {
	h := sha256.Sum256(args)
	path, start, end, oldHash := loopCallMeta(args)
	d.history = append(d.history, loopEntry{
		tool:     name,
		argsHash: hex.EncodeToString(h[:8]),
		revision: revision,
		path:     path,
		start:    start,
		end:      end,
		oldHash:  oldHash,
	})
	if len(d.history) > loopWindow {
		d.history = d.history[len(d.history)-loopWindow:]
	}
}

// RecordResult marks the latest matching call as succeeded or failed.
func (d *loopDetector) RecordResult(call provider.ToolCall, ok bool) {
	sum := sha256.Sum256(call.Arguments)
	want := hex.EncodeToString(sum[:8])
	for i := len(d.history) - 1; i >= 0; i-- {
		e := &d.history[i]
		if e.tool == call.Name && !e.resultSet && e.argsHash == want {
			e.ok = ok
			e.resultSet = true
			return
		}
	}
}

// Check reports a loop nudge. pattern is repeated_call, reread, or patch_thrash.
func (d *loopDetector) Check() (msg, pattern string, fire bool) {
	checks := []struct {
		pattern string
		fn      func() (string, bool)
	}{
		{"repeated_call", d.repeated},
		{"reread", d.reread},
		{"patch_thrash", d.patchThrash},
	}
	for _, c := range checks {
		text, ok := c.fn()
		if !ok {
			continue
		}
		d.fires++
		return text, c.pattern, true
	}
	return "", "", false
}

func (d *loopDetector) ShouldEscalate() bool {
	return d.fires > 1
}

func (d *loopDetector) repeated() (string, bool) {
	if len(d.history) < loopRepeatThreshold {
		return "", false
	}
	last := d.history[len(d.history)-1]
	count := 0
	for i := len(d.history) - 1; i >= 0 && i >= len(d.history)-loopWindow; i-- {
		e := d.history[i]
		if e.tool == last.tool && e.argsHash == last.argsHash && e.revision == last.revision {
			count++
		}
	}
	if count < loopRepeatThreshold {
		return "", false
	}
	return fmt.Sprintf("You called %s with the same arguments %d times without progress. Try a different approach.", last.tool, count), true
}

func (d *loopDetector) reread() (string, bool) {
	if len(d.history) == 0 {
		return "", false
	}
	last := d.history[len(d.history)-1]
	if last.tool != "read" || last.path == "" {
		return "", false
	}
	count := 0
	for _, e := range d.window() {
		if e.tool == "read" && e.path == last.path && e.revision == last.revision && rangesOverlap(e.start, e.end, last.start, last.end) {
			count++
		}
	}
	if count < loopRereadThreshold {
		return "", false
	}
	return fmt.Sprintf("You have read %s lines %d–%d %d times; it has not changed. Try a different approach.", last.path, last.start, last.end, count), true
}

func (d *loopDetector) patchThrash() (string, bool) {
	type key struct{ path, old string }
	counts := map[key]int{}
	var hit key
	found := false
	for _, e := range d.window() {
		if e.tool != "patch" || !e.resultSet || e.ok || e.path == "" {
			continue
		}
		k := key{e.path, e.oldHash}
		counts[k]++
		if counts[k] >= loopPatchThrash {
			hit = k
			found = true
		}
	}
	if !found {
		return "", false
	}
	return fmt.Sprintf("A patch of %s was rejected twice with the same change. Re-read the file and try a different edit.", hit.path), true
}

func (d *loopDetector) window() []loopEntry {
	if len(d.history) <= loopWindow {
		return d.history
	}
	return d.history[len(d.history)-loopWindow:]
}

func rangesOverlap(a0, a1, b0, b1 int) bool {
	if (a0 == 0 && a1 == 0) || (b0 == 0 && b1 == 0) {
		return true
	}
	if a1 < a0 || b1 < b0 {
		return false
	}
	return a0 <= b1 && b0 <= a1
}

func loopCallMeta(args []byte) (path string, start, end int, oldHash string) {
	var meta struct {
		Path         string `json:"path"`
		StartLine    int    `json:"start_line"`
		EndLine      int    `json:"end_line"`
		Diff         string `json:"diff"`
		Replacements []struct {
			Search string `json:"search"`
		} `json:"replacements"`
	}
	if err := json.Unmarshal(args, &meta); err != nil {
		return "", 0, 0, ""
	}
	var old strings.Builder
	for _, r := range meta.Replacements {
		old.WriteString(r.Search)
		old.WriteByte(0)
	}
	if old.Len() == 0 {
		old.WriteString(meta.Diff)
	}
	sum := sha256.Sum256([]byte(old.String()))
	return meta.Path, meta.StartLine, meta.EndLine, hex.EncodeToString(sum[:8])
}

func loopEvent(tw *observability.Writer, pattern string) {
	_ = tw.Append(observability.Event{Type: "loop.detected", Reason: pattern})
}

func verifySignatureMessage(sig string) string {
	return "verify failed:\nsignature: " + sig + "\n"
}

func appendVerifyFailure(history []provider.Message, text string) []provider.Message {
	return append(history, provider.Message{Role: "user", Content: text})
}
