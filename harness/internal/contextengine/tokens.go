package contextengine

import (
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
)

const approxCharsPerToken = 3.5

// Estimator tracks approx token counts with a per-session correction ratio.
type Estimator struct {
	ratio float64
}

// NewEstimator returns an estimator with ratio 1.
func NewEstimator() *Estimator {
	return &Estimator{ratio: 1}
}

// ApproxTokens estimates tokens from text length.
func ApproxTokens(text string) int {
	if text == "" {
		return 0
	}
	n := float64(len(text)) / approxCharsPerToken
	if n < 1 {
		return 1
	}
	return int(n)
}

// EstimateMessages sums message content with the current ratio.
func (e *Estimator) EstimateMessages(msgs []provider.Message) int {
	if e == nil {
		e = NewEstimator()
	}
	total := 0
	for _, m := range msgs {
		total += ApproxTokens(m.Content)
		for _, tc := range m.ToolCalls {
			total += ApproxTokens(string(tc.Arguments))
		}
	}
	return int(float64(total) * e.ratio)
}

// ObserveUsage sets ratio from provider-reported prompt tokens vs estimate.
func (e *Estimator) ObserveUsage(reportedPromptTokens int, estimated int) {
	if e == nil || estimated <= 0 || reportedPromptTokens <= 0 {
		return
	}
	e.ratio = float64(reportedPromptTokens) / float64(estimated)
}
