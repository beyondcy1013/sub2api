package service

import (
	"context"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const scheduledAccountActionPollInterval = 15 * time.Second

type ScheduledAccountActionRunner struct {
	service   *ScheduledAccountActionService
	startOnce sync.Once
	stopOnce  sync.Once
	stop      chan struct{}
	done      chan struct{}
}

func NewScheduledAccountActionRunner(service *ScheduledAccountActionService) *ScheduledAccountActionRunner {
	return &ScheduledAccountActionRunner{
		service: service,
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (r *ScheduledAccountActionRunner) Start() {
	if r == nil || r.service == nil {
		return
	}
	r.startOnce.Do(func() {
		go r.run()
		logger.LegacyPrintf("service.scheduled_account_action_runner", "[ScheduledAccountActionRunner] started (poll=%s)", scheduledAccountActionPollInterval)
	})
}

func (r *ScheduledAccountActionRunner) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stop)
		select {
		case <-r.done:
		case <-time.After(3 * time.Second):
			logger.LegacyPrintf("service.scheduled_account_action_runner", "[ScheduledAccountActionRunner] stop timed out")
		}
	})
}

func (r *ScheduledAccountActionRunner) run() {
	defer close(r.done)
	r.runOnce()
	ticker := time.NewTicker(scheduledAccountActionPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.runOnce()
		case <-r.stop:
			return
		}
	}
}

func (r *ScheduledAccountActionRunner) runOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	processed, err := r.service.ProcessDue(ctx, 20)
	if err != nil {
		logger.LegacyPrintf("service.scheduled_account_action_runner", "[ScheduledAccountActionRunner] processed=%d error=%v", processed, err)
	}
}
