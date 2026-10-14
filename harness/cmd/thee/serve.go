package main

import "github.com/spf13/cobra"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "IDE protocol (not in this build)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return exitError{code: 3, msg: "IDE protocol is not in this build"}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
