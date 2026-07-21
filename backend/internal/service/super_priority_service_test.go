package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// superPriorityFakeRepo 是仅覆盖 SetAccountFlag 所需方法的最小假账号仓储。
// 嵌入 AccountRepository 接口以补齐其余方法；若模式切换错误地改写账号状态，
// SetSchedulable 会留下可断言的调用记录。
type superPriorityFakeRepo struct {
	AccountRepository
	accounts    []Account
	schedulable map[int64]bool // 记录每次 SetSchedulable 后的状态
	extraWrites map[int64]map[string]any
}

func newSuperPriorityFakeRepo() *superPriorityFakeRepo {
	return &superPriorityFakeRepo{
		schedulable: make(map[int64]bool),
		extraWrites: make(map[int64]map[string]any),
	}
}

func (f *superPriorityFakeRepo) ListAllWithFilters(_ context.Context, _, _, _, _ string, _ int64, _ string, _ bool) ([]Account, error) {
	return append([]Account(nil), f.accounts...), nil
}

func (f *superPriorityFakeRepo) FindByExtraField(_ context.Context, key string, value any) ([]Account, error) {
	var out []Account
	want, _ := value.(bool)
	for _, a := range f.accounts {
		if getExtraBool(a.Extra, key) == want {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *superPriorityFakeRepo) SetSchedulable(_ context.Context, id int64, schedulable bool) error {
	f.schedulable[id] = schedulable
	return nil
}

func (f *superPriorityFakeRepo) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	existing := f.extraWrites[id]
	if existing == nil {
		existing = map[string]any{}
	}
	for k, v := range updates {
		existing[k] = v
	}
	f.extraWrites[id] = existing
	return nil
}

func makeSuperPriorityTestAccount(id int64, superPriority, schedulable bool) Account {
	a := Account{ID: id, Status: "active", Schedulable: schedulable, Extra: map[string]any{}}
	if superPriority {
		a.Extra[SuperPriorityExtraKey] = true
	}
	return a
}

func newSuperPriorityTestService(repo *superPriorityFakeRepo) *SuperPriorityService {
	return &SuperPriorityService{
		accountRepo:    repo,
		cfg:            &config.Config{SuperPriority: config.SuperPriorityConfig{Mode: "normal", FailureThreshold: 2}},
		failureWindows: make(map[int64][]time.Time),
	}
}

func TestSuperPriority_Activate_DoesNotMutateSchedulable(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{
		makeSuperPriorityTestAccount(1, true, true),
		makeSuperPriorityTestAccount(2, true, true),
		makeSuperPriorityTestAccount(3, false, true),
		makeSuperPriorityTestAccount(4, false, true),
	}
	svc := newSuperPriorityTestService(repo)
	svc.persistFunc = func() error { return nil }

	if err := svc.Activate(context.Background()); err != nil {
		t.Fatalf("Activate: %v", err)
	}

	if !svc.IsActive() {
		t.Fatalf("expected mode=super_priority, got %q", svc.Mode())
	}
	if len(repo.schedulable) != 0 {
		t.Fatalf("activating the overlay must not mutate schedulable, got %+v", repo.schedulable)
	}
}

func TestSuperPriority_Deactivate_DoesNotMutateSchedulable(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	repo.accounts = []Account{
		makeSuperPriorityTestAccount(1, true, true),
		makeSuperPriorityTestAccount(2, false, true),
	}
	svc := newSuperPriorityTestService(repo)
	svc.persistFunc = func() error { return nil }

	if err := svc.Activate(context.Background()); err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if err := svc.Deactivate(context.Background(), "test"); err != nil {
		t.Fatalf("Deactivate: %v", err)
	}

	if svc.IsActive() {
		t.Fatalf("expected mode=normal after demote, got %q", svc.Mode())
	}
	if len(repo.schedulable) != 0 {
		t.Fatalf("deactivating the overlay must not mutate schedulable, got %+v", repo.schedulable)
	}
}

func TestSuperPriority_UpdateRuntimeParams_NormalizesBaseStrategy(t *testing.T) {
	svc := newSuperPriorityTestService(newSuperPriorityFakeRepo())

	svc.UpdateRuntimeParams(3, "@every 2m", "model", "prompt", AccountSchedulingStrategyLowestCost)
	if got := svc.BaseStrategy(); got != AccountSchedulingStrategyLowestCost {
		t.Fatalf("expected lowest_cost, got %q", got)
	}

	svc.UpdateRuntimeParams(3, "@every 2m", "model", "prompt", "unknown")
	if got := svc.BaseStrategy(); got != AccountSchedulingStrategyDefault {
		t.Fatalf("unknown strategy should normalize to default, got %q", got)
	}
}

// 滚动 1 分钟窗口内失败次数达到阈值应触发降级信号。
func TestSuperPriority_RecordFailure_TriggersDemoteAtThreshold(t *testing.T) {
	svc := newSuperPriorityTestService(newSuperPriorityFakeRepo())

	if svc.RecordFailure(1) {
		t.Fatalf("first failure should not yet reach threshold=2")
	}
	if !svc.RecordFailure(1) {
		t.Fatalf("second failure within 1 minute should reach threshold")
	}
}

// 记录失败应隔离到每个账号的独立窗口。
func TestSuperPriority_RecordFailure_PerAccountIsolated(t *testing.T) {
	svc := newSuperPriorityTestService(newSuperPriorityFakeRepo())

	svc.RecordFailure(1)
	if svc.RecordFailure(2) {
		t.Fatalf("failure on account 2 should not count toward account 1 threshold")
	}
}

// SetAccountFlag 应写入 extra JSONB。
func TestSuperPriority_SetAccountFlag(t *testing.T) {
	repo := newSuperPriorityFakeRepo()
	svc := newSuperPriorityTestService(repo)

	if err := svc.SetAccountFlag(context.Background(), 7, true); err != nil {
		t.Fatalf("SetAccountFlag: %v", err)
	}
	if v, _ := repo.extraWrites[7][SuperPriorityExtraKey].(bool); !v {
		t.Fatalf("expected extra.super_priority=true, got %+v", repo.extraWrites[7])
	}

	if err := svc.SetAccountFlag(context.Background(), 7, false); err != nil {
		t.Fatalf("SetAccountFlag disable: %v", err)
	}
	if v, _ := repo.extraWrites[7][SuperPriorityExtraKey].(bool); v {
		t.Fatalf("expected extra.super_priority=false after disable")
	}
}
