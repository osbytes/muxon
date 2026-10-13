//go:build integration

package router

import "fmt"

// ProbeServedContext measures the context window an endpoint actually honors.
// Pass thresholds are not defined yet; this entry point stays out of unit tests.
func ProbeServedContext(baseURL, modelID string) (int, error) {
	return 0, fmt.Errorf("served_context probe is integration-only for %s at %s", modelID, baseURL)
}
