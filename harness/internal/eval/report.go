package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"sort"
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
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
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
	return formatMetrics(rows, false)
}

// FormatGroupedReport prints one block per model and override set, with a
// task-clustered 95% bootstrap interval on pass rate.
func FormatGroupedReport(rows []ResultRow) string {
	if len(rows) == 0 {
		return "no eval results\n"
	}
	groups := map[string][]ResultRow{}
	var keys []string
	for _, r := range rows {
		k := r.ModelID + "\x00" + overrideKey(r.Overrides)
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], r)
	}
	sort.Strings(keys)
	var b strings.Builder
	var passed int
	var cost float64
	for _, r := range rows {
		passed += r.Reward
		cost += r.CostUSD
	}
	fmt.Fprintf(&b, "%d/%d passed   $%.3f\n\n", passed, len(rows), cost)
	for _, k := range keys {
		parts := strings.SplitN(k, "\x00", 2)
		fmt.Fprintf(&b, "%s\n", parts[0])
		if parts[1] != "" {
			fmt.Fprintf(&b, "  %s\n", parts[1])
		}
		b.WriteString(formatMetrics(groups[k], true))
		b.WriteString("\n")
	}
	return b.String()
}

type metricTally struct {
	n         int
	reward    float64
	cost      float64
	tokens    float64
	wall      float64
	esc       int
	malformed int
	salvaged  int
	loops     int
	rounds    int
	timeouts  int
}

func tallyRows(rows []ResultRow) metricTally {
	var t metricTally
	t.n = len(rows)
	for _, r := range rows {
		t.reward += float64(r.Reward)
		t.cost += r.CostUSD
		t.tokens += float64(r.InputTokens + r.OutputTokens)
		t.wall += r.WallS
		t.malformed += r.MalformedToolCalls
		t.salvaged += r.SalvagedToolCalls
		t.rounds += r.ToolRounds
		if r.Escalations > 0 {
			t.esc++
		}
		if r.LoopsDetected > 0 {
			t.loops++
		}
		if r.TimeoutHit {
			t.timeouts++
		}
	}
	return t
}

func formatMetrics(rows []ResultRow, withCI bool) string {
	t := tallyRows(rows)
	if t.n == 0 {
		return "no eval results\n"
	}
	ci := ""
	if withCI {
		lo, hi := BootstrapPassRate(rows, 1000, 1)
		ci = fmt.Sprintf("  (95%% CI %.3f-%.3f)", lo, hi)
	}
	mal, salvage := t.toolRates()
	var b strings.Builder
	writeMetric(&b, "passed", fmt.Sprintf("%d/%d", int(t.reward), t.n))
	writeMetric(&b, "pass rate", fmt.Sprintf("%.3f%s", t.reward/float64(t.n), ci))
	writeMetric(&b, "total cost", fmt.Sprintf("$%.3f", t.cost))
	writeMetric(&b, "cost / pass", perPass(t.reward, t.cost, "%.4f"))
	writeMetric(&b, "tokens / pass", perPass(t.reward, t.tokens, "%.1f"))
	writeMetric(&b, "time / task", fmt.Sprintf("%.2fs", t.wall/float64(t.n)))
	writeMetric(&b, "timeouts", fmt.Sprintf("%.3f", float64(t.timeouts)/float64(t.n)))
	writeMetric(&b, "escalations", fmt.Sprintf("%.3f", float64(t.esc)/float64(t.n)))
	writeMetric(&b, "malformed", fmt.Sprintf("%.3f", mal))
	writeMetric(&b, "salvage", fmt.Sprintf("%.3f", salvage))
	writeMetric(&b, "loops", fmt.Sprintf("%.3f", float64(t.loops)/float64(t.n)))
	return b.String()
}

func (t metricTally) toolRates() (malformed, salvage float64) {
	if t.rounds == 0 {
		return 0, 0
	}
	return float64(t.malformed) / float64(t.rounds), float64(t.salvaged) / float64(t.rounds)
}

func perPass(reward, total float64, spec string) string {
	if reward <= 0 {
		return "n/a"
	}
	return fmt.Sprintf(spec, total/reward)
}

func writeMetric(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "  %-16s %s\n", label, value)
}

