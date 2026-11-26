package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/autodetection"
)

// All policy decisions live in autodetection. This adapter only translates the
// existing repositories, test executor and recovery operations into module ports.
type automaticDetectionAdapter struct {
	repo     SettingRepository
	accounts AccountRepository
	tests    *AccountTestService
	recovery *RateLimitService
	db       *sql.DB
}

func (a *automaticDetectionAdapter) Get(ctx context.Context, key string) (string, error) {
	v, err := a.repo.GetValue(ctx, key)
	if errors.Is(err, ErrSettingNotFound) {
		return "", nil
	}
	return v, err
}
func (a *automaticDetectionAdapter) Set(ctx context.Context, key, value string) error {
	return a.repo.Set(ctx, key, value)
}
func (a *automaticDetectionAdapter) Lease(ctx context.Context) (func(), bool, error) {
	conn, err := a.db.Conn(ctx)
	if err != nil {
		return nil, false, err
	}
	var ok bool
	if err = conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock(18381, 20261018)").Scan(&ok); err != nil || !ok {
		conn.Close()
		return nil, false, err
	}
	return func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := conn.ExecContext(releaseCtx, "SELECT pg_advisory_unlock(18381, 20261018)"); err != nil {
			_ = conn.Raw(func(any) error { return driver.ErrBadConn })
		}
		_ = conn.Close()
	}, true, nil
}

type automaticDetectionExecutor struct{ adapter *automaticDetectionAdapter }

