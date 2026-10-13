package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/director"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
)

type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a headless agent session",
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt, _ := cmd.Flags().GetString("prompt")
		trust, _ := cmd.Flags().GetBool("trust")
		dir, _ := cmd.Flags().GetString("directory")
		if prompt == "" {
			return exitError{code: 3, msg: "missing -p/--prompt"}
		}
		if dir == "" {
			wd, err := os.Getwd()
			if err != nil {
				return err
			}
			dir = wd
		}
		abs, err := filepath.Abs(dir)
		if err != nil {
			return err
		}
		manifest, err := config.LoadEmbeddedManifest()
		if err != nil {
			return exitError{code: 3, msg: err.Error()}
		}
		prov, err := loadRunProvider()
		if err != nil {
			return exitError{code: 3, msg: err.Error()}
		}
		out, err := director.Run(context.Background(), director.Config{
			Goal:        prompt,
			Workspace:   abs,
			Trusted:     trust,
			TheeVersion: version,
			Provider:    prov,
			Manifest:    manifest,
		})
		if err != nil {
			return err
		}
		if out.ExitCode != 0 {
			return exitError{code: out.ExitCode, msg: fmt.Sprintf("%s: %s", out.State, out.Reason)}
		}
		return nil
	},
}

func init() {
	runCmd.Flags().StringP("prompt", "p", "", "Goal prompt")
	runCmd.Flags().Bool("trust", false, "Trust workspace for verify")
	runCmd.Flags().String("directory", "", "Workspace directory")
}

func loadRunProvider() (provider.Provider, error) {
	if path := os.Getenv("THEE_SCRIPTED_PROVIDER_FILE"); path != "" {
		return scripted.FromEnv()
	}
	return nil, fmt.Errorf("no provider configured: set THEE_SCRIPTED_PROVIDER_FILE for scripted runs")
}
