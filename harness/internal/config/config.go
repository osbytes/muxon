package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// EffectiveConfig is merged user + project settings.
type EffectiveConfig struct {
	MaxPermission string
	Permission    string
	VerifyCommand string
	FilterCommand string
	AllowCommit   bool
	Baseline      bool
}

// LoadEffective merges config layers for a workspace.
func LoadEffective(workspaceRoot string) (EffectiveConfig, error) {
	out := EffectiveConfig{
		MaxPermission: "full",
		Permission:    "edit",
		Baseline:      true,
	}
	home, err := TheeHome()
	if err == nil {
		mergeFile(&out, filepath.Join(home, "config.toml"), false)
	}
	mergeFile(&out, filepath.Join(workspaceRoot, "thee.toml"), true)
	out.Permission = capPermission(out.Permission, out.MaxPermission)
	return out, nil
}

type fileConfig struct {
	Security struct {
		Permission    string `toml:"permission"`
		MaxPermission string `toml:"max_permission"`
	} `toml:"security"`
	Verify struct {
		Command       string `toml:"command"`
		FilterCommand string `toml:"filter_command"`
		Baseline      *bool  `toml:"baseline"`
	} `toml:"verify"`
	Git struct {
		AllowCommit bool `toml:"allow_commit"`
	} `toml:"git"`
}

func mergeFile(out *EffectiveConfig, path string, project bool) {
	var fc fileConfig
	if _, err := toml.DecodeFile(path, &fc); err != nil {
		return
	}
	if fc.Security.MaxPermission != "" {
		out.MaxPermission = fc.Security.MaxPermission
	}
	if fc.Security.Permission != "" {
		if project {
			out.Permission = capPermission(fc.Security.Permission, out.MaxPermission)
		} else {
			out.Permission = fc.Security.Permission
			if fc.Security.MaxPermission != "" {
				out.MaxPermission = fc.Security.MaxPermission
			}
		}
	}
	if fc.Verify.Command != "" {
		out.VerifyCommand = fc.Verify.Command
	}
	if fc.Verify.FilterCommand != "" {
		out.FilterCommand = fc.Verify.FilterCommand
	}
	if fc.Git.AllowCommit {
		out.AllowCommit = true
	}
	if fc.Verify.Baseline != nil {
		out.Baseline = *fc.Verify.Baseline
	}
}

func capPermission(project, ceiling string) string {
	rank := map[string]int{"read-only": 0, "edit": 1, "full": 2}
	p := rank[normalizePerm(project)]
	c := rank[normalizePerm(ceiling)]
	if p > c {
		return ceiling
	}
	return project
}

func normalizePerm(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if p == "" {
		return "edit"
	}
	return p
}

// VerifyCommandFromEnv returns CLI/env override.
func VerifyCommandFromEnv() string {
	if v := os.Getenv("THEE_VERIFY_CMD"); v != "" {
		return v
	}
	return ""
}
