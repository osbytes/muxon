package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "thee",
	Short:        "thee coding agent harness",
	SilenceUsage: true,
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