// BootstrapPassRate resamples tasks with replacement and returns the 2.5 and
// 97.5 percentiles of the mean reward. Trials inside a task stay together.
func BootstrapPassRate(rows []ResultRow, samples int, seed int64) (low, high float64) {
	if len(rows) == 0 {
		return 0, 0
	}
	byTask := map[string][]ResultRow{}
	var ids []string
	for _, r := range rows {
		if _, ok := byTask[r.TaskID]; !ok {
			ids = append(ids, r.TaskID)
		}
		byTask[r.TaskID] = append(byTask[r.TaskID], r)
	}
	sort.Strings(ids)
	if samples < 2 {
		samples = 2
	}
	rng := rand.New(rand.NewSource(seed))
	means := make([]float64, samples)
	for i := 0; i < samples; i++ {
		var reward, n float64
		for k := 0; k < len(ids); k++ {
			id := ids[rng.Intn(len(ids))]
			for _, r := range byTask[id] {
				reward += float64(r.Reward)
				n++
			}
		}
		if n == 0 {
			continue
		}
		means[i] = reward / n
	}
	sort.Float64s(means)
	lo := int(math.Floor(0.025 * float64(samples)))
	hi := int(math.Ceil(0.975*float64(samples))) - 1
	if lo < 0 {
		lo = 0
	}
	if hi >= samples {
		hi = samples - 1
	}
	if hi < lo {
		hi = lo
	}
	return means[lo], means[hi]
}

// CompareRows pairs tasks and reports an exact McNemar p-value on discordant outcomes.
func CompareRows(a, b []ResultRow) string {
	left := taskPass(a)
	right := taskPass(b)
	var keys []string
	seen := map[string]bool{}
	for k := range left {
		seen[k] = true
		keys = append(keys, k)
	}
	for k := range right {
		if !seen[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var both, aOnly, bOnly int
	var flipped []string
	for _, k := range keys {
		la, lok := left[k]
		rb, rok := right[k]
		if !lok || !rok {
			continue
		}
		switch {
		case la && rb:
			both++
		case !la && !rb:
		case la && !rb:
			aOnly++
			flipped = append(flipped, fmt.Sprintf("  %s 1 -> 0", strings.ReplaceAll(k, "\x00", " ")))
		default:
			bOnly++
			flipped = append(flipped, fmt.Sprintf("  %s 0 -> 1", strings.ReplaceAll(k, "\x00", " ")))
		}
	}
	p := exactMcNemar(bOnly, aOnly)
	var bld strings.Builder
	fmt.Fprintf(&bld, "pairs %d\na_only_pass %d\nb_only_pass %d\nmcnemar_p %.4f\n", both+aOnly+bOnly, aOnly, bOnly, p)
	if len(flipped) > 0 {
		bld.WriteString("flipped:\n")
		bld.WriteString(strings.Join(flipped, "\n"))
		bld.WriteString("\n")
	}
	return bld.String()
}

func taskPass(rows []ResultRow) map[string]bool {
	type acc struct{ pass, n int }
	sum := map[string]*acc{}
	for _, r := range rows {
		k := r.TaskID + "\x00" + r.ModelID + "\x00" + overrideKey(r.Overrides)
		a := sum[k]
		if a == nil {
			a = &acc{}
			sum[k] = a
		}
		a.n++
		a.pass += r.Reward
	}
	out := map[string]bool{}
	for k, a := range sum {
		out[k] = a.pass*2 >= a.n
	}
	return out
}

func exactMcNemar(b, c int) float64 {
	n := b + c
	if n == 0 {
		return 1
	}
	k := b
	if c < k {
		k = c
	}
	num := new(big.Int)
	for i := 0; i <= k; i++ {
		num.Add(num, binom(n, i))
	}
	den := new(big.Int).Lsh(big.NewInt(1), uint(n))
	rat := new(big.Rat).SetFrac(num, den)
	rat.Mul(rat, big.NewRat(2, 1))
	if rat.Cmp(big.NewRat(1, 1)) > 0 {
		return 1
	}
	f, _ := rat.Float64()
	return f
}

func binom(n, k int) *big.Int {
	if k < 0 || k > n {
		return big.NewInt(0)
	}
	if k > n-k {
		k = n - k
	}
	res := big.NewInt(1)
	for i := 1; i <= k; i++ {
		res.Mul(res, big.NewInt(int64(n-k+i)))
		res.Div(res, big.NewInt(int64(i)))
	}
	return res
}
