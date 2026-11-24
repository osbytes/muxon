package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/muxon/harness/internal/observability"
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Show session cost from trace",
	RunE: func(cmd *cobra.Command, args []string) error {
		setupLogging()
		dir, err := workspaceDir(cmd)
		if err != nil {
			return err
		}
		sum, path, err := observability.ReadLatestSessionCost(dir)
		if err != nil {
			return err
		}
		_, _ = cmd.Flags().GetBool("last")
		costStr := "$—"
		if sum.TotalCostUSD > 0 {
			costStr = fmt.Sprintf("$%.2f", sum.TotalCostUSD)
		}
		fmt.Printf("Session %s: %d in (%d cached) / %d out (%d reasoning)  %s\n",
			path, sum.InputTokens, sum.CachedTokens, sum.OutputTokens, sum.ReasoningTokens, costStr)
		return nil
	},
}

func init() {
	costCmd.Flags().Bool("last", false, "Show last goal block only")
	rootCmd.AddCommand(costCmd)
}
