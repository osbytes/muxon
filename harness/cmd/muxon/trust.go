package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/muxon/harness/internal/security"
)

var trustCmd = &cobra.Command{
	Use:   "trust",
	Short: "Manage workspace trust",
}

var trustListCmd = &cobra.Command{
	Use:   "list",
	Short: "List trusted workspace roots",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := security.LoadTrust()
		if err != nil {
			return err
		}
		for _, r := range s.Roots {
			fmt.Println(r)
		}
		return nil
	},
}

var trustAddCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Trust a workspace root",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := absPath(args[0])
		if err != nil {
			return err
		}
		s, err := security.LoadTrust()
		if err != nil {
			return err
		}
		s.AddRootWithConfigHash(root, security.HashMuxonToml(root))
		return security.SaveTrust(s)
	},
}

var trustRmCmd = &cobra.Command{
	Use:   "rm [path]",
	Short: "Remove a trusted workspace root",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := absPath(args[0])
		if err != nil {
			return err
		}
		s, err := security.LoadTrust()
		if err != nil {
			return err
		}
		s.RemoveRoot(root)
		return security.SaveTrust(s)
	},
}

func init() {
	trustCmd.AddCommand(trustListCmd, trustAddCmd, trustRmCmd)
	rootCmd.AddCommand(trustCmd)
}
