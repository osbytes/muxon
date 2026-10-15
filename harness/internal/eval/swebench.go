package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SWEBenchPrediction is one line of the official SWE-bench predictions file.
type SWEBenchPrediction struct {
	InstanceID string `json:"instance_id"`
	ModelName  string `json:"model_name_or_path"`
	ModelPatch string `json:"model_patch"`
}

// ExportSWEBench writes a predictions.jsonl line from the workspace git diff.
// swebench.harness.run_evaluation consumes that file.
func ExportSWEBench(workspace, instanceID, modelName, outPath string) error {
	if instanceID == "" {
		return fmt.Errorf("missing instance id")
	}
	if modelName == "" {
		return fmt.Errorf("missing model name")
	}
	patch, err := workspacePatch(workspace)
	if err != nil {
		return err
	}
	row := SWEBenchPrediction{
		InstanceID: instanceID,
		ModelName:  modelName,
		ModelPatch: patch,
	}
	b, err := json.Marshal(row)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil && filepath.Dir(outPath) != "." {
		return err
	}
	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func workspacePatch(workspace string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git is required to export a patch")
	}
	add := exec.Command("git", "-C", workspace, "add", "-N", "--", ".")
	if out, err := add.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add -N: %w: %s", err, out)
	}
	diff := exec.Command("git", "-C", workspace, "diff", "HEAD")
	out, err := diff.Output()
	reset := exec.Command("git", "-C", workspace, "reset", "-q")
	_ = reset.Run()
	if err != nil {
		return "", fmt.Errorf("git diff: %w", err)
	}
	return string(bytes.TrimSpace(out)) + "\n", nil
}
