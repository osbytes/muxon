package main

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/replay"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
)

var traceCmd = &cobra.Command{
	Use:   "trace",
	Short: "Record, replay, or export session traces",
}

var traceRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Write a replay file from the scripted provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		out, _ := cmd.Flags().GetString("output")
		if out == "" {
			return fmt.Errorf("missing --output")
		}
		p, err := scripted.FromEnv()
		if err != nil {
			return err
		}
		var steps []replay.Step
		for {
			resp, err := p.Complete(cmd.Context(), provider.Request{})
			if err != nil {
				break
			}
			steps = append(steps, replay.Step{Response: resp})
		}
		return replay.WriteFile(out, steps)
	},
}

var traceReplayCmd = &cobra.Command{
	Use:   "replay [file]",
	Short: "Load a replay file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := replay.Load(args[0])
		if err != nil {
			return err
		}
		n := 0
		for {
			_, err := p.Complete(cmd.Context(), provider.Request{})
			if err != nil {
				break
			}
			n++
		}
		fmt.Printf("replay loaded %d steps\n", n)
		return nil
	},
}

var traceExportCmd = &cobra.Command{
	Use:   "export [session-id]",
	Short: "Export a redacted trace",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := workspaceDir(cmd)
		if err != nil {
			return err
		}
		dest, _ := cmd.Flags().GetString("output")
		if dest == "" {
			return fmt.Errorf("missing --output")
		}
		src := filepath.Join(dir, ".thee", "sessions", args[0], "trace.jsonl")
		return observability.ExportRedacted(src, dest)
	},
}

func init() {
	traceRecordCmd.Flags().String("output", "", "Replay JSON path")
	traceExportCmd.Flags().String("output", "", "Export path")
	traceCmd.AddCommand(traceRecordCmd, traceReplayCmd, traceExportCmd)
	rootCmd.AddCommand(traceCmd)
}
