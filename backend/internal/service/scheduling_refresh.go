package service

// SchedulingRefreshResult summarizes one manually triggered scheduling refresh.
type SchedulingRefreshResult struct {
	Checked   int `json:"checked"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
}
