package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// UserConfig is ~/.thee/config.toml (user-level only).
type UserConfig struct {
	Providers struct {
		Catalog string `toml:"catalog"`
	} `toml:"providers"`
	Session struct {
		MaxToolRounds            int     `toml:"max_tool_rounds"`
		CostBudgetUSD            float64 `toml:"cost_budget_usd"`
		EscalateAfterVerifyFails int     `toml:"escalate_after_verify_fails"`
		MaxEscalations           int     `toml:"max_escalations"`
	} `toml:"session"`
}

// LoadUserConfig reads user config when present.
func LoadUserConfig() (UserConfig, error) {
	home, err := TheeHome()
	if err != nil {
		return UserConfig{}, err
	}
	path := filepath.Join(home, "config.toml")
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return UserConfig{}, nil
		}
		return UserConfig{}, err
	}
	var uc UserConfig
	if _, err := toml.DecodeFile(path, &uc); err != nil {
		return UserConfig{}, err
	}
	return uc, nil
}
