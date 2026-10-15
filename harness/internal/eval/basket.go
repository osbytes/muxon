package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/PLACEHOLDER/thee/harness/internal/config"
	"github.com/PLACEHOLDER/thee/harness/internal/provider"
	"github.com/PLACEHOLDER/thee/harness/internal/session"
)

// ModelSpec is one row of the model matrix.
type ModelSpec struct {
	ModelID   string            `yaml:"model_id" json:"model_id"`
	Overrides map[string]string `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Provider  provider.Provider `yaml:"-" json:"-"`
}

// Matrix is a models list for a basket run.
type Matrix struct {
	Models []ModelSpec `yaml:"models"`
}

// BasketOptions controls a multi-task, multi-model run.
type BasketOptions struct {
	Trials         int
	Concurrency    int
	BudgetUSD      float64
	OutputDir      string
	TheeVersion    string
	Trusted        bool
	Dataset        string
	DatasetVersion string
	RunID          string
	ProviderFor    func(modelID string) (provider.Provider, error)
	Manifest       *config.Manifest
	Progress       *TrialProgress
}

// RunHeader is the run.json written next to results.jsonl.
type RunHeader struct {
	RunID          string    `json:"run_id"`
	Dataset        string    `json:"dataset"`
	DatasetVersion string    `json:"dataset_version"`
	Models         []string  `json:"models"`
	Trials         int       `json:"trials"`
	HarnessSHA     string    `json:"harness_sha"`
	ManifestHash   string    `json:"manifest_hash"`
	StartedAt      time.Time `json:"started_at"`
	BudgetUSD      float64   `json:"budget_usd,omitempty"`
	StoppedReason  string    `json:"stopped_reason,omitempty"`
}

// LoadMatrix reads a matrix YAML file.
func LoadMatrix(path string) (Matrix, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Matrix{}, err
	}
	var m Matrix
	if err := yaml.Unmarshal(b, &m); err != nil {
		return Matrix{}, err
	}
	if len(m.Models) == 0 {
		return Matrix{}, fmt.Errorf("matrix %s has no models", path)
	}
	return m, nil
}

// RunBasket runs every task for every model, for Trials repetitions.
func RunBasket(ctx context.Context, tasks []PreparedTask, models []ModelSpec, opt BasketOptions) (RunHeader, []ResultRow, error) {
	opt = normalizeBasket(opt)
	manifest, models, err := basketManifest(opt, models)
	if err != nil {
		return RunHeader{}, nil, err
	}
	header := newRunHeader(opt, manifest, models)
	runDir := filepath.Join(opt.OutputDir, "runs", opt.RunID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return header, nil, err
	}
	resultsPath := filepath.Join(runDir, "results.jsonl")
	if err := writeHeader(filepath.Join(runDir, "run.json"), header); err != nil {
		return header, nil, err
	}

	jobs := basketJobs(models, tasks, opt.Trials)
	if opt.Progress != nil {
		opt.Progress.SetTotal(len(jobs))
		opt.Progress.Banner(opt.RunID, len(jobs))
	}
	state := &basketState{
		header:      &header,
		opt:         opt,
		manifest:    manifest,
		resultsPath: resultsPath,
	}
	jobCh := make(chan basketJob)
	go state.produce(ctx, jobs, jobCh)
	workers := opt.Concurrency
	if workers > len(jobs) {
		workers = len(jobs)
	}
	if workers < 1 {
		workers = 1
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			state.work(ctx, jobCh)
		}()
	}
	wg.Wait()
	if err := writeHeader(filepath.Join(runDir, "run.json"), header); err != nil && state.runErr == nil {
		state.runErr = err
	}
	return header, state.rows, state.runErr
}

type basketJob struct {
	task  PreparedTask
	model ModelSpec
	trial int
}

type basketState struct {
	mu          sync.Mutex
	spent       float64
	stopped     bool
	rows        []ResultRow
	runErr      error
	header      *RunHeader
	opt         BasketOptions
	manifest    *config.Manifest
	resultsPath string
}

func normalizeBasket(opt BasketOptions) BasketOptions {
	if opt.Trials <= 0 {
		opt.Trials = 1
	}
	if opt.Concurrency <= 0 {
		opt.Concurrency = 1
	}
	if opt.Dataset == "" {
		opt.Dataset = "thee-basket"
	}
	if opt.DatasetVersion == "" {
		opt.DatasetVersion = "1"
	}
	if opt.RunID == "" {
		opt.RunID = session.NewID()
	}
	if opt.OutputDir == "" {
		opt.OutputDir = filepath.Join(".thee", "eval")
	}
	return opt
}

func basketManifest(opt BasketOptions, models []ModelSpec) (*config.Manifest, []ModelSpec, error) {
	manifest := opt.Manifest
	var err error
	if manifest == nil {
		manifest, err = config.LoadMergedManifest()
		if err != nil {
			return nil, nil, err
		}
	}
	if len(models) == 0 {
		id := ""
		if w, err := manifest.ResolveRole("worker"); err == nil {
			id = w.ID
		}
		models = []ModelSpec{{ModelID: id}}
	}
	return manifest, models, nil
}

func newRunHeader(opt BasketOptions, manifest *config.Manifest, models []ModelSpec) RunHeader {
	header := RunHeader{
		RunID:          opt.RunID,
		Dataset:        opt.Dataset,
		DatasetVersion: opt.DatasetVersion,
		Trials:         opt.Trials,
		HarnessSHA:     HarnessSHA(""),
		ManifestHash:   ManifestHash(manifest),
		StartedAt:      time.Now().UTC(),
		BudgetUSD:      opt.BudgetUSD,
	}
	for _, m := range models {
		header.Models = append(header.Models, m.ModelID)
	}
	return header
}

func basketJobs(models []ModelSpec, tasks []PreparedTask, trials int) []basketJob {
	var jobs []basketJob
	for _, model := range models {
		for _, task := range tasks {
			for trial := 1; trial <= trials; trial++ {
				jobs = append(jobs, basketJob{task: task, model: model, trial: trial})
			}
		}
	}
	return jobs
}

func (s *basketState) overBudget() bool {
	return s.opt.BudgetUSD > 0 && s.spent >= s.opt.BudgetUSD
}

func (s *basketState) produce(ctx context.Context, jobs []basketJob, jobCh chan<- basketJob) {
	defer close(jobCh)
	for _, j := range jobs {
		s.mu.Lock()
		stop := s.stopped || s.overBudget() || ctx.Err() != nil
		if stop && s.opt.BudgetUSD > 0 && (s.stopped || s.overBudget()) {
			s.header.StoppedReason = "budget"
		}
		s.mu.Unlock()
		if stop {
			return
		}
		select {
		case <-ctx.Done():
			return
		case jobCh <- j:
		}
	}
}

func (s *basketState) work(ctx context.Context, jobCh <-chan basketJob) {
	for j := range jobCh {
		remain, skip := s.claim()
		if skip {
			continue
		}
		idx := s.opt.Progress.Start(j.task.Task.ID, j.model.ModelID, j.trial)
		res, err := s.runJob(ctx, j, remain)
		s.finish(res, err)
		if err != nil && res.Row.TaskID == "" {
			s.opt.Progress.Error(idx, j.task.Task.ID, j.model.ModelID, j.trial, err)
			continue
		}
		s.opt.Progress.Done(idx, res.Row)
	}
}

func (s *basketState) claim() (remain float64, skip bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.overBudget() {
		s.stopped = true
		s.header.StoppedReason = "budget"
		return 0, true
	}
	if s.opt.BudgetUSD > 0 {
		remain = s.opt.BudgetUSD - s.spent
	}
	return remain, false
}

func (s *basketState) runJob(ctx context.Context, j basketJob, remain float64) (TrialResult, error) {
	prov := j.model.Provider
	if s.opt.ProviderFor != nil {
		p, err := s.opt.ProviderFor(j.model.ModelID)
		if err != nil {
			return TrialResult{}, err
		}
		prov = p
	}
	return RunTrial(ctx, j.task.Task, TrialOptions{
		ModelID:          j.model.ModelID,
		Provider:         prov,
		Manifest:         s.manifest,
		Trial:            j.trial,
		RunID:            s.opt.RunID,
		Overrides:        j.model.Overrides,
		Timeout:          time.Duration(j.task.Task.TimeoutS) * time.Second,
		SessionBudgetUSD: remain,
		TheeVersion:      s.opt.TheeVersion,
		Trusted:          s.opt.Trusted,
		TaskDir:          j.task.TaskDir,
		HarnessSHA:       s.header.HarnessSHA,
		ManifestHash:     s.header.ManifestHash,
	})
}

func (s *basketState) finish(res TrialResult, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil && s.runErr == nil {
		s.runErr = err
	}
	if res.Row.TaskID == "" {
		return
	}
	s.rows = append(s.rows, res.Row)
	s.spent += res.Row.CostUSD
	if s.overBudget() {
		s.stopped = true
		s.header.StoppedReason = "budget"
	}
	_ = AppendResult(s.resultsPath, res.Row)
}

func writeHeader(path string, header RunHeader) error {
	b, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// LatestRunID returns the lexicographically last run directory name.
func LatestRunID(outputDir string) (string, error) {
	dir := filepath.Join(outputDir, "runs")
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
		return "", fmt.Errorf("no eval runs in %s", dir)
	}
	sort.Strings(names)
	return names[len(names)-1], nil
}
