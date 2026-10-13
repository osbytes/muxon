package director

// SessionCaps are per-goal limits from thee.toml defaults.
type SessionCaps struct {
	MaxToolRounds      int
	MaxVerifyAttempts  int
	TokenBudget        int
	MaxEscalations     int
	MaxOutputTokens    int
}

// DefaultCaps returns spec defaults.
func DefaultCaps() SessionCaps {
	return SessionCaps{
		MaxToolRounds:     40,
		MaxVerifyAttempts: 6,
		TokenBudget:       2_000_000,
		MaxEscalations:    1,
		MaxOutputTokens:   4096,
	}
}

// Budget tracks remaining token budget for the goal.
type Budget struct {
	Caps      SessionCaps
	Remaining int
	Compacted bool
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
