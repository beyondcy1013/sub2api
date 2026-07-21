package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/robfig/cron/v3"
)

// superPriorityCronParser 支持 @every 描述符与传统 5 字段 cron 表达式。
var superPriorityCronParser = cron.NewParser(
	cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

// SuperPriorityRunner 定时探测超级优先账号可用性。
//
// 每个周期：枚举 super_priority=true 的账号，复用 AccountTestService.RunTestBackground
// 进行真实对话测试；任一账号失败则调用 SuperPriorityService.RecordFailure，当滚动窗口内
// 失败次数达到阈值时自动 Deactivate。成功不重置窗口（窗口本身按时间自然过期）。
type SuperPriorityRunner struct {
	state           *SuperPriorityService
	accountTestSvc  *AccountTestService
	accountRepo     AccountRepository

	cron      *cron.Cron
	startOnce sync.Once
	stopOnce  sync.Once
}

// NewSuperPriorityRunner 创建探测运行器。
func NewSuperPriorityRunner(state *SuperPriorityService, accountTestSvc *AccountTestService, accountRepo AccountRepository) *SuperPriorityRunner {
	return &SuperPriorityRunner{
		state:          state,
		accountTestSvc: accountTestSvc,
		accountRepo:    accountRepo,
	}
}

// Start 启动定时探测（仅在配置的表达式上生效；模式非激活时 runOnce 直接返回）。
func (r *SuperPriorityRunner) Start() {
	if r == nil || r.state == nil || r.state.cfg == nil {
		return
	}
	r.startOnce.Do(func() {
		expr := r.state.cfg.SuperPriority.CheckInterval
		if expr == "" {
			expr = "@every 1m"
		}
		loc := time.Local
		if r.state.cfg != nil {
			if parsed, err := time.LoadLocation(r.state.cfg.Timezone); err == nil && parsed != nil {
				loc = parsed
			}
		}

		c := cron.New(cron.WithParser(superPriorityCronParser), cron.WithLocation(loc))
		if _, err := c.AddFunc(expr, func() { r.runOnce() }); err != nil {
			logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] not started (invalid schedule %q): %v", expr, err)
			return
		}
		r.cron = c
		r.cron.Start()
		logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] started (schedule=%s)", expr)
	})
}

// Stop 优雅关闭。
func (r *SuperPriorityRunner) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		if r.cron != nil {
			ctx := r.cron.Stop()
			select {
			case <-ctx.Done():
			case <-time.After(3 * time.Second):
				logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] cron stop timed out")
			}
		}
	})
}

// RunOnce 暴露单次探测，便于测试与手动触发。
func (r *SuperPriorityRunner) RunOnce(ctx context.Context) {
	r.runOnceOnce(ctx)
}

func (r *SuperPriorityRunner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	r.runOnceOnce(ctx)
}

func (r *SuperPriorityRunner) runOnceOnce(ctx context.Context) {
	if r == nil || r.state == nil || !r.state.IsActive() {
		return
	}

	// 枚举被标记为超级优先的账号。
	accounts, err := r.accountRepo.FindByExtraField(ctx, SuperPriorityExtraKey, true)
	if err != nil {
		logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] FindByExtraField error: %v", err)
		return
	}
	if len(accounts) == 0 {
		return
	}

	modelID := ""
	if r.state.cfg != nil {
		modelID = r.state.cfg.SuperPriority.TestModelID
	}

	for _, acc := range accounts {
		// 仅探测 active 账号；inactive/error 账号视为不可用，计入失败窗口。
		if acc.Status != "active" {
			if r.state.RecordFailure(acc.ID) {
				if derr := r.state.Deactivate(ctx, "super_priority account not active"); derr != nil {
					logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] auto-demote failed: %v", derr)
				}
				return
			}
			continue
		}

		result, testErr := r.accountTestSvc.RunTestBackground(ctx, acc.ID, modelID)
		failed := testErr != nil || result == nil || result.Status != "success"
		if !failed {
			continue
		}

		errMsg := ""
		if testErr != nil {
			errMsg = testErr.Error()
		} else if result != nil {
			errMsg = result.ErrorMessage
		}
		logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] probe account=%d failed: %s", acc.ID, errMsg)

		if r.state.RecordFailure(acc.ID) {
			if derr := r.state.Deactivate(ctx, "probe failure threshold reached"); derr != nil {
				logger.LegacyPrintf("service.super_priority_runner", "[SuperPriorityRunner] auto-demote failed: %v", derr)
			}
			return
		}
	}
}
