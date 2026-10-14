package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/security"
	"github.com/PLACEHOLDER/thee/harness/internal/session"
	"github.com/PLACEHOLDER/thee/harness/internal/ui/tui"
)

var resumeCmd = &cobra.Command{
	Use:   "resume [session-id]",
	Short: "Resume an interactive session",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := workspaceDir(cmd)
		if err != nil {
			return err
		}
		id := ""
		if len(args) == 1 {
			id = args[0]
		} else {
			id, err = newestSession(dir)
			if err != nil {
				return err
			}
		}
		tracePath := filepath.Join(dir, ".thee", "sessions", id, "trace.jsonl")
		major, err := session.MajorSchema(tracePath)
		if err != nil {
			return err
		}
		if major > 1 {
			return exitError{code: 3, msg: fmt.Sprintf("trace schema %d needs a newer thee", major)}
		}
		manifest, err := config.LoadMergedManifest()
		if err != nil {
			return err
		}
		prov, _, err := loadRunProvider(manifest, "")
		if err != nil {
			return exitError{code: 3, msg: err.Error()}
		}
		worker := ""
		if md, err := manifest.ResolveRole("worker"); err == nil {
			worker = md.Model
		}
		eff, _ := config.LoadEffective(dir)
		return tui.Run(tui.Options{
			Workspace:   dir,
			SessionID:   id,
			Trusted:     security.WorkspaceTrusted(dir, false),
			Provider:    prov,
			Manifest:    manifest,
			Version:     version,
			WorkerModel: worker,
			Permission:  eff.Permission,
		})
	},
}

func newestSession(workspace string) (string, error) {
	dir := filepath.Join(workspace, ".thee", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	if len(names) == 0 {
		return "", fmt.Errorf("no sessions")
	}
	sort.Strings(names)
	return names[len(names)-1], nil
}

func init() {
	rootCmd.AddCommand(resumeCmd)
}
