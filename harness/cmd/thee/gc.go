package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/session"
)

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Prune old sessions and shadow refs",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := workspaceDir(cmd)
		if err != nil {
			return err
		}
		n, err := session.GC(dir, time.Now())
		if err != nil {
			return err
		}
		fmt.Printf("removed %d\n", n)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(gcCmd)
}
