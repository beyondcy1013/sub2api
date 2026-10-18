package autodetection

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu     sync.Mutex
	data   map[string]string
	leased bool
}

func (s *memoryStore) Get(_ context.Context, k string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data[k], nil
}
func (s *memoryStore) Set(_ context.Context, k, v string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[k] = v
	return nil
}
func (s *memoryStore) Lease(context.Context) (func(), bool, error) {
	if s.leased {
		return func() {}, false, nil
	}
	return func() {}, true, nil
}

type fakeExecutor struct {
	mu         sync.Mutex
	accounts   []Account
	outcome    Outcome
	probed     []int64
	applied    []Actions
	hook       func()
	validation error
	applyErr   error
}

func (x *fakeExecutor) Validate(context.Context, Policy) error { return x.validation }
func (x *fakeExecutor) List(context.Context) ([]Account, error) {
	return append([]Account{}, x.accounts...), nil
}
func (x *fakeExecutor) Get(_ context.Context, id int64) (Account, error) {
	for _, a := range x.accounts {
		if a.ID == id {
			return a, nil
		}
	}
	return Account{}, fmt.Errorf("missing account")
}
func (x *fakeExecutor) Probe(_ context.Context, _ Policy, a Account) (Outcome, error) {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.probed = append(x.probed, a.ID)
	if x.hook != nil {
		x.hook()
	}
	return x.outcome, nil
}
func (x *fakeExecutor) Apply(_ context.Context, _ Account, a Actions) error {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.applied = append(x.applied, a)
	return x.applyErr
}
func fixture(t *testing.T) (*Engine, *memoryStore, *fakeExecutor, Config) {
	t.Helper()
	s := &memoryStore{data: map[string]string{}}
	x := &fakeExecutor{accounts: []Account{{ID: 1, Status: "error", Platform: "openai", Schedulable: true}}, outcome: Outcome{Healthy: true}}
	e := New(s, x)
	c := Defaults()
	c.Policies[0].Enabled = true
	c.Policies[0].Recover = true
	c.Policies[0].Resume = true
	if err := e.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	return e, s, x, c
}
func TestEligible(t *testing.T) {
	p := Defaults().Policies[0]
	base := Account{ID: 1, Status: "error", Platform: "openai", Schedulable: true}
	for _, tc := range []struct {
		name   string
		change func(*Account)
		want   bool
	}{{"error", func(*Account) {}, true}, {"active", func(a *Account) { a.Status = "active" }, false}, {"paused", func(a *Account) { a.Schedulable = false }, false}, {"trash", func(a *Account) { a.Deleted = true }, false}, {"archived", func(a *Account) { a.Archived = true }, false}, {"expired", func(a *Account) { a.Expired = true }, false}} {
		t.Run(tc.name, func(t *testing.T) {
			a := base
			tc.change(&a)
			if Eligible(p, a) != tc.want {
				t.Fatal("unexpected eligibility")
			}
		})
	}
	p.AccountScope = "selected"
	p.AccountIDs = []int64{2}
	if Eligible(p, base) {
		t.Fatal("unselected account matched")
	}
	p.AccountIDs = []int64{1}
	if !Eligible(p, base) {
		t.Fatal("selected account excluded")
	}
	p.Kind = "quality"
	base.Platform = "anthropic"
	if Eligible(p, base) {
		t.Fatal("unsupported quality platform")
	}
}
func TestRunPersistsAndBacksOff(t *testing.T) {
	e, s, x, _ := fixture(t)
	ctx := context.Background()
	if err := e.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(x.probed) != 1 || len(x.applied) != 1 {
		t.Fatal("expected one test and recovery")
	}
	e = New(s, x)
	if err := e.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if len(x.probed) != 1 {
		t.Fatal("restart lost interval state")
	}
	r, _ := e.Runtime(ctx)
	if r.LastRun.Recovered != 1 || r.NextRunAt == nil {
		t.Fatal("runtime missing")
	}
}
func TestAllIncludesFutureAccountsSelectedDoesNot(t *testing.T) {
	for _, scope := range []string{"all", "selected"} {
		t.Run(scope, func(t *testing.T) {
			e, _, x, c := fixture(t)
			c.Policies[0].AccountScope = scope
			c.Policies[0].AccountIDs = []int64{1}
			_ = e.Save(context.Background(), c)
			x.accounts = append(x.accounts, Account{ID: 2, Status: "error", Platform: "openai", Schedulable: true})
			_ = e.Run(context.Background())
			want := 2
			if scope == "selected" {
				want = 1
			}
			if len(x.probed) != want {
				t.Fatal("wrong account scope")
			}
		})
	}
}
func TestManualPauseAndPolicyEditCancelRecovery(t *testing.T) {
	for _, edit := range []string{"pause", "config"} {
		t.Run(edit, func(t *testing.T) {
			e, _, x, c := fixture(t)
			x.hook = func() {
				if edit == "pause" {
					x.accounts[0].Schedulable = false
				} else {
					c.Policies[0].Enabled = false
					_ = e.Save(context.Background(), c)
				}
			}
			_ = e.Run(context.Background())
			if len(x.applied) != 0 {
				t.Fatal("stale recovery applied")
			}
		})
	}
}
func TestConnectionCannotClearQualityFailure(t *testing.T) {
	e, _, x, _ := fixture(t)
	x.accounts[0].Downgraded = true
	_ = e.Run(context.Background())
	if len(x.applied) != 0 {
		t.Fatal("connection cleared quality failure")
	}
}
func TestQualityRequiresStandardProof(t *testing.T) {
	for _, standard := range []bool{false, true} {
		t.Run(fmt.Sprint(standard), func(t *testing.T) {
			e, _, x, c := fixture(t)
			c.Policies[0].Enabled = false
			c.Policies[1].Enabled = true
			c.Policies[1].Recover = true
			c.Policies[1].IncludePaused = true
			x.accounts[0].Downgraded = true
			x.outcome.Standard = standard
			_ = e.Save(context.Background(), c)
			_ = e.Run(context.Background())
			if (len(x.applied) > 0) != standard {
				t.Fatal("invalid standard recovery")
			}
		})
	}
}
func TestPauseThresholdAndFailureRecoveryError(t *testing.T) {
	e, s, x, c := fixture(t)
	c.Policies[0].Enabled = false
	p := &c.Policies[1]
	p.Enabled = true
	p.IncludePaused = true
	p.PauseOnDowngrade = true
	p.IntervalMinutes = 1
	x.outcome = Outcome{Standard: true, Downgraded: true}
	_ = e.Save(context.Background(), c)
	_ = e.Run(context.Background())
	if len(x.applied) > 0 {
		t.Fatal("paused below threshold")
	}
	r, _ := e.Runtime(context.Background())
	for k := range r.Attempts {
		r.Attempts[k] = time.Now().Add(-2 * time.Minute)
	}
	_ = write(context.Background(), s, RuntimeKey, r)
	_ = e.Run(context.Background())
	if len(x.applied) != 1 || !x.applied[0].Pause {
		t.Fatal("threshold did not pause")
	}
}
func TestBatchIsBoundedAndFair(t *testing.T) {
	e, s, x, _ := fixture(t)
	x.accounts = nil
	for i := int64(1); i <= 25; i++ {
		x.accounts = append(x.accounts, Account{ID: i, Status: "error", Platform: "openai", Schedulable: true})
	}
	_ = e.Run(context.Background())
	if len(x.probed) != 20 {
		t.Fatal("batch not bounded")
	}
	e = New(s, x)
	_ = e.Run(context.Background())
	if len(x.probed) != 25 {
		t.Fatal("remaining accounts starved")
	}
}
func TestSaveRejectsInvalidAndUnavailableStandard(t *testing.T) {
	e, _, x, c := fixture(t)
	c.Policies[0].AccountScope = "selected"
	if e.Save(context.Background(), c) == nil {
		t.Fatal("accepted empty selection")
	}
	c = Defaults()
	c.Policies[1].Enabled = true
	x.validation = fmt.Errorf("no standard question")
	if e.Save(context.Background(), c) == nil {
		t.Fatal("accepted unavailable standard")
	}
}
func TestLeaseAndDisabledDoNotProbe(t *testing.T) {
	e, s, x, c := fixture(t)
	s.leased = true
	_ = e.Run(context.Background())
	if len(x.probed) > 0 {
		t.Fatal("lease ignored")
	}
	s.leased = false
	c.Policies[0].Enabled = false
	_ = e.Save(context.Background(), c)
	_ = e.Run(context.Background())
	if len(x.probed) > 0 {
		t.Fatal("disabled policy ran")
	}
	raw, _ := s.Get(context.Background(), ConfigKey)
	var saved Config
	if json.Unmarshal([]byte(raw), &saved) != nil || saved.Revision == "" {
		t.Fatal("config not persisted")
	}
}

