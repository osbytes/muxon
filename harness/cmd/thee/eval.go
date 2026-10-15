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
	"github.com/PLACEHOLDER/thee/harness/internal/eval"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
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
		prepared, err := eval.LoadPrepared(args[0])
		if err != nil {
			return err
		}
		trust, _ := cmd.Flags().GetBool("trust")
		modelID, _ := cmd.Flags().GetString("model")
		timeoutS, _ := cmd.Flags().GetInt("timeout")
		outDir, _ := cmd.Flags().GetString("out")
		manifest, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		prov, resolved, err := loadRunProvider(manifest, modelID)
		if err != nil {
			return exitError{code: 3, msg: err.Error()}
		}
		if modelID == "" {
			modelID = resolved
		}
		progress := eval.NewTrialProgress(cmd.ErrOrStderr())
		progress.SetTotal(1)
		idx := progress.Start(prepared.Task.ID, modelID, 1)
		runID := evalRunID(cmd)
		timeout := time.Duration(timeoutS) * time.Second
		if timeout == 0 && prepared.Task.TimeoutS > 0 {
			timeout = time.Duration(prepared.Task.TimeoutS) * time.Second
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		res, err := eval.RunTrial(ctx, prepared.Task, eval.TrialOptions{
			ModelID:     modelID,
			Provider:    prov,
			Manifest:    manifest,
			Trial:       1,
			RunID:       runID,
			Timeout:     timeout,
			ResultsPath: filepath.Join(outDir, "runs", runID, "results.jsonl"),
			TheeVersion: version,
			Trusted:     trust,
			TaskDir:     prepared.TaskDir,
		})
		if err != nil && res.Row.TaskID == "" {
			progress.Error(idx, prepared.Task.ID, modelID, 1, err)
			return err
		}
		progress.Done(idx, res.Row)
		if err != nil {
			return err
		}
		if resolved != "" && res.Row.ModelID == "" {
			res.Row.ModelID = resolved
		}
		fmt.Printf("reward %d task %s run %s\n", res.Row.Reward, res.Row.TaskID, runID)
		if res.Row.Reward == 0 {
			return exitError{code: 1, msg: res.Row.Reason}
		}
		return nil
	},
}

var evalBasketCmd = &cobra.Command{
	Use:   "basket",
	Short: "Run the task basket across a model matrix",
	RunE: func(cmd *cobra.Command, args []string) error {
		setupLogging()
		pattern, _ := cmd.Flags().GetString("tasks")
		if pattern == "" {
			pattern = defaultTaskGlob()
		}
		tasks, err := eval.LoadTasks(pattern)
		if err != nil {
			return err
		}
		matrixPath, _ := cmd.Flags().GetString("matrix")
		var models []eval.ModelSpec
		if matrixPath != "" {
			matrix, err := eval.LoadMatrix(matrixPath)
			if err != nil {
				return err
			}
			models = matrix.Models
		}
		trials, _ := cmd.Flags().GetInt("trials")
		conc, _ := cmd.Flags().GetInt("concurrency")
		budget, _ := cmd.Flags().GetFloat64("budget-usd")
		trust, _ := cmd.Flags().GetBool("trust")
		outDir, _ := cmd.Flags().GetString("out")
		manifest, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		header, rows, err := eval.RunBasket(ctx, tasks, models, eval.BasketOptions{
			Trials:      trials,
			Concurrency: conc,
			BudgetUSD:   budget,
			OutputDir:   outDir,
			TheeVersion: version,
			Trusted:     trust,
			Manifest:    manifest,
			Progress:    eval.NewTrialProgress(cmd.ErrOrStderr()),
			ProviderFor: func(modelID string) (provider.Provider, error) {
				p, _, err := loadRunProvider(manifest, modelID)
				return p, err
			},
		})
		if err != nil {
			return err
		}
		fmt.Printf("run %s rows %d\n", header.RunID, len(rows))
		fmt.Print(eval.FormatGroupedReport(rows))
		return nil
	},
}

var evalReportCmd = &cobra.Command{
	Use:   "report [run-id]",
	Short: "Summarize an eval run",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, _ := cmd.Flags().GetString("out")
		arg := ""
		if len(args) == 1 {
			arg = args[0]
		}
		path, err := resolveResults(outDir, arg)
		if err != nil {
			return err
		}
		rows, err := eval.LoadResults(path)
		if err != nil {
			return err
		}
		fmt.Print(eval.FormatGroupedReport(rows))
		return nil
	},
}

