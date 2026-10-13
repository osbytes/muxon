package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check local setup (keys, providers, rg)",
	RunE: func(cmd *cobra.Command, args []string) error {
		setupLogging()
		m, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		rep := config.RunDoctor(m)
		for _, line := range rep.Lines {
			fmt.Println(line)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
