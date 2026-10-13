package director

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

type loopDetector struct {
	history []loopEntry
	nudged  bool
	fires   int
}

type loopEntry struct {
	tool     string
	argsHash string
}

// RecordToolCall tracks tool usage for loop detection.
func (d *loopDetector) RecordToolCall(name string, args []byte) {
	h := sha256.Sum256(args)
	d.history = append(d.history, loopEntry{
		tool:     name,
		argsHash: hex.EncodeToString(h[:8]),
	})
	if len(d.history) > 8 {
		d.history = d.history[len(d.history)-8:]
	}
}

// CheckRepeated returns a nudge message when the same call repeats 3 times in the window.
func (d *loopDetector) CheckRepeated(workspaceChanged bool) (string, bool) {
	if workspaceChanged || len(d.history) < 3 {
		return "", false
	}
	last := d.history[len(d.history)-1]
	count := 0
	for i := len(d.history) - 1; i >= 0 && i >= len(d.history)-8; i-- {
		e := d.history[i]
		if e.tool == last.tool && e.argsHash == last.argsHash {
			count++
		}
	}
	if count < 3 {
		return "", false
	}
	d.fires++
	msg := fmt.Sprintf("You called %s with the same arguments %d times without progress. Try a different approach.", last.tool, count)
	if d.nudged {
		return msg, true
	}
	d.nudged = true
	return msg, true
}

func (d *loopDetector) ShouldEscalate() bool {
	return d.fires > 1
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