func TestFailedProbeAndFailedActionNeverCountAsRecovery(t *testing.T) {
	for _, actionError := range []bool{false, true} {
		e, _, x, _ := fixture(t)
		if actionError {
			x.applyErr = fmt.Errorf("concurrent edit")
		} else {
			x.outcome = Outcome{Healthy: false, Error: "upstream unavailable"}
		}
		if err := e.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		r, _ := e.Runtime(context.Background())
		if r.LastRun.Recovered != 0 || r.LastRun.Failed != 1 || r.LastRun.Succeeded != 0 {
			t.Fatal("failed operation counted as recovery")
		}
		if !actionError && len(x.applied) != 0 {
			t.Fatal("failed probe applied recovery")
		}
	}
}
func TestQualityQuestionChangeResetsFailureThreshold(t *testing.T) {
	e, s, x, c := fixture(t)
	c.Policies[0].Enabled = false
	c.Policies[1].Enabled = true
	c.Policies[1].PauseOnDowngrade = true
	if err := e.Save(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	x.outcome = Outcome{Standard: true, StandardKey: "old question", Downgraded: true}
	_ = e.Run(context.Background())
	r, _ := e.Runtime(context.Background())
	for k := range r.Attempts {
		r.Attempts[k] = time.Now().Add(-time.Hour)
	}
	_ = write(context.Background(), s, RuntimeKey, r)
	x.outcome.StandardKey = "edited question"
	_ = e.Run(context.Background())
	if len(x.applied) != 0 {
		t.Fatal("old question failures triggered pause for edited question")
	}
}