var evalCompareCmd = &cobra.Command{
	Use:   "compare [run-a] [run-b]",
	Short: "Pair two runs and report a McNemar p-value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		outDir, _ := cmd.Flags().GetString("out")
		leftPath, err := resolveResults(outDir, args[0])
		if err != nil {
			return err
		}
		rightPath, err := resolveResults(outDir, args[1])
		if err != nil {
			return err
		}
		left, err := eval.LoadResults(leftPath)
		if err != nil {
			return err
		}
		right, err := eval.LoadResults(rightPath)
		if err != nil {
			return err
		}
		fmt.Print(eval.CompareRows(left, right))
		return nil
	},
}

var evalExportCmd = &cobra.Command{
	Use:   "export-swebench",
	Short: "Append a SWE-bench predictions.jsonl line from the workspace diff",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := workspaceDir(cmd)
		if err != nil {
			return err
		}
		instance, _ := cmd.Flags().GetString("instance")
		modelName, _ := cmd.Flags().GetString("model")
		out, _ := cmd.Flags().GetString("out")
		if out == "" {
			out = "predictions.jsonl"
		}
		return eval.ExportSWEBench(dir, instance, modelName, out)
	},
}

func init() {
	evalRunCmd.Flags().Bool("trust", false, "Trust workspace for verify")
	evalRunCmd.Flags().String("model", "", "Worker model id")
	evalRunCmd.Flags().Int("timeout", 0, "Trial timeout in seconds")
	evalRunCmd.Flags().String("out", filepath.Join(".thee", "eval"), "Directory for run results")
	evalRunCmd.Flags().String("run-id", "", "Run id (default: generated)")

	evalBasketCmd.Flags().String("matrix", "", "Matrix YAML of model ids and capability overrides")
	evalBasketCmd.Flags().String("tasks", "", "Glob of task JSON files")
	evalBasketCmd.Flags().Int("trials", 3, "Trials per task")
	evalBasketCmd.Flags().Int("concurrency", 1, "Trials to run at once")
	evalBasketCmd.Flags().Float64("budget-usd", 0, "Stop scheduling once spent cost reaches this many dollars")
	evalBasketCmd.Flags().Bool("trust", true, "Trust fixture workspaces")
	evalBasketCmd.Flags().String("out", filepath.Join(".thee", "eval"), "Directory for run results")

	evalReportCmd.Flags().String("out", filepath.Join(".thee", "eval"), "Directory for run results")
	evalCompareCmd.Flags().String("out", filepath.Join(".thee", "eval"), "Directory for run results")

	evalExportCmd.Flags().String("instance", "", "SWE-bench instance id")
	evalExportCmd.Flags().String("model", "", "model_name_or_path written into the prediction")
	evalExportCmd.Flags().String("out", "predictions.jsonl", "Predictions JSONL path")
	_ = evalExportCmd.MarkFlagRequired("instance")
	_ = evalExportCmd.MarkFlagRequired("model")

	evalCmd.AddCommand(evalRunCmd, evalBasketCmd, evalReportCmd, evalCompareCmd, evalExportCmd)
	rootCmd.AddCommand(evalCmd)
}

func evalRunID(cmd *cobra.Command) string {
	if cmd.Flags().Lookup("run-id") != nil {
		if id, err := cmd.Flags().GetString("run-id"); err == nil && id != "" {
			return id
		}
	}
	return fmt.Sprintf("run-%d", time.Now().UTC().Unix())
}

func defaultTaskGlob() string {
	candidates := []string{
		filepath.Join("testdata", "eval", "*.json"),
		filepath.Join("harness", "testdata", "eval", "*.json"),
	}
	for _, c := range candidates {
		matches, _ := filepath.Glob(c)
		if len(matches) > 0 {
			return c
		}
	}
	return filepath.Join("testdata", "eval", "*.json")
}

func resolveResults(outDir, arg string) (string, error) {
	if arg != "" {
		if st, err := os.Stat(arg); err == nil && !st.IsDir() {
			return arg, nil
		}
		p := filepath.Join(outDir, "runs", arg, "results.jsonl")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("run %s not found", arg)
	}
	id, err := eval.LatestRunID(outDir)
	if err != nil {
		legacy := filepath.Join(outDir, "results.jsonl")
		if _, statErr := os.Stat(legacy); statErr == nil {
			return legacy, nil
		}
		return "", err
	}
	return filepath.Join(outDir, "runs", id, "results.jsonl"), nil
}
