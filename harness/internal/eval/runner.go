package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PLACEHOLDER/muxon/harness/internal/config"
	"github.com/PLACEHOLDER/muxon/harness/internal/director"
	"github.com/PLACEHOLDER/muxon/harness/internal/observability"
	"github.com/PLACEHOLDER/muxon/harness/internal/provider"
	"github.com/PLACEHOLDER/muxon/harness/internal/router"
	"github.com/PLACEHOLDER/muxon/harness/internal/session"
	"github.com/PLACEHOLDER/muxon/harness/internal/subproc"
)

// TrialOptions controls one eval trial.
type TrialOptions struct {
	ModelID          string
	Provider         provider.Provider
	Manifest         *config.Manifest
	Trial            int
	RunID            string
	Overrides        map[string]string
	Timeout          time.Duration
	SessionBudgetUSD float64
	ResultsPath      string
	MuxonVersion      string
	Trusted          bool
	TaskDir          string
	FixtureRoots     []string
	Workspace        string
	Prompt           string
	KeepWorkspace    bool
	HarnessSHA       string
	ManifestHash     string
	ProviderName     string
}

// TrialResult is one finished trial.
type TrialResult struct {
	Row         ResultRow
	Outcome     director.Outcome
	Workspace   string
	CheckOutput string
}

// RunTrial copies the fixture into a fresh directory, runs the director, then grades
// with the hidden check. Reward comes from that check when one is configured.
func RunTrial(ctx context.Context, task Task, opt TrialOptions) (TrialResult, error) {
	opt = normalizeTrial(opt)
	manifest, err := trialManifest(&opt)
	if err != nil {
		return TrialResult{}, err
	}

	ownWorkspace := opt.Workspace == ""
	work := opt.Workspace
	if ownWorkspace {
		work, err = prepareWorkspace(task, opt)
		if err != nil {
			return TrialResult{}, err
		}
		if !opt.KeepWorkspace {
			defer os.RemoveAll(work)
		}
	}

	start := time.Now()
	runCtx, cancel := trialContext(ctx, opt.Timeout)
	defer cancel()
	out, runErr := director.Run(runCtx, trialConfig(task, opt, manifest, work))
	row := ResultRow{
		RunID:        opt.RunID,
		TaskID:       task.ID,
		Trial:        opt.Trial,
		ModelID:      modelID(manifest, opt.ModelID),
		Provider:     opt.ProviderName,
		HarnessSHA:   opt.HarnessSHA,
		ManifestHash: opt.ManifestHash,
		Overrides:    opt.Overrides,
		WallS:        time.Since(start).Seconds(),
		Reason:       out.Reason,
	}
	if sum, _, err := observability.ReadLatestSessionCost(work); err == nil {
		row.InputTokens = sum.InputTokens
		row.OutputTokens = sum.OutputTokens
		row.CachedTokens = sum.CachedTokens
		row.CostUSD = sum.TotalCostUSD
		row.ToolRounds = sum.ToolRounds
		row.Escalations = sum.Escalations
		row.MalformedToolCalls = sum.MalformedToolCalls
		row.LoopsDetected = sum.LoopsDetected
		row.SalvagedToolCalls = sum.SalvagedToolCalls
		row.PatchApplies = sum.PatchApplies
		row.WellFormedPatches = sum.WellFormedPatches
	}
	res := TrialResult{Row: row, Outcome: out, Workspace: work}
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		res.Row.TimeoutHit = true
		res.Row.Reward = 0
		res.Row.Reason = "timeout"
		return finishTrial(res, opt)
	}
	if runErr != nil {
		return res, runErr
	}
	if len(task.Check.Cmd) == 0 {
		if out.State == director.StateDone {
			res.Row.Reward = 1
		}
		return finishTrial(res, opt)
	}
	output, code, err := runCheck(context.Background(), work, task, opt.TaskDir)
	res.CheckOutput = string(output)
	if err != nil && code == 0 {
		return res, err
	}
	if code == 0 {
		res.Row.Reward = 1
	}
	return finishTrial(res, opt)
}

func normalizeTrial(opt TrialOptions) TrialOptions {
	if opt.Trial == 0 {
		opt.Trial = 1
	}
	if opt.HarnessSHA == "" {
		opt.HarnessSHA = HarnessSHA("")
	}
	return opt
}

func trialManifest(opt *TrialOptions) (*config.Manifest, error) {
	manifest := opt.Manifest
	var err error
	if manifest == nil {
		manifest, err = config.LoadMergedManifest()
		if err != nil {
			return nil, err
		}
	}
	manifest, err = applyOverrides(manifest, opt.ModelID, opt.Overrides)
	if err != nil {
		return nil, err
	}
	if opt.ManifestHash == "" {
		opt.ManifestHash = ManifestHash(manifest)
	}
	if opt.ProviderName == "" {
		opt.ProviderName = providerName(manifest, opt.ModelID)
	}
	return manifest, nil
}

func trialContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func trialConfig(task Task, opt TrialOptions, manifest *config.Manifest, work string) director.Config {
	caps := director.DefaultCaps()
	if opt.SessionBudgetUSD > 0 {
		caps.CostBudgetUSD = opt.SessionBudgetUSD
	}
	goal := task.Prompt
	if opt.Prompt != "" {
		goal = opt.Prompt
	}
	cfg := director.Config{
		Goal:                goal,
		Workspace:           work,
		Trusted:             opt.Trusted,
		MuxonVersion:         opt.MuxonVersion,
		Provider:            opt.Provider,
		Manifest:            manifest,
		SessionID:           session.NewID(),
		Caps:                caps,
		RegisterAllTools:    true,
		WorkerModelOverride: opt.ModelID,
	}
	if len(task.VerifyCmd) > 0 {
		cfg.VerifyCmd = strings.Join(task.VerifyCmd, " ")
	}
	return cfg
}

func finishTrial(res TrialResult, opt TrialOptions) (TrialResult, error) {
	if opt.ResultsPath != "" {
		if err := AppendResult(opt.ResultsPath, res.Row); err != nil {
			return res, err
		}
	}
	if !opt.KeepWorkspace && opt.Workspace == "" {
		res.Workspace = ""
	}
	return res, nil
}

func prepareWorkspace(task Task, opt TrialOptions) (string, error) {
	work, err := os.MkdirTemp("", "muxon-eval-"+task.ID+"-")
	if err != nil {
		return "", err
	}
	src, err := taskSource(task, opt)
	if err != nil {
		os.RemoveAll(work)
		return "", err
	}
	if src != "" {
		if _, err := copyTree(src, work); err != nil {
			os.RemoveAll(work)
			return "", err
		}
	}
	gitBaseline(work)
	return work, nil
}

func taskSource(task Task, opt TrialOptions) (string, error) {
	if task.Directory != "" {
		return findDir(task.Directory, append([]string{opt.TaskDir}, opt.FixtureRoots...))
	}
	if task.Fixture == "" {
		return "", fmt.Errorf("task %s needs fixture or directory", task.ID)
	}
	bases := fixtureCandidates(task.Fixture, opt.FixtureRoots)
	if opt.TaskDir != "" {
		bases = append([]string{
			filepath.Join(opt.TaskDir, "..", "fixtures"),
			filepath.Join(opt.TaskDir, "..", "..", "fixtures"),
		}, bases...)
	}
	return findDir(task.Fixture, bases)
}

func runCheck(ctx context.Context, work string, task Task, taskDir string) ([]byte, int, error) {
	var copied []string
	if task.Check.Files != "" {
		src, err := findDir(task.Check.Files, []string{taskDir, filepath.Join(taskDir, ".."), "."})
		if err != nil {
			return nil, 0, fmt.Errorf("task %s check files: %w", task.ID, err)
		}
		copied, err = copyTree(src, work)
		if err != nil {
			removePaths(copied)
			return nil, 0, err
		}
		defer removePaths(copied)
	}
	if len(task.Check.Cmd) == 0 {
		return nil, 1, fmt.Errorf("task %s: check.cmd is empty", task.ID)
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return subproc.Run(checkCtx, work, task.Check.Cmd, nil)
}

func applyOverrides(m *config.Manifest, modelID string, ov map[string]string) (*config.Manifest, error) {
	if len(ov) == 0 {
		return m, nil
	}
	cp := *m
	cp.Models = append([]config.ModelDef(nil), m.Models...)
	if modelID == "" {
		if w, err := cp.ResolveRole("worker"); err == nil {
			modelID = w.ID
		}
	}
	md, err := cp.ResolveModel(modelID)
	if err != nil {
		return nil, err
	}
	for k, v := range ov {
		switch k {
		case "tool_protocol":
			md.Capabilities.ToolProtocol = v
		case "prompt_variant":
			md.Capabilities.PromptVariant = v
		case "edit_format":
			md.Capabilities.EditFormat = v
		case "structured_output":
			md.Capabilities.StructuredOut = v
		default:
			return nil, fmt.Errorf("unknown capability override %q", k)
		}
	}
	return &cp, nil
}

func modelID(m *config.Manifest, override string) string {
	if override != "" {
		return override
	}
	if w, err := m.ResolveRole("worker"); err == nil {
		return w.ID
	}
	return "worker"
}

func providerName(m *config.Manifest, modelID string) string {
	id := modelID
	if id == "" {
		if w, err := m.ResolveRole("worker"); err == nil {
			id = w.ID
		}
	}
	md, err := m.ResolveModel(id)
	if err != nil {
		return ""
	}
	return md.Provider
}

// HarnessSHA is the git commit of dir, or of the current directory, with -dirty
// when the tree has uncommitted changes.
func HarnessSHA(dir string) string {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		return "unknown"
	}
	sha := strings.TrimSpace(string(out))
	dirty, _ := exec.Command("git", "-C", dir, "status", "--porcelain").Output()
	if len(strings.TrimSpace(string(dirty))) > 0 {
		sha += "-dirty"
	}
	return sha
}

// ManifestHash is a short hash of the manifest JSON.
func ManifestHash(m *config.Manifest) string {
	b, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// NewProvider builds a live provider for a manifest model id.
func NewProvider(m *config.Manifest, modelID string) (provider.Provider, error) {
	bind, err := router.ResolveWorker(m, modelID)
	if err != nil {
		return nil, err
	}
	return router.NewProvider(bind)
}
