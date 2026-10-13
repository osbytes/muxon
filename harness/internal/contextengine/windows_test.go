package contextengine

import (
	"strings"
	"testing"
)

func TestSmallWindowTrimsRepoMapFirst(t *testing.T) {
	b := PrefixBudget{TurnInputBudget: 32768}
	hugeAgents := strings.Repeat("A", 5000)
	hugeMap := strings.Repeat("M", 20000)
	agents, repo, _ := b.Apply(hugeAgents, hugeMap)
	if len(repo) >= len(hugeMap) {
		t.Fatal("repo map should be trimmed")
	}
	if len(agents) < len(repo) {
		t.Fatalf("agents should retain more than repo map: agents=%d repo=%d", len(agents), len(repo))
	}
}
