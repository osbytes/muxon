package verifier

import (
	"regexp"
	"sort"
	"strings"
)

// Failure is one parsed test failure.
type Failure struct {
	Test     string
	Location string
	Message  string
}

// GoBundle is parsed go test output.
type GoBundle struct {
	Failures  []Failure
	Signature string
	Excerpt   string
}

var goFailLine = regexp.MustCompile(`--- FAIL: (\S+) \((.+)\)`)
var goFileLine = regexp.MustCompile(`\s+(\S+_test\.go:\d+):`)

// ParseGoTest extracts failures and a stall signature.
func ParseGoTest(output string) GoBundle {
	var fails []Failure
	lines := strings.Split(output, "\n")
	var names []string
	for i, line := range lines {
		if m := goFailLine.FindStringSubmatch(line); len(m) > 1 {
			f := Failure{Test: m[1]}
			if i+1 < len(lines) {
				if loc := goFileLine.FindStringSubmatch(lines[i+1]); len(loc) > 1 {
					f.Location = loc[1]
				}
				f.Message = strings.TrimSpace(lines[i+1])
			}
			fails = append(fails, f)
			names = append(names, m[1])
		}
	}
	sort.Strings(names)
	sig := ""
	if len(names) > 0 {
		sig = strings.Join(names, ",")
	} else {
		sig = firstErrorLine(output)
	}
	return GoBundle{
		Failures:  fails,
		Signature: sig,
		Excerpt:   excerptOutput(output, 20),
	}
}

func firstErrorLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "FAIL") || strings.Contains(line, "Error") {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

// FormatVerifyFailure builds the model-facing verify failure text.
func FormatVerifyFailure(vres Result, bundle GoBundle) string {
	var b strings.Builder
	b.WriteString("verify failed:\n")
	if bundle.Signature != "" {
		b.WriteString("signature: " + bundle.Signature + "\n")
	}
	for _, f := range bundle.Failures {
		b.WriteString(f.Test)
		if f.Location != "" {
			b.WriteString(" @ " + f.Location)
		}
		b.WriteByte('\n')
	}
	if b.Len() == len("verify failed:\n") {
		b.WriteString(vres.Excerpt)
	}
	return b.String()
}
