package contextengine

// PrefixBudget caps slow-changing prefix sections for small windows.
type PrefixBudget struct {
	TurnInputBudget int
}

const (
	agentsSharePct   = 0.05
	repoMapSharePct  = 0.08
	prefixMaxShare   = 0.25
	agentsFloorBytes = 1024
	agentsCeilBytes  = 4096
	repoFloorBytes   = 1024
	repoCeilBytes    = 16384
)

// Apply trims repo map before AGENTS.md when over budget.
func (b PrefixBudget) Apply(agents, repoMap string) (string, string, string) {
	if b.TurnInputBudget <= 0 {
		b.TurnInputBudget = 32768
	}
	agentsMax := clampInt(int(float64(b.TurnInputBudget)*agentsSharePct*3.5), agentsFloorBytes, agentsCeilBytes)
	repoMax := clampInt(int(float64(b.TurnInputBudget)*repoMapSharePct*3.5), repoFloorBytes, repoCeilBytes)
	prefixMax := int(float64(b.TurnInputBudget) * prefixMaxShare)

	repoMap = truncateBytes(repoMap, repoMax)
	agents = truncateBytes(agents, agentsMax)

	note := ""
	for len(agents)+len(repoMap) > prefixMax && len(repoMap) > 0 {
		repoMap = truncateBytes(repoMap, len(repoMap)*3/4)
		note = "Full AGENTS.md and repo-map available via read and repo-map.\n"
	}
	if len(agents)+len(repoMap) > prefixMax {
		agents = truncateBytes(agents, prefixMax-len(repoMap))
		if note == "" {
			note = "Full AGENTS.md and repo-map available via read and repo-map.\n"
		}
	}
	return agents, repoMap, note
}

func truncateBytes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "\n...[trimmed]"
}

func clampInt(v, floor, ceil int) int {
	if v < floor {
		return floor
	}
	if v > ceil {
		return ceil
	}
	return v
}
