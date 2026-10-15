package observability

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// SessionCostSummary is parsed from trace session.end.
type SessionCostSummary struct {
	InputTokens        int
	OutputTokens       int
	CachedTokens       int
	ReasoningTokens    int
	TotalCostUSD       float64
	Outcome            string
	ToolRounds         int
	Escalations        int
	MalformedToolCalls int
	LoopsDetected      int
	SalvagedToolCalls  int
	PatchApplies       int
	WellFormedPatches  int
}

// ReadLatestSessionCost reads the newest session trace under workspace/.thee/sessions.
func ReadLatestSessionCost(workspaceRoot string) (SessionCostSummary, string, error) {
	sessionsDir := filepath.Join(workspaceRoot, ".thee", "sessions")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return SessionCostSummary{}, "", err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		return SessionCostSummary{}, "", os.ErrNotExist
	}
	sort.Strings(dirs)
	latest := dirs[len(dirs)-1]
	tracePath := filepath.Join(sessionsDir, latest, "trace.jsonl")
	return ReadSessionCost(tracePath)
}

// ReadSessionCost parses totals from a trace file.
func ReadSessionCost(tracePath string) (SessionCostSummary, string, error) {
	f, err := os.Open(tracePath)
	if err != nil {
		return SessionCostSummary{}, "", err
	}
	defer f.Close()
	var sum SessionCostSummary
	var salvaged, patches, wellFormed int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ev Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			continue
		}
		switch ev.Type {
		case "tool.call":
			if ev.Salvaged {
				salvaged++
			}
		case "tool.result":
			if ev.ApplyStrategy == "" {
				continue
			}
			patches++
			if ev.ApplyStrategy == "exact" || ev.ApplyStrategy == "whitespace" {
				wellFormed++
			}
		case "session.end":
			sum = SessionCostSummary{
				InputTokens:        ev.InputTokens,
				OutputTokens:       ev.OutputTokens,
				CachedTokens:       ev.CachedTokens,
				ReasoningTokens:    ev.ReasoningTokens,
				TotalCostUSD:       ev.TotalCostUSD,
				Outcome:            ev.Outcome,
				ToolRounds:         ev.ToolRounds,
				Escalations:        ev.EscalationGeneration,
				MalformedToolCalls: ev.MalformedToolCalls,
				LoopsDetected:      ev.LoopsDetected,
			}
		}
	}
	sum.SalvagedToolCalls = salvaged
	sum.PatchApplies = patches
	sum.WellFormedPatches = wellFormed
	return sum, tracePath, sc.Err()
}
