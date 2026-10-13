package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadResults reads results.jsonl.
func LoadResults(path string) ([]ResultRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var rows []ResultRow
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var row ResultRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, sc.Err()
}

// FormatReport prints basket aggregates from result rows.
func FormatReport(rows []ResultRow) string {
	n := len(rows)
	if n == 0 {
		return "no eval results\n"
	}
	var reward, cost, tokens float64
	var wall float64
	var esc, malformed, loops, rounds int
	for _, r := range rows {
		reward += float64(r.Reward)
		cost += r.CostUSD
		tokens += float64(r.InputTokens + r.OutputTokens)
		wall += r.WallS
		if r.Escalations > 0 {
			esc++
		}
		malformed += r.MalformedToolCalls
		if r.LoopsDetected > 0 {
			loops++
		}
		rounds += r.ToolRounds
	}
	pass := reward / float64(n)
	costPer := "undefined"
	tokPer := "undefined"
	if reward > 0 {
		costPer = fmt.Sprintf("%.4f", cost/reward)
		tokPer = fmt.Sprintf("%.1f", tokens/reward)
	}
	malRate := 0.0
	salvage := 0.0
	if rounds > 0 {
		malRate = float64(malformed) / float64(rounds)
	}
	return fmt.Sprintf(
		"pass_rate %.3f\ncost_per_pass %s\ntokens_per_pass %s\nseconds_per_task %.2f\nescalation_rate %.3f\nmalformed_rate %.3f\nsalvage_rate %.3f\nloop_rate %.3f\n",
		pass, costPer, tokPer, wall/float64(n), float64(esc)/float64(n), malRate, salvage, float64(loops)/float64(n),
	)
}
