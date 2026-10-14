package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/router"
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
	prov, providerID, hint, err := loadInteractiveProvider(manifest)
	if err != nil {
		return exitError{code: 3, msg: err.Error()}
	}
	eff, _ := config.LoadEffective(dir)
	trusted := security.WorkspaceTrusted(dir, false)
	if err := tui.Run(tui.Options{
		Workspace:   dir,
		Trusted:     trusted,
		Provider:    prov,
		ProviderID:  providerID,
		Manifest:    manifest,
		Version:     version,
		WorkerModel: worker,
		Permission:  eff.Permission,
		SetupHint:   hint,
		AutoConnect: autoConnectID(prov),
	}); err != nil {
		return fmt.Errorf("chat: %w", err)
	}
	return nil
}

func autoConnectID(prov provider.Provider) string {
	if prov != nil || os.Getenv("THEE_SCRIPTED_PROVIDER_FILE") != "" {
		return ""
	}
	return config.LastProvider()
}

// loadInteractiveProvider opens the TUI even when the worker key is missing.
// The second result is the provider id. The third is a setup hint when the key is absent.
func loadInteractiveProvider(manifest *config.Manifest) (provider.Provider, string, string, error) {
	if os.Getenv("THEE_SCRIPTED_PROVIDER_FILE") != "" {
		p, _, err := loadRunProvider(manifest, "")
		return p, "", "", err
	}
	bind, err := router.ResolveWorker(manifest, "")
	if err != nil {
		return nil, "", "", err
	}
	p, err := router.NewProvider(bind)
	if err != nil {
		envName := bind.Provider.APIKeyEnv
		if envName == "" {
			envName = "the provider API key"
		}
		hint := "No provider key yet. Set " + envName + ", or run: thee auth login " + bind.ProviderName
		return nil, bind.ProviderName, hint, nil
	}
	return p, bind.ProviderName, "", nil
}
