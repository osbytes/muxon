package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
)

var rootCmd = &cobra.Command{
	Use:          "thee",
	Short:        "thee coding agent harness",
	SilenceUsage: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		_, err = config.LoadDotEnv(wd)
		return err
	},
}

func execute() error {
	return rootCmd.Execute()
}

func init() {
	bindGlobalFlags(rootCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(runCmd)
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
