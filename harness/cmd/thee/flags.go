package main

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagJSON      bool
	flagQuiet     bool
	flagVerbose   bool
	flagDirectory string
)

func bindGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Machine-readable NDJSON on stdout")
	cmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Errors and final outcome only")
	cmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Extra harness logging to stderr")
	cmd.PersistentFlags().StringVarP(&flagDirectory, "directory", "C", "", "Workspace root")
}

func setupLogging() {
	if flagVerbose {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})))
	}
}

func workspaceDir(cmd *cobra.Command) (string, error) {
	if flagDirectory != "" {
		return absPath(flagDirectory)
	}
	if d, _ := cmd.Flags().GetString("directory"); d != "" {
		return absPath(d)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return wd, nil
}
