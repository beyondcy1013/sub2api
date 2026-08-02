package service

import "time"

// SchedulingRefreshResult summarizes one scheduling probe batch.
type SchedulingRefreshResult struct {
	Checked   int `json:"checked"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
}

// SchedulingLivenessRunStatus describes the latest real liveness batch. Empty
// background scans do not replace it, so operators keep seeing the last useful
// result while the runner waits for another account to become due.
type SchedulingLivenessRunStatus struct {
	Trigger    string                  `json:"trigger"`
	StartedAt  time.Time               `json:"started_at"`
	FinishedAt time.Time               `json:"finished_at"`
	Result     SchedulingRefreshResult `json:"result"`
	Error      string                  `json:"error,omitempty"`
}

// SchedulingLivenessRuntimeStatus is the server-side source of truth for the
// scheduling-rules dialog countdown and latest-result summary.
type SchedulingLivenessRuntimeStatus struct {
	Enabled   bool                         `json:"enabled"`
	Running   bool                         `json:"running"`
	NextRunAt *time.Time                   `json:"next_run_at,omitempty"`
	LastRun   *SchedulingLivenessRunStatus `json:"last_run,omitempty"`
}
