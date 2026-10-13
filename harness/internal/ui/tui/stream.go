package tui

import (
	"fmt"
	"strings"

	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

// FormatEvent returns one transcript line for tool and verify events.
func FormatEvent(ev observability.Event) (string, bool) {
	switch ev.Type {
	case "tool.call":
		return fmt.Sprintf("▸ %s", ev.Name), true
	case "tool.result":
		status := "fail"
		if ev.OK {
			status = "ok"
		}
		return fmt.Sprintf("▸ result %s", status), true
	case "verify.run":
		status := "fail"
		if ev.Passed {
			status = "pass"
		}
		return fmt.Sprintf("▸ verify %s", status), true
	default:
		return "", false
	}
}

// CollectText reads assistant text from a provider stream.
func CollectText(s provider.Stream) (string, error) {
	if s == nil {
		return "", nil
	}
	defer s.Close()
	var b strings.Builder
	for {
		resp, err := s.Next()
		if err != nil {
			if err.Error() == "stream closed" || err.Error() == "EOF" {
				break
			}
			if b.Len() > 0 && (resp.Text == "") {
				break
			}
			return b.String(), err
		}
		b.WriteString(resp.Text)
		if resp.FinishReason != "" {
			break
		}
	}
	return b.String(), nil
}
