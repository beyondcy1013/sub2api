// Package autodetection owns periodic account detection policies. Core services
// are supplied through ports; this module never imports the legacy service layer.
package autodetection

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"
)

const ConfigKey = "account_auto_detection_policies"
const RuntimeKey = "account_auto_detection_runtime"

type Policy struct {
	Kind             string   `json:"kind"`
	Enabled          bool     `json:"enabled"`
	AccountScope     string   `json:"account_scope"`
	AccountIDs       []int64  `json:"account_ids"`
	Platforms        []string `json:"platforms"`
	Statuses         []string `json:"statuses"`
	IncludePaused    bool     `json:"include_paused"`
	IntervalMinutes  int      `json:"interval_minutes"`
	ModelID          string   `json:"model_id"`
	Recover          bool     `json:"recover"`
	Activate         bool     `json:"activate"`
	Resume           bool     `json:"resume"`
	PauseOnDowngrade bool     `json:"pause_on_downgrade"`
	FailureThreshold int      `json:"failure_threshold"`
}
type Config struct {
	Revision string   `json:"revision"`
	Policies []Policy `json:"policies"`
}

func Defaults() Config {
	policies := make([]Policy, 0, 2)
	for _, kind := range []string{"connection", "quality"} {
		policies = append(policies, Policy{Kind: kind, AccountScope: "all", AccountIDs: []int64{}, Platforms: []string{}, Statuses: []string{"error"}, IntervalMinutes: 5, FailureThreshold: 2})
	}
	return Config{Policies: policies}
}
func Validate(c Config) error {
	if len(c.Policies) != 2 {
		return fmt.Errorf("exactly two policies (connection and quality) are required")
	}
	seen := map[string]bool{}
	for _, p := range c.Policies {
		if (p.Kind != "connection" && p.Kind != "quality") || seen[p.Kind] {
			return fmt.Errorf("invalid or duplicate policy kind")
		}
		seen[p.Kind] = true
		if p.AccountScope != "all" && p.AccountScope != "selected" {
			return fmt.Errorf("account_scope must be all or selected")
		}
		if p.Enabled && p.AccountScope == "selected" && len(p.AccountIDs) == 0 {
			return fmt.Errorf("select at least one account")
		}
		for _, id := range p.AccountIDs {
			if id <= 0 {
				return fmt.Errorf("account IDs must be positive")
			}
		}
		if len(p.Statuses) == 0 {
			return fmt.Errorf("select at least one target state")
		}
		for _, s := range p.Statuses {
			if !slices.Contains([]string{"active", "inactive", "error", "paused", "rate_limited", "downgraded"}, s) {
				return fmt.Errorf("invalid target state %q", s)
			}
		}
		for _, s := range p.Platforms {
			if !slices.Contains([]string{"openai", "anthropic", "gemini", "antigravity", "grok"}, s) {
				return fmt.Errorf("invalid platform %q", s)
			}
		}
		if p.IntervalMinutes < 1 || p.IntervalMinutes > 1440 {
			return fmt.Errorf("interval_minutes must be 1..1440")
		}
		if p.FailureThreshold < 1 || p.FailureThreshold > 10 {
			return fmt.Errorf("failure_threshold must be 1..10")
		}
		if p.Kind != "quality" && p.PauseOnDowngrade {
			return fmt.Errorf("connection tests cannot pause for downgrade")
		}
	}
	return nil
}

type Account struct {
	ID                                                               int64
	Name, Platform, Status                                           string
	Schedulable, Archived, Deleted, Expired, RateLimited, Downgraded bool
}

func Eligible(p Policy, a Account) bool {
	if a.Deleted || a.Archived || a.Expired || (!a.Schedulable && !p.IncludePaused) {
		return false
	}
	if p.Kind == "quality" && a.Platform != "openai" {
		return false
	}
	if p.AccountScope == "selected" && !slices.Contains(p.AccountIDs, a.ID) {
		return false
	}
	if len(p.Platforms) > 0 && !slices.Contains(p.Platforms, a.Platform) {
		return false
	}
	return slices.Contains(p.Statuses, a.Status) || (!a.Schedulable && slices.Contains(p.Statuses, "paused")) || (a.RateLimited && slices.Contains(p.Statuses, "rate_limited")) || (a.Downgraded && slices.Contains(p.Statuses, "downgraded"))
}

type Store interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string) error
	// Lease must be database-scoped so main and free remain independent.
	Lease(context.Context) (func(), bool, error)
}
type Outcome struct {
	StandardKey string `json:"standard_key,omitempty"`
	Healthy     bool   `json:"healthy"`
	Downgraded  bool   `json:"downgraded"`
	Standard    bool   `json:"standard"`
	Error       string `json:"error,omitempty"`
}
type Actions struct{ Recover, Activate, Resume, Pause bool }
type Executor interface {
	Validate(context.Context, Policy) error
	List(context.Context) ([]Account, error)
	Get(context.Context, int64) (Account, error)
	Probe(context.Context, Policy, Account) (Outcome, error)
	Apply(context.Context, Account, Actions) error
}
type Result struct {
	AccountID   int64     `json:"account_id"`
	AccountName string    `json:"account_name"`
	Kind        string    `json:"kind"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	FinishedAt  time.Time `json:"finished_at"`
}
type Run struct {
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
	Checked    int       `json:"checked"`
	Succeeded  int       `json:"succeeded"`
	Failed     int       `json:"failed"`
	Skipped    int       `json:"skipped"`
	Recovered  int       `json:"recovered"`
	Error      string    `json:"error,omitempty"`
}
type Runtime struct {
	Standards map[string]string    `json:"standards,omitempty"`
	Running   bool                 `json:"running"`
	NextRunAt *time.Time           `json:"next_run_at"`
	LastRun   *Run                 `json:"last_run"`
	History   []Run                `json:"history"`
	Results   []Result             `json:"results"`
	Attempts  map[string]time.Time `json:"attempts"`
	Failures  map[string]int       `json:"failures"`
}

func read[T any](ctx context.Context, s Store, key string, v *T) error {
	raw, err := s.Get(ctx, key)
	if err != nil {
		return err
	}
	if raw == "" {
		return nil
	}
	return json.Unmarshal([]byte(raw), v)
}
func write(ctx context.Context, s Store, key string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return s.Set(ctx, key, string(raw))
}
