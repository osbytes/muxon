package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/ui/tui"
)

var chatCmd = &cobra.Command{
	Use:   "chat [path]",
	Short: "Interactive session",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return startChat(cmd, args)
	},
}

func init() {
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return startChat(cmd, args)
	}
	rootCmd.AddCommand(chatCmd)
}

func startChat(cmd *cobra.Command, args []string) error {
	setupLogging()
	dir, err := workspaceDir(cmd)
	if err != nil {
		return err
	}
	if len(args) == 1 {
		dir, err = absPath(args[0])
		if err != nil {
			return err
		}
	}
	manifest, err := config.LoadMergedManifest()
	if err != nil {
		return err
	}
	worker := ""
	if md, err := manifest.ResolveRole("worker"); err == nil {
		worker = md.Model
	}
	prov, _, err := loadRunProvider(manifest, "")
	if err != nil {
		return exitError{code: 3, msg: err.Error()}
	}
	eff, _ := config.LoadEffective(dir)
	trusted := security.WorkspaceTrusted(dir, false)
	if err := tui.Run(tui.Options{
		Workspace:   dir,
		Trusted:     trusted,
		Provider:    prov,
		Manifest:    manifest,
		Version:     version,
		WorkerModel: worker,
		Permission:  eff.Permission,
	}); err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	return nil
}
