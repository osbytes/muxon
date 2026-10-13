package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/director"
	"github.com/PLACEHOLDER/thee/harness/internal/eval"
	"github.com/PLACEHOLDER/thee/harness/internal/observability"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/provider/scripted"
	"github.com/PLACEHOLDER/thee/harness/internal/router"
	"github.com/PLACEHOLDER/thee/harness/internal/session"
	"github.com/PLACEHOLDER/thee/harness/internal/ui/headless"
)

type exitError struct {
	code int
	msg  string
}

func (e exitError) Error() string { return e.msg }

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a headless agent session",
	RunE: func(cmd *cobra.Command, args []string) error {
		setupLogging()
		prompt, _ := cmd.Flags().GetString("prompt")
		trust, _ := cmd.Flags().GetBool("trust")
		if os.Getenv("THEE_TRUST") == "1" {
			trust = true
		}
		verifyCmd, _ := cmd.Flags().GetString("verify-cmd")
		permission, _ := cmd.Flags().GetString("permission")
		yolo, _ := cmd.Flags().GetBool("yolo")
		model, _ := cmd.Flags().GetString("model")
		noPlan, _ := cmd.Flags().GetBool("no-plan")
		evalPath, _ := cmd.Flags().GetString("eval")
		maxRounds, _ := cmd.Flags().GetInt("max-tool-rounds")
		costBudget, _ := cmd.Flags().GetFloat64("cost-budget-usd")

		if prompt == "" {
			return exitError{code: 3, msg: "missing -p/--prompt"}
		}
		dir, err := workspaceDir(cmd)
		if err != nil {
			return err
		}

		mode := headless.ModeHuman
		if flagJSON {
			mode = headless.ModeJSON
		}
		if flagQuiet {
			mode = headless.ModeQuiet
		}
		present := headless.NewPresenter(mode)

		manifest, err := config.LoadMergedManifest()
		if err != nil {
			return exitError{code: 3, msg: err.Error()}
		}
		prov, workerModelID, err := loadRunProvider(manifest, model)
		if err != nil {
			return exitError{code: 3, msg: err.Error()}
		}

		eff, _ := config.LoadEffective(dir)
		perm := eff.Permission
		if permission != "" {
			perm = permission
		}
		if yolo {
			perm = eff.MaxPermission
		}
		_ = perm

		caps := director.DefaultCaps()
		if maxRounds > 0 {
			caps.MaxToolRounds = maxRounds
		}
		if costBudget > 0 {
			caps.CostBudgetUSD = costBudget
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		start := time.Now()
		sid := session.NewID()
		out, err := director.Run(ctx, director.Config{
			Goal:                prompt,
			Workspace:           dir,
			Trusted:             trust,
			TheeVersion:         version,
			Provider:            prov,
			Manifest:            manifest,
			SessionID:           sid,
			VerifyCmd:           verifyCmd,
			WorkerModelOverride: model,
			NoPlan:              noPlan,
			Caps:                caps,
			RegisterAllTools:    true,
			OnEvent:             present.Handle,
		})
		if err != nil {
			return err
		}

		if mode == headless.ModeQuiet {
			fmt.Println(present.FinalLine())
		}

		if evalPath != "" {
			_ = appendEvalRow(dir, evalPath, workerModelID, out, start)
		}

		if out.ExitCode != 0 {
			return exitError{code: mapExitCode(out), msg: fmt.Sprintf("%s: %s", out.State, out.Reason)}
		}
		return nil
	},
}

func init() {
	runCmd.Flags().StringP("prompt", "p", "", "Goal prompt")
	runCmd.Flags().Bool("trust", false, "Trust workspace for verify")
	runCmd.Flags().String("verify-cmd", "", "Override verify command (argv string)")
	runCmd.Flags().String("permission", "", "Permission tier: read-only, edit, full")
	runCmd.Flags().Bool("yolo", false, "Auto-approve within max_permission")
	runCmd.Flags().String("model", "", "Force worker model id")
	runCmd.Flags().Bool("no-plan", false, "Skip planning phase")
	runCmd.Flags().String("eval", "", "Append eval result row for task file")
	runCmd.Flags().Int("max-tool-rounds", 0, "Override max tool rounds")
	runCmd.Flags().Float64("cost-budget-usd", 0, "Override session cost budget")
}

func loadRunProvider(manifest *config.Manifest, modelOverride string) (provider.Provider, string, error) {
	if path := os.Getenv("THEE_SCRIPTED_PROVIDER_FILE"); path != "" {
		p, err := scripted.FromEnv()
		if err != nil {
			return nil, "", err
		}
		w, err := router.ResolveWorker(manifest, modelOverride)
		if err != nil {
			return nil, "", err
		}
		return p, w.Model.ID, nil
	}
	p, bind, err := router.NewProviderForRole(manifest, "worker", modelOverride)
	if err != nil {
		return nil, "", err
	}
	return p, bind.Model.ID, nil
}

func mapExitCode(out director.Outcome) int {
	if out.Reason == "approval_required" {
		return 2
	}
	if out.ExitCode != 0 {
		return out.ExitCode
	}
	return 1
}

func appendEvalRow(workspace, taskPath, modelID string, out director.Outcome, start time.Time) error {
	task, err := eval.LoadTask(taskPath)
	if err != nil {
		return err
	}
	sum, _, _ := observability.ReadLatestSessionCost(workspace)
	reward := 0
	if out.State == director.StateDone {
		reward = 1
	}
	return eval.AppendResult(workspace, eval.ResultRow{
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
}

func absPath(dir string) (string, error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	return filepath.Abs(dir)
}
