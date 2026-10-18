package autodetection

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Engine struct {
	store    Store
	executor Executor
	mu       sync.Mutex
	running  atomic.Bool
}

func New(store Store, executor Executor) *Engine { return &Engine{store: store, executor: executor} }
func (e *Engine) Config(ctx context.Context) (Config, error) {
	c := Defaults()
	err := read(ctx, e.store, ConfigKey, &c)
	return c, err
}
func (e *Engine) Save(ctx context.Context, c Config) error {
	if err := Validate(c); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
	}
	for _, p := range c.Policies {
		if p.Enabled {
			if err := e.executor.Validate(ctx, p); err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidConfig, err)
			}
		}
	}
	c.Revision = fmt.Sprint(time.Now().UnixNano())
	return write(ctx, e.store, ConfigKey, c)
}
func (e *Engine) Runtime(ctx context.Context) (Runtime, error) {
	r := Runtime{History: []Run{}, Results: []Result{}, Attempts: map[string]time.Time{}, Failures: map[string]int{}}
	err := read(ctx, e.store, RuntimeKey, &r)
	r.Running = e.running.Load()
	if err != nil {
		return r, err
	}
	c, configErr := e.Config(ctx)
	if configErr != nil {
		return r, configErr
	}
	enabled := false
	for _, p := range c.Policies {
		enabled = enabled || p.Enabled
	}
	if !enabled {
		r.NextRunAt = nil
	} else {
		tick := time.Now().Truncate(time.Minute).Add(time.Minute)
		currentProgress := false
		for key := range r.Attempts {
			if strings.HasPrefix(key, c.Revision+"/") {
				currentProgress = true
				break
			}
		}
		if r.NextRunAt == nil || !currentProgress || r.NextRunAt.Before(tick) {
			r.NextRunAt = &tick
		} else {
			rounded := r.NextRunAt.Truncate(time.Minute)
			if rounded.Before(*r.NextRunAt) {
				rounded = rounded.Add(time.Minute)
			}
			r.NextRunAt = &rounded
		}
	}
	return r, nil
}

