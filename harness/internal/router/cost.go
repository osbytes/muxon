package router

import (
	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
)

// RatesFromModel extracts list prices from a model definition.
func RatesFromModel(m *config.ModelDef) Rates {
	return Rates{
		InputPerM:       m.InputPerM,
		OutputPerM:      m.OutputPerM,
		CachedInputPerM: m.CachedInputPerM,
		CacheWritePerM:  m.CacheWritePerM,
		ReasoningPerM:   m.ReasoningPerM,
	}
}

// Rates holds per-million token prices in USD.
type Rates struct {
	InputPerM       float64
	OutputPerM      float64
	CachedInputPerM float64
	CacheWritePerM  float64
	ReasoningPerM   float64
}

// CostUSD computes session cost from usage and rates.
func CostUSD(r Rates, u provider.Usage) (float64, bool) {
	if u.ProviderCostUSD > 0 {
		return u.ProviderCostUSD, true
	}
	if r.InputPerM == 0 && r.OutputPerM == 0 {
		return 0, false
	}
	cached := u.CachedTokens
	if cached < 0 {
		cached = 0
	}
	cacheWrite := u.CacheWriteTokens
	uncached := u.InputTokens - cached - cacheWrite
	if uncached < 0 {
		uncached = 0
	}
	cachedRate := r.CachedInputPerM
	if cachedRate == 0 {
		cachedRate = r.InputPerM
	}
	cacheWriteRate := r.CacheWritePerM
	if cacheWriteRate == 0 {
		cacheWriteRate = r.InputPerM
	}
	outTok := u.OutputTokens
	reasoning := u.ReasoningTokens
	if reasoning > outTok {
		reasoning = 0
	}
	nonReasoningOut := outTok - reasoning
	cost := (float64(uncached)*r.InputPerM + float64(cached)*cachedRate + float64(cacheWrite)*cacheWriteRate + float64(nonReasoningOut)*r.OutputPerM) / 1e6
	if reasoning > 0 && r.ReasoningPerM > 0 {
		cost += float64(reasoning) * r.ReasoningPerM / 1e6
	} else if reasoning > 0 {
		cost += float64(reasoning) * r.OutputPerM / 1e6
	}
	return cost, true
}
