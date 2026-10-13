package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/director"
	"github.com/PLACEHOLDER/thee/harness/internal/eval"
	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/session"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Run eval tasks",
}

var evalRunCmd = &cobra.Command{
	Use:   "run [task.json]",
	Short: "Run one eval task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		setupLogging()
		task, err := eval.LoadTask(args[0])
		if err != nil {
			return err
		}
		workDir, err := evalWorkspace(task)
		if err != nil {
			return err
		}
		trust, _ := cmd.Flags().GetBool("trust")
		manifest, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		prov, modelID, err := loadRunProvider(manifest, "")
		if err != nil {
			return exitError{code: 3, msg: err.Error()}
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		start := time.Now()
		cfg := director.Config{
			Goal:             task.Prompt,
			Workspace:        workDir,
			Trusted:          trust,
			TheeVersion:      version,
			Provider:         prov,
			Manifest:         manifest,
			SessionID:        session.NewID(),
			RegisterAllTools: true,
		}
		if len(task.VerifyCmd) > 0 {
			cfg.VerifyCmd = strings.Join(task.VerifyCmd, " ")
		}
		out, err := director.Run(ctx, cfg)
		if err != nil {
			return err
		}
		sum, _, _ := observability.ReadLatestSessionCost(workDir)
		reward := 0
		if out.State == director.StateDone {
			reward = 1
		}
		_ = eval.AppendResult(workDir, eval.ResultRow{
			TaskID:       task.ID,
			Trial:        1,
			Reward:       reward,
			ModelID:      modelID,
			InputTokens:  sum.InputTokens,
			OutputTokens: sum.OutputTokens,
			CachedTokens: sum.CachedTokens,
			CostUSD:      sum.TotalCostUSD,
			WallS:        time.Since(start).Seconds(),
		})
		if out.ExitCode != 0 {
			return exitError{code: mapExitCode(out), msg: out.Reason}
		}
		return nil
	},
}

func init() {
	evalRunCmd.Flags().Bool("trust", false, "Trust workspace for verify")
	evalCmd.AddCommand(evalRunCmd)
	rootCmd.AddCommand(evalCmd)
}

func evalWorkspace(task eval.Task) (string, error) {
	if task.Directory != "" {
		return absPath(task.Directory)
	}
	if task.Fixture == "" {
		return "", fmt.Errorf("task needs fixture or directory")
	}
	src := fixtureSourcePath(task.Fixture)
	if src == "" {
		return "", fmt.Errorf("fixture %q not found", task.Fixture)
	}
	dest := filepath.Join(os.TempDir(), "thee-eval-"+task.ID)
	if err := copyDir(src, dest); err != nil {
		return "", err
	}
	return dest, nil
}

func fixtureSourcePath(name string) string {
	candidates := []string{
		filepath.Join("testdata", "fixtures", name),
		filepath.Join("..", "testdata", "fixtures", name),
		filepath.Join("..", "..", "testdata", "fixtures", name),
	}
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// runEvalTask runs a task with an explicit provider (tests).
func runEvalTask(ctx context.Context, task eval.Task, workDir string, prov provider.Provider, trusted bool) (director.Outcome, error) {
	manifest, err := config.LoadMergedManifest()
	if err != nil {
		return director.Outcome{}, err
	}
	start := time.Now()
	cfg := director.Config{
		Goal:             task.Prompt,
		Workspace:        workDir,
		Trusted:          trusted,
		TheeVersion:      version,
		Provider:         prov,
		Manifest:         manifest,
		SessionID:        session.NewID(),
		RegisterAllTools: true,
	}
	if len(task.VerifyCmd) > 0 {
		cfg.VerifyCmd = strings.Join(task.VerifyCmd, " ")
	}
	out, err := director.Run(ctx, cfg)
	if err != nil {
		return out, err
	}
	sum, _, _ := observability.ReadLatestSessionCost(workDir)
	reward := 0
	if out.State == director.StateDone {
		reward = 1
	}
	modelID := "worker"
	if w, err := manifest.ResolveRole("worker"); err == nil {
		modelID = w.ID
	}
	_ = eval.AppendResult(workDir, eval.ResultRow{
		TaskID:       task.ID,
		Trial:        1,
		Reward:       reward,
		ModelID:      modelID,
		InputTokens:  sum.InputTokens,
		OutputTokens: sum.OutputTokens,
		CachedTokens: sum.CachedTokens,
		CostUSD:      sum.TotalCostUSD,
		WallS:        time.Since(start).Seconds(),
	})
	return out, nil
}
