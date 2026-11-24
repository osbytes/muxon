package director

import "github.com/PLACEHOLDER/muxon/harness/internal/config"

// DefaultInteractiveMaxToolRounds is the per-goal cap for an interactive
// session when neither config nor a flag sets one.
const DefaultInteractiveMaxToolRounds = 100

// SessionCaps are per-goal limits from muxon.toml defaults.
type SessionCaps struct {
	MaxToolRounds            int
	MaxVerifyAttempts        int
	TokenBudget              int
	MaxEscalations           int
	MaxOutputTokens          int
	CostBudgetUSD            float64
	EscalateAfterVerifyFails int
}

// DefaultCaps returns spec defaults.
func DefaultCaps() SessionCaps {
	return SessionCaps{
		MaxToolRounds:            40,
		MaxVerifyAttempts:        6,
		TokenBudget:              2_000_000,
		MaxEscalations:           1,
		MaxOutputTokens:          4096,
		EscalateAfterVerifyFails: 2,
	}
}

// Budget tracks remaining token budget for the goal.
type Budget struct {
	Caps         SessionCaps
	Remaining    int
	Compacted    bool
	CostSpentUSD float64
}

// ResolveCaps applies defaults, then user config, then a flag override.
// Interactive runs use DefaultInteractiveMaxToolRounds when nothing else sets the cap.
func ResolveCaps(interactive bool, cfgCaps SessionCaps, maxRounds int, uc config.UserConfig) SessionCaps {
	caps := cfgCaps
	fromDefault := false
	if caps.MaxToolRounds == 0 {
		caps = DefaultCaps()
		fromDefault = true
	}
	if uc.Session.MaxToolRounds > 0 && maxRounds == 0 {
		caps.MaxToolRounds = uc.Session.MaxToolRounds
		fromDefault = false
	}
	if uc.Session.CostBudgetUSD > 0 && caps.CostBudgetUSD == 0 {
		caps.CostBudgetUSD = uc.Session.CostBudgetUSD
	}
	if uc.Session.MaxEscalations > 0 && caps.MaxEscalations == 0 {
		caps.MaxEscalations = uc.Session.MaxEscalations
	}
	if uc.Session.EscalateAfterVerifyFails > 0 && caps.EscalateAfterVerifyFails == 0 {
		caps.EscalateAfterVerifyFails = uc.Session.EscalateAfterVerifyFails
	}
	if maxRounds > 0 {
		caps.MaxToolRounds = maxRounds
		fromDefault = false
	}
	if interactive && fromDefault {
		caps.MaxToolRounds = DefaultInteractiveMaxToolRounds
	}
	return caps
}

// NewBudget creates a budget tracker.
func NewBudget(caps SessionCaps) Budget {
	if caps.TokenBudget <= 0 {
		caps = DefaultCaps()
	}
	return Budget{Caps: caps, Remaining: caps.TokenBudget}
}

// FitsModelCall reports whether estimated input+output fits.
func (b *Budget) FitsModelCall(estimatedInput int) bool {
	return estimatedInput+b.Caps.MaxOutputTokens <= b.Remaining
}

// Charge subtracts usage after a model call.
func (b *Budget) Charge(input, output int) {
	b.Remaining -= input + output
}

// FitsCost reports whether estimated dollars fit the session cost cap.
func (b *Budget) FitsCost(estimateUSD float64) bool {
	if b.Caps.CostBudgetUSD <= 0 {
		return true
	}
	return b.CostSpentUSD+estimateUSD <= b.Caps.CostBudgetUSD
}

// ChargeCost adds USD to the running total.
func (b *Budget) ChargeCost(usd float64) {
	if usd > 0 {
		b.CostSpentUSD += usd
	}
}

// RemainingCostUSD returns dollars left when a cap is set.
func (b *Budget) RemainingCostUSD() float64 {
	if b.Caps.CostBudgetUSD <= 0 {
		return 0
	}
	return b.Caps.CostBudgetUSD - b.CostSpentUSD
}
