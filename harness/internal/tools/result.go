package tools

import (
	"fmt"

	"github.com/PLACEHOLDER/muxon/harness/internal/security"
)

// FormatResult wraps tool output for the model context.
func FormatResult(toolName string, trustedWorkspace bool, body string, maxBytes int, maxLines int) string {
	body = security.RedactSecrets(body)
	if maxLines > 0 {
		body = security.TruncateLines(body, maxLines)
	}
	if maxBytes > 0 {
		body = security.TruncateBytes(body, maxBytes)
	}
	trust := "workspace"
	if !trustedWorkspace {
		trust = "untrusted"
	}
	return fmt.Sprintf("<tool_result tool=\"%s\" trust=\"%s\">\n%s\n</tool_result>", toolName, trust, body)
}

// OK formats a successful tool result.
func OK(env *Env, toolName string, body string, maxBytes int, maxLines int) Result {
	return Result{OK: true, Text: FormatResult(toolName, env.Trusted, body, maxBytes, maxLines)}
}

// Fail formats a failed tool result (still wrapped for consistency).
func Fail(env *Env, toolName string, msg string) Result {
	return Result{OK: false, Text: FormatResult(toolName, env.Trusted, msg, 0, 0)}
}