func automaticDetectionAccount(a *Account) autodetection.Account {
	if a == nil {
		return autodetection.Account{Deleted: true}
	}
	now := time.Now()
	archived, _ := a.Extra["recycled"].(bool)
	downgraded, _ := a.Extra["pelican_downgraded"].(bool)
	return autodetection.Account{ID: a.ID, Name: a.Name, Platform: a.Platform, Status: a.Status, Schedulable: a.Schedulable, Archived: archived, Deleted: a.DeletedAt != nil || a.IsDeletedStaging(), Expired: a.ExpiresAt != nil && !a.ExpiresAt.After(now), RateLimited: (a.RateLimitResetAt != nil && a.RateLimitResetAt.After(now)) || (a.TempUnschedulableUntil != nil && a.TempUnschedulableUntil.After(now)) || (a.OverloadUntil != nil && a.OverloadUntil.After(now)), Downgraded: downgraded}
}
func (x automaticDetectionExecutor) List(ctx context.Context) ([]autodetection.Account, error) {
	accounts, err := x.adapter.accounts.ListAllWithFilters(ctx, "", "", "", "", 0, "", false)
	if err != nil {
		return nil, err
	}
	out := make([]autodetection.Account, 0, len(accounts))
	for i := range accounts {
		out = append(out, automaticDetectionAccount(&accounts[i]))
	}
	return out, nil
}
func (x automaticDetectionExecutor) Get(ctx context.Context, id int64) (autodetection.Account, error) {
	a, err := x.adapter.accounts.GetByID(ctx, id)
	return automaticDetectionAccount(a), err
}
func (x automaticDetectionExecutor) Probe(ctx context.Context, p autodetection.Policy, snapshot autodetection.Account) (autodetection.Outcome, error) {
	a := x.adapter
	account, err := a.accounts.GetByID(ctx, snapshot.ID)
	if err != nil {
		return autodetection.Outcome{}, err
	}
	model := strings.TrimSpace(p.ModelID)
	if model == "" {
		models := accountTestModelCandidates(account)
		if len(models) > 0 {
			model = models[0]
		}
	}
	mode := AccountTestModeDefault
	if p.Kind == "quality" {
		mode = AccountTestModePelican
	}
	ctx = withDetectionOrigin(ctx, "scheduled", mode, snapshot.ID, 0, strings.TrimSpace(p.ModelID) == "")
	var result *ScheduledTestResult
	if p.Kind == "quality" {
		settings, loadErr := a.tests.SettingService().GetRequestQualityProbeSettings(ctx)
		if loadErr != nil {
			return autodetection.Outcome{}, loadErr
		}
		if IsQualityProbeExcluded(settings, model) {
			return autodetection.Outcome{Healthy: true, Standard: false}, nil
		}
		if len(settings.RuleIDs) != 1 {
			return autodetection.Outcome{}, fmt.Errorf("请先在试题库指定自动测智标准试题")
		}
		rule, loadErr := a.tests.qualityRuleRepo.GetByID(ctx, settings.RuleIDs[0])
		if loadErr != nil {
			return autodetection.Outcome{}, loadErr
		}
		if rule == nil || !rule.Enabled {
			return autodetection.Outcome{}, fmt.Errorf("自动测智标准试题不存在或已停用")
		}
		result, err = a.tests.RunDegradationProbeRules(ctx, snapshot.ID, model, []*DegradationRule{rule})
	} else {
		result, err = a.tests.RunTestBackground(ctx, snapshot.ID, model)
	}
	if err != nil {
		return autodetection.Outcome{}, err
	}
	if result == nil {
		return autodetection.Outcome{}, fmt.Errorf("test returned no result")
	}
	standard := a.tests.IsAutomaticQualityResult(ctx, result)
	standardKey := ""
	if standard {
		standardKey = fmt.Sprintf("%+v", result.QualityQuestion)
	}
	return autodetection.Outcome{Healthy: result.Status == "success" && !result.Downgraded, Downgraded: result.Status == "success" && result.Downgraded, Standard: standard, StandardKey: standardKey, Error: result.ErrorMessage}, nil
}
func (x automaticDetectionExecutor) Apply(ctx context.Context, before autodetection.Account, actions autodetection.Actions) error {
	a := x.adapter
	current, err := a.accounts.GetByID(ctx, before.ID)
	if err != nil {
		return err
	}
	snapshot := automaticDetectionAccount(current)
	if snapshot.Deleted || snapshot.Archived || snapshot.Expired || snapshot.Status != before.Status || snapshot.Schedulable != before.Schedulable {
		return fmt.Errorf("account changed; automatic action cancelled")
	}
	if actions.Recover {
		if _, err = a.recovery.RecoverAccountState(ctx, before.ID, AccountRecoveryOptions{InvalidateToken: true}); err != nil {
			return err
		}
	}
	// Re-read after recovery because it can change error to active and updated_at.
	current, err = a.accounts.GetByID(ctx, before.ID)
	if err != nil {
		return err
	}
	if current.Schedulable != before.Schedulable {
		return fmt.Errorf("scheduling changed; automatic action cancelled")
	}
	status := current.Status
	if actions.Activate && status == "inactive" {
		status = StatusActive
	}
	schedulable := current.Schedulable
	if actions.Resume {
		schedulable = true
	}
	if actions.Pause {
		schedulable = false
	}
	writer, ok := a.accounts.(interface {
		CompareAndSetAutomaticDetectionState(context.Context, *Account, string, bool) error
	})
	if !ok {
		return fmt.Errorf("guarded automatic recovery writer unavailable")
	}
	if err := writer.CompareAndSetAutomaticDetectionState(ctx, current, status, schedulable); err != nil {
		return err
	}
	a.recovery.notifyAccountSchedulingBlockCleared(current.ID)
	return nil
}
func newAutomaticDetection(repo SettingRepository, accounts AccountRepository, tests *AccountTestService, recovery *RateLimitService, db *sql.DB) *autodetection.Engine {
	adapter := &automaticDetectionAdapter{repo: repo, accounts: accounts, tests: tests, recovery: recovery, db: db}
	return autodetection.New(adapter, automaticDetectionExecutor{adapter})
}

func (x automaticDetectionExecutor) Validate(ctx context.Context, p autodetection.Policy) error {
	if p.Kind != "quality" {
		return nil
	}
	tests := x.adapter.tests
	if tests == nil || tests.SettingService() == nil || tests.qualityRuleRepo == nil {
		return fmt.Errorf("quality question service unavailable")
	}
	settings, err := tests.SettingService().GetRequestQualityProbeSettings(ctx)
	if err != nil {
		return err
	}
	if len(settings.RuleIDs) != 1 {
		return fmt.Errorf("请先指定自动测智标准试题")
	}
	rule, err := tests.qualityRuleRepo.GetByID(ctx, settings.RuleIDs[0])
	if err != nil {
		return err
	}
	if rule == nil || !rule.Enabled {
		return fmt.Errorf("自动测智标准试题不存在或已停用")
	}
	return nil
}