// Handles lets older runners yield matching accounts to the policy owner.
func (e *Engine) Handles(ctx context.Context, a Account) bool {
	c, err := e.Config(ctx)
	if err != nil {
		return false
	}
	for _, p := range c.Policies {
		if p.Enabled && Eligible(p, a) {
			return true
		}
	}
	return false
}
func (e *Engine) Run(ctx context.Context) error {
	if !e.mu.TryLock() {
		return nil
	}
	defer e.mu.Unlock()
	release, ok, err := e.store.Lease(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	defer release()
	c, err := e.Config(ctx)
	if err != nil {
		return err
	}
	enabled := false
	for _, p := range c.Policies {
		enabled = enabled || p.Enabled
	}
	if !enabled {
		return nil
	}
	r, err := e.Runtime(ctx)
	if err != nil {
		return err
	}
	if r.Standards == nil {
		r.Standards = map[string]string{}
	}
	if r.Attempts == nil {
		r.Attempts = map[string]time.Time{}
	}
	if r.Failures == nil {
		r.Failures = map[string]int{}
	}
	accounts, err := e.executor.List(ctx)
	if err != nil {
		return err
	}
	type job struct {
		p   Policy
		a   Account
		key string
	}
	jobs := []job{}
	valid := map[string]bool{}
	now := time.Now()
	var next time.Time
	for _, p := range c.Policies {
		if !p.Enabled {
			continue
		}
		for _, a := range accounts {
			if !Eligible(p, a) {
				continue
			}
			key := fmt.Sprintf("%s/%s/%d", c.Revision, p.Kind, a.ID)
			valid[key] = true
			due := r.Attempts[key].Add(time.Duration(p.IntervalMinutes) * time.Minute)
			if !due.After(now) {
				jobs = append(jobs, job{p, a, key})
			} else if next.IsZero() || due.Before(next) {
				next = due
			}
		}
	}
	for key := range r.Attempts {
		if !valid[key] {
			delete(r.Attempts, key)
			delete(r.Failures, key)
			delete(r.Standards, key)
		}
	}
	// Oldest attempts first; newly added accounts enter without starving retries.
	sort.SliceStable(jobs, func(i, j int) bool { return r.Attempts[jobs[i].key].Before(r.Attempts[jobs[j].key]) })
	unique := make([]job, 0, 20)
	seen := map[int64]bool{}
	for _, j := range jobs {
		if seen[j.a.ID] || len(unique) >= 20 {
			next = now.Add(time.Minute)
			continue
		}
		seen[j.a.ID] = true
		unique = append(unique, j)
	}
	jobs = unique
	if len(jobs) == 0 {
		if !next.IsZero() {
			r.NextRunAt = &next
		} else {
			r.NextRunAt = nil
		}
		return write(ctx, e.store, RuntimeKey, r)
	}
	e.running.Store(true)
	defer e.running.Store(false)
	run := Run{StartedAt: now}
	var wg sync.WaitGroup
	var resultMu sync.Mutex
	sem := make(chan struct{}, 4)
	for _, j := range jobs {
		if ctx.Err() != nil {
			break
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			continue
		}
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			defer func() { <-sem }()
			probeCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
			defer cancel()
			result := Result{AccountID: j.a.ID, AccountName: j.a.Name, Kind: j.p.Kind, Status: "skipped"}
			a, getErr := e.executor.Get(probeCtx, j.a.ID)
			probeConfig, probeConfigErr := e.Config(probeCtx)
			attempted := false
			action := false
			if getErr != nil {
				result.Message = getErr.Error()
			} else if probeConfigErr != nil || probeConfig.Revision != c.Revision {
				result.Message = "policy changed; detection skipped"
			} else if !Eligible(j.p, a) {
				result.Message = "account no longer matches policy"
			} else {
				attempted = true
				out, err := e.executor.Probe(probeCtx, j.p, a)
				if err != nil {
					out.Error = err.Error()
					out.Healthy = false
				}
				result.Status = "failed"
				result.Message = out.Error
				if out.Healthy {
					result.Status = "success"
				}
				if out.Downgraded {
					result.Message = "standard quality question failed"
				}
				// Re-read both policy and account after the upstream request. A changed
				// policy or a manual status/scheduling change invalidates pending actions.
				fresh, loadErr := e.executor.Get(probeCtx, a.ID)
				current, configErr := e.Config(probeCtx)
				stable := loadErr == nil && configErr == nil && current.Revision == c.Revision && fresh.Status == a.Status && fresh.Schedulable == a.Schedulable && Eligible(j.p, fresh)
				resultMu.Lock()
				failures := r.Failures[j.key]
				if out.Standard && r.Standards[j.key] != out.StandardKey {
					failures = 0
					r.Standards[j.key] = out.StandardKey
				}
				resultMu.Unlock()
				if out.Downgraded && out.Standard {
					failures++
				} else if out.Healthy && out.Standard {
					failures = 0
				}
				resultMu.Lock()
				r.Failures[j.key] = failures
				resultMu.Unlock()
				acts := Actions{}
				if stable && out.Healthy && (j.p.Kind != "quality" || out.Standard) && (!a.Downgraded || out.Standard) {
					acts = Actions{Recover: j.p.Recover, Activate: j.p.Activate, Resume: j.p.Resume}
				}
				if stable && out.Standard && out.Downgraded && j.p.PauseOnDowngrade && failures >= j.p.FailureThreshold {
					acts.Pause = true
				}
				if acts.Recover || acts.Activate || acts.Resume || acts.Pause {
					if applyErr := e.executor.Apply(probeCtx, fresh, acts); applyErr != nil {
						result.Status = "failed"
						result.Message = "apply account action: " + applyErr.Error()
					} else {
						action = true
						if acts.Pause {
							result.Message = "paused after consecutive quality failures"
						} else {
							result.Message = "configured recovery actions completed"
						}
					}
				} else if out.Healthy && !stable {
					result.Message = "account or policy changed; recovery skipped"
				} else if out.Healthy && a.Downgraded && !out.Standard {
					result.Message = "connection passed; quality recovery requires the standard question"
				}
			}
			result.FinishedAt = time.Now()
			resultMu.Lock()
			defer resultMu.Unlock()
			// Also back off skips to avoid an invalid model/account busy loop.
			r.Attempts[j.key] = result.FinishedAt
			if attempted {
				run.Checked++
			}
			switch result.Status {
			case "success":
				run.Succeeded++
				if action {
					run.Recovered++
				}
			case "failed":
				run.Failed++
			default:
				run.Skipped++
			}
			r.Results = append([]Result{result}, r.Results...)
			if len(r.Results) > 100 {
				r.Results = r.Results[:100]
			}
			due := result.FinishedAt.Add(time.Duration(j.p.IntervalMinutes) * time.Minute)
			if next.IsZero() || due.Before(next) {
				next = due
			}
			progressCtx, progressCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			if progressErr := write(progressCtx, e.store, RuntimeKey, r); progressErr != nil {
				run.Error = "persist detection progress: " + progressErr.Error()
			}
			progressCancel()
		}(j)
	}
	wg.Wait()
	run.FinishedAt = time.Now()
	if ctx.Err() != nil {
		run.Error = ctx.Err().Error()
		next = run.FinishedAt.Add(time.Minute)
	}
	r.LastRun = &run
	r.History = append([]Run{run}, r.History...)
	if len(r.History) > 20 {
		r.History = r.History[:20]
	}
	if !next.IsZero() {
		r.NextRunAt = &next
	}
	// Preserve completion even when the scan deadline expired.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return write(saveCtx, e.store, RuntimeKey, r)
}
