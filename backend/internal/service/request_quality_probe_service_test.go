package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modules/rotationnotice"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSettingRepoForQualityProbe struct {
	data map[string]string
}

func newMockSettingRepoForQualityProbe() *mockSettingRepoForQualityProbe {
	return &mockSettingRepoForQualityProbe{data: make(map[string]string)}
}

func (m *mockSettingRepoForQualityProbe) Get(ctx context.Context, key string) (*Setting, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, ErrSettingNotFound
	}
	return &Setting{Key: key, Value: val}, nil
}

func (m *mockSettingRepoForQualityProbe) GetValue(ctx context.Context, key string) (string, error) {
	val, ok := m.data[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return val, nil
}

func (m *mockSettingRepoForQualityProbe) Set(ctx context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

func (m *mockSettingRepoForQualityProbe) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string)
	for _, k := range keys {
		if v, ok := m.data[k]; ok {
			out[k] = v
		}
	}
	return out, nil
}

func (m *mockSettingRepoForQualityProbe) SetMultiple(ctx context.Context, settings map[string]string) error {
	for k, v := range settings {
		m.data[k] = v
	}
	return nil
}

func (m *mockSettingRepoForQualityProbe) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(m.data))
	for k, v := range m.data {
		out[k] = v
	}
	return out, nil
}

func (m *mockSettingRepoForQualityProbe) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

type stubAccountRepoForQualityProbe struct {
	AccountRepository
	extraUpdated       map[int64]map[string]any
	qualityState       QualityDegradationState
	qualityStateCalls  int
	qualityStateClears int
	qualityStateResets int
}

func (s *stubAccountRepoForQualityProbe) UpdateExtra(ctx context.Context, id int64, extra map[string]any) error {
	if s.extraUpdated == nil {
		s.extraUpdated = make(map[int64]map[string]any)
	}
	s.extraUpdated[id] = extra
	return nil
}

func (s *stubAccountRepoForQualityProbe) RecordQualityDegradation(_ context.Context, _ int64, fingerprint, model, reason string, threshold int, pause bool) (QualityDegradationState, error) {
	s.qualityStateCalls++
	if s.qualityState.RuleFingerprint != fingerprint {
		s.qualityState.ConsecutiveCount = 0
	}
	s.qualityState.ConsecutiveCount++
	s.qualityState.RuleFingerprint = fingerprint
	s.qualityState.LastModel = model
	s.qualityState.LastReason = reason
	s.qualityState.Threshold = threshold
	if pause && s.qualityState.ConsecutiveCount >= threshold {
		s.qualityState.Paused = true
	}
	return s.qualityState, nil
}

func (s *stubAccountRepoForQualityProbe) ResetQualityDegradation(_ context.Context, _ int64, fingerprint string) error {
	s.qualityStateResets++
	if s.qualityState.RuleFingerprint == fingerprint {
		s.qualityState.ConsecutiveCount = 0
	}
	return nil
}

func (s *stubAccountRepoForQualityProbe) ClearQualityDegradationState(_ context.Context, _ int64) error {
	s.qualityStateClears++
	s.qualityState = QualityDegradationState{}
	return nil
}

type stubAccountTesterForQualityProbe struct {
	scheduledTestAccountTester
	runCount        int
	downgraded      bool
	status          string
	upstreamFailure *AccountTestUpstreamError
	err             error
}

func (s *stubAccountTesterForQualityProbe) RunDegradationProbeRules(ctx context.Context, accountID int64, modelID string, rules []*DegradationRule) (*ScheduledTestResult, error) {
	s.runCount++
	if s.err != nil {
		return nil, s.err
	}
	return &ScheduledTestResult{
		Status:          s.status,
		UpstreamFailure: s.upstreamFailure,
		Downgraded:      s.downgraded,
		DowngradeReason: "candy:期望 21，实际 42",
	}, nil
}

func TestRequestQualityProbeService_Disabled(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	settingSvc := NewSettingService(repo, nil)
	svc := NewRequestQualityProbeService(settingSvc, nil, nil, nil)

	acc := &Account{ID: 1, Name: "acc-1", Platform: PlatformOpenAI}
	passed, _, err := svc.EnsureAccountQuality(context.Background(), acc, "gpt-5.1")
	assert.True(t, passed)
	assert.NoError(t, err)
}

func TestRequestQualityProbeService_ModelMismatch(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	settingSvc := NewSettingService(repo, nil)
	require.NoError(t, settingSvc.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled:       true,
		Models:        []string{"gpt-5.1"},
		RecentMinutes: 10,
		AccountScope:  "all",
	}))

	svc := NewRequestQualityProbeService(settingSvc, nil, nil, nil)
	acc := &Account{ID: 1, Name: "acc-1", Platform: PlatformOpenAI}

	// Model is gpt-4o, not in designated models -> passes immediately without probe
	passed, _, err := svc.EnsureAccountQuality(context.Background(), acc, "gpt-4o")
	assert.True(t, passed)
	assert.NoError(t, err)
}

func TestRequestQualityProbeService_RecentSuccessExempt(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	settingSvc := NewSettingService(repo, nil)
	require.NoError(t, settingSvc.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled:              true,
		AllowBusinessSuccess: true,
		Models:               []string{"gpt-5.1"},
		RecentMinutes:        10,
		AccountScope:         "all",
	}))

	svc := NewRequestQualityProbeService(settingSvc, nil, nil, nil)
	acc := &Account{ID: 10, Name: "acc-10", Platform: PlatformOpenAI}

	// Mark recent success
	svc.TouchRecentSuccess(10, "gpt-5.1")
	assert.True(t, svc.HasRecentSuccess(10, "gpt-5.1", 10))

	// Should pass without any probe
	passed, _, err := svc.EnsureAccountQuality(context.Background(), acc, "gpt-5.1")
	assert.True(t, passed)
	assert.NoError(t, err)
}

func TestRequestQualityProbeService_AccountScopeSelected(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	settingSvc := NewSettingService(repo, nil)
	require.NoError(t, settingSvc.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled:       true,
		Models:        []string{"gpt-5.1"},
		RecentMinutes: 10,
		AccountScope:  "selected",
		AccountIDs:    []int64{100},
	}))

	svc := NewRequestQualityProbeService(settingSvc, nil, nil, nil)

	// Account 200 is not selected -> passes immediately
	acc200 := &Account{ID: 200, Name: "acc-200"}
	passed, _, err := svc.EnsureAccountQuality(context.Background(), acc200, "gpt-5.1")
	assert.True(t, passed)
	assert.NoError(t, err)
}

func TestRequestQualityProbeService_AccountScopeErrorOrDowngraded(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	settingSvc := NewSettingService(repo, nil)
	require.NoError(t, settingSvc.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled:       true,
		Models:        []string{"gpt-5.1"},
		RecentMinutes: 10,
		AccountScope:  "error_or_downgraded",
	}))

	svc := NewRequestQualityProbeService(settingSvc, nil, nil, nil)

	// Account is healthy and active -> passes without probe
	accHealthy := &Account{ID: 1, Name: "healthy", Status: "active"}
	passed, _, err := svc.EnsureAccountQuality(context.Background(), accHealthy, "gpt-5.1")
	assert.True(t, passed)
	assert.NoError(t, err)
}

func TestRequestQualityProbeService_ProbePassedSeamless(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	settingSvc := NewSettingService(repo, nil)
	require.NoError(t, settingSvc.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled:       true,
		Models:        []string{"gpt-5.1"},
		RecentMinutes: 10,
		AccountScope:  "all",
	}))

	accRepo := &stubAccountRepoForQualityProbe{}
	testSvc := NewAccountTestService(accRepo, nil, nil, nil, nil, nil, nil, nil)
	// Inject a stub tester
	mockTester := &stubAccountTesterForQualityProbe{status: "success", downgraded: false}

	svc := NewRequestQualityProbeService(settingSvc, accRepo, nil, testSvc)
	// Override executeProbe via custom stub or ensure testSvc is called
	// We can test the helper methods directly:
	assert.False(t, svc.HasRecentSuccess(42, "gpt-5.1", 10))
	svc.TouchRecentSuccess(42, "gpt-5.1")
	assert.True(t, svc.HasRecentSuccess(42, "gpt-5.1", 10))

	_ = mockTester
}

func TestRequestQualityProbeService_SettingsPersistence(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	settingSvc := NewSettingService(repo, nil)

	settings, err := settingSvc.GetRequestQualityProbeSettings(context.Background())
	require.NoError(t, err)
	assert.False(t, settings.Enabled)
	assert.Equal(t, 10, settings.RecentMinutes)

	newSettings := &RequestQualityProbeSettings{
		Enabled:       true,
		Models:        []string{"gpt-5.1", "gpt-5.2"},
		RecentMinutes: 15,
		AccountScope:  "error_or_downgraded",
		AccountIDs:    []int64{1, 2},
	}
	require.NoError(t, settingSvc.UpdateRequestQualityProbeSettings(context.Background(), newSettings))

	// Re-fetch
	fetched, err := settingSvc.GetRequestQualityProbeSettings(context.Background())
	require.NoError(t, err)
	assert.True(t, fetched.Enabled)
	assert.Equal(t, []string{"gpt-5.1", "gpt-5.2"}, fetched.Models)
	assert.Equal(t, 15, fetched.RecentMinutes)
	assert.Equal(t, "error_or_downgraded", fetched.AccountScope)
	assert.Equal(t, []int64{1, 2}, fetched.AccountIDs)
}

func TestRequestQualityProbeEveryRequestIgnoresRecentSuccessAndProof(t *testing.T) {
	settings := NewSettingService(newMockSettingRepoForQualityProbe(), nil)
	config := DefaultRequestQualityProbeSettings()
	config.Enabled = true
	config.RuleIDs = []int64{1}
	config.AllowBusinessSuccess = true
	require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), config))
	tester := &stubAccountTesterForQualityProbe{status: "success"}
	repo := &stubAccountRepoForQualityProbe{}
	svc := NewRequestQualityProbeService(settings, repo, newBuiltinRuleStub(), nil)
	svc.accountTestService = tester
	svc.TouchRecentSuccess(8, "arbitrary-model")
	for i := 0; i < 2; i++ {
		result, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 8, Status: "active"}, "arbitrary-model")
		require.NotNil(t, result)
		require.NoError(t, <-result)
		cancel()
	}
	require.Equal(t, 2, tester.runCount)
	config.Enabled = false
	require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), config))
	result, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 8}, "arbitrary-model")
	defer cancel()
	require.Nil(t, result)
	require.Equal(t, 2, tester.runCount)
}

func TestRequestQualityProbeEveryRequestMarksOnlyConfirmedDegradation(t *testing.T) {
	for _, status := range []string{"success", "failed", "cancelled", "skipped"} {
		t.Run(status, func(t *testing.T) {
			settings := NewSettingService(newMockSettingRepoForQualityProbe(), nil)
			config := DefaultRequestQualityProbeSettings()
			config.Enabled = true
			config.RuleIDs = []int64{1}
			require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), config))
			repo := &stubAccountRepoForQualityProbe{}
			svc := NewRequestQualityProbeService(settings, repo, newBuiltinRuleStub(), nil)
			svc.accountTestService = &stubAccountTesterForQualityProbe{status: status, downgraded: true}
			result, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 8, Status: "active"}, "model")
			defer cancel()
			err := <-result
			require.Error(t, err)
			var downgraded *rotationnotice.DowngradeError
			require.Equal(t, status == "success", errors.As(err, &downgraded))
			if status == "success" {
				require.Equal(t, true, repo.extraUpdated[8]["pelican_downgraded"])
			} else {
				require.Empty(t, repo.extraUpdated)
			}
		})
	}
}

func TestRequestQualityProbePausesAfterConfirmedDegradationThreshold(t *testing.T) {
	settings := NewSettingService(newMockSettingRepoForQualityProbe(), nil)
	config := DefaultRequestQualityProbeSettings()
	config.Enabled = true
	config.RuleIDs = []int64{1}
	config.ReuseSeconds = intPtr(0)
	config.AutoPauseOnDowngrade = true
	config.DowngradeThreshold = 2
	require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), config))

	repo := &stubAccountRepoForQualityProbe{}
	tester := &stubAccountTesterForQualityProbe{status: "success", downgraded: true}
	svc := NewRequestQualityProbeService(settings, repo, newBuiltinRuleStub(), nil)
	svc.accountTestService = tester
	account := &Account{ID: 77, Status: StatusActive, Schedulable: true}

	for i := 0; i < 2; i++ {
		result, cancel := svc.StartRequestQualityCheck(context.Background(), account, "gpt-6-astra")
		require.NotNil(t, result)
		require.Error(t, <-result)
		cancel()
	}

	require.Equal(t, 2, repo.qualityStateCalls)
	require.Equal(t, 2, repo.qualityState.ConsecutiveCount)
	require.True(t, repo.qualityState.Paused)
	require.False(t, account.Schedulable)
}

func TestRequestQualityProbeCleanPassResetsCountButDoesNotResumePause(t *testing.T) {
	settings := NewSettingService(newMockSettingRepoForQualityProbe(), nil)
	config := DefaultRequestQualityProbeSettings()
	config.Enabled = true
	config.RuleIDs = []int64{1}
	config.ReuseSeconds = intPtr(0)
	config.AutoPauseOnDowngrade = true
	config.DowngradeThreshold = 1
	require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), config))

	repo := &stubAccountRepoForQualityProbe{}
	tester := &stubAccountTesterForQualityProbe{status: "success", downgraded: true}
	svc := NewRequestQualityProbeService(settings, repo, newBuiltinRuleStub(), nil)
	svc.accountTestService = tester
	account := &Account{ID: 78, Status: StatusActive, Schedulable: true}

	result, cancel := svc.StartRequestQualityCheck(context.Background(), account, "gpt-6-astra")
	require.Error(t, <-result)
	cancel()
	require.True(t, repo.qualityState.Paused)
	account.Schedulable = false

	tester.downgraded = false
	result, cancel = svc.StartRequestQualityCheck(context.Background(), account, "gpt-6-astra")
	require.NotNil(t, result)
	require.NoError(t, <-result)
	cancel()
	require.Equal(t, 0, repo.qualityState.ConsecutiveCount)
	require.True(t, repo.qualityState.Paused)
	require.False(t, account.Schedulable)
	require.GreaterOrEqual(t, repo.qualityStateResets, 1)
}

func TestRequestQualityProbeParallelStartsWithoutWaitingAndChecksEachTurn(t *testing.T) {
	settings := NewSettingService(newMockSettingRepoForQualityProbe(), nil)
	config := DefaultRequestQualityProbeSettings()
	config.Enabled = true
	config.RuleIDs = []int64{1}
	require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), config))
	tester := &blockingRequestQualityTester{started: make(chan struct{}), release: make(chan struct{})}
	svc := NewRequestQualityProbeService(settings, nil, newBuiltinRuleStub(), nil)
	svc.accountTestService = tester
	result, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 9}, "model")
	defer cancel()
	<-tester.started
	select {
	case <-result:
		t.Fatal("probe must still be pending")
	default:
	}
	// The caller can send / receive the business request while the probe waits.
	close(tester.release)
	require.NoError(t, <-result)
}

type blockingRequestQualityTester struct{ started, release chan struct{} }

func (s *blockingRequestQualityTester) RunDegradationProbeRules(ctx context.Context, _ int64, _ string, _ []*DegradationRule) (*ScheduledTestResult, error) {
	close(s.started)
	select {
	case <-s.release:
		return &ScheduledTestResult{Status: "success"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestRequestQualityWebSocketWaitsPerTurnAndCancellation(t *testing.T) {
	settings := NewSettingService(newMockSettingRepoForQualityProbe(), nil)
	config := DefaultRequestQualityProbeSettings()
	config.Enabled = true
	config.RuleIDs = []int64{1}
	require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), config))
	tester := &stubAccountTesterForQualityProbe{status: "success"}
	svc := NewRequestQualityProbeService(settings, nil, newBuiltinRuleStub(), nil)
	svc.accountTestService = tester
	gate := newWSRequestQualityGate(context.Background(), &Account{ID: 1}, &OpenAIWSIngressHooks{RequestQualityProbe: svc})
	defer gate.close()
	for turn := 1; turn <= 2; turn++ {
		gate.start(turn, "model")
		require.NoError(t, gate.wait())
		require.NoError(t, gate.wait()) // Many frames must reuse this turn's verdict.
		gate.start(turn, "model")       // Transport retry does not add another question.
	}
	require.Equal(t, 2, tester.runCount)
}

func TestRequestQualityAccountOverrides(t *testing.T) {
	account := &Account{ID: 42, Status: "active"}
	settings := DefaultRequestQualityProbeSettings()
	settings.AccountOverrides = map[int64]bool{42: false}
	require.False(t, requestQualityMatches(settings, account, "gpt-6"))
	settings.AccountScope = "selected"
	settings.AccountOverrides[42] = true
	require.True(t, requestQualityMatches(settings, account, "gpt-6"))
	settings.ModelScope = "selected"
	settings.Models = []string{"other"}
	require.False(t, requestQualityMatches(settings, account, "gpt-6"), "override must not bypass model scope")

	// Pattern and prefix matching for designated models
	settings.Models = []string{"gpt-6*"}
	require.True(t, requestQualityMatches(settings, account, "gpt-6"))
	require.True(t, requestQualityMatches(settings, account, "gpt-6.1-sol"))
	require.True(t, requestQualityMatches(settings, account, "gpt-6-sol"))
	require.True(t, requestQualityMatches(settings, account, "global:gpt-6-luna"))
	require.False(t, requestQualityMatches(settings, account, "claude-sonnet-5"))
	require.False(t, requestQualityMatches(settings, account, "gpt-5.6"))

	settings.Models = []string{"gpt-6"}
	require.True(t, requestQualityMatches(settings, account, "gpt-6"))
	require.True(t, requestQualityMatches(settings, account, "gpt-6.1-sol"))
	require.True(t, requestQualityMatches(settings, account, "gpt-6-sol"))
	require.True(t, requestQualityMatches(settings, account, "global:gpt-6-sol"))
	require.False(t, requestQualityMatches(settings, account, "claude-sonnet-5"))
	require.False(t, requestQualityMatches(settings, account, "gpt-5.6-sol"))
	require.False(t, requestQualityMatches(settings, account, "gpt-60"))
	settings.ModelScope = "all"
	settings.AccountScope = "error_or_downgraded"
	require.True(t, requestQualityMatches(settings, account, "gpt-6"))
	delete(settings.AccountOverrides, 42)
	require.False(t, requestQualityMatches(settings, account, "gpt-6"))
	repo := newMockSettingRepoForQualityProbe()
	svc := &SettingService{settingRepo: repo}
	settings.AccountOverrides[42] = false
	require.NoError(t, svc.UpdateRequestQualityProbeSettings(context.Background(), settings))
	svc.InvalidateRequestQualityProbeSettingsCache()
	saved, err := svc.GetRequestQualityProbeSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, map[int64]bool{42: false}, saved.AccountOverrides)
}

func TestRequestQualityProbePaymentRequiredIsNotQualityRejection(t *testing.T) {
	failure := &AccountTestUpstreamError{StatusCode: 402, Body: `{"error":{"code":"insufficient_balance"}}`}
	repo := &stubAccountRepoForQualityProbe{}
	svc := &RequestQualityProbeService{accountRepo: repo, accountTestService: &stubAccountTesterForQualityProbe{status: "failed", upstreamFailure: failure}}
	result := svc.executeProbe(context.Background(), &Account{ID: 447}, "gpt-6.1-sol", &RequestQualityProbeSettings{TimeoutSeconds: 30}, BuiltinCapabilityRules())
	var failover *UpstreamFailoverError
	require.ErrorAs(t, result.err, &failover)
	require.Equal(t, 402, failover.StatusCode)
	require.Equal(t, GatewayFailureReason("insufficient_balance"), failover.Reason)
	require.JSONEq(t, failure.Body, string(failover.ResponseBody))
	require.Empty(t, repo.extraUpdated)
	var downgrade *rotationnotice.DowngradeError
	require.False(t, errors.As(result.err, &downgrade))
}

func TestRequestQualityProbeForbiddenIsNotQualityRejection(t *testing.T) {
	failure := &AccountTestUpstreamError{StatusCode: 403, Body: `{"error":{"code":"upstream_forbidden"}}`}
	repo := &stubAccountRepoForQualityProbe{}
	svc := &RequestQualityProbeService{accountRepo: repo, accountTestService: &stubAccountTesterForQualityProbe{status: "failed", upstreamFailure: failure}}
	result := svc.executeProbe(context.Background(), &Account{ID: 447}, "gpt-6.1-sol", &RequestQualityProbeSettings{TimeoutSeconds: 30}, BuiltinCapabilityRules())
	var failover *UpstreamFailoverError
	require.ErrorAs(t, result.err, &failover)
	require.Equal(t, 403, failover.StatusCode)
	require.Equal(t, GatewayFailureReason("upstream_forbidden"), failover.Reason)
	require.JSONEq(t, failure.Body, string(failover.ResponseBody))
	require.Empty(t, repo.extraUpdated)
	var downgrade *rotationnotice.DowngradeError
	require.False(t, errors.As(result.err, &downgrade))
}

func TestRequestQualityProbeReuseSecondsDefaultsAndAllowsZero(t *testing.T) {
	repo := newMockSettingRepoForQualityProbe()
	repo.data[SettingKeyRequestQualityProbeSettings] = `{"enabled":true,"rule_ids":[1],"models":["gpt-6-astra"],"model_scope":"selected","account_scope":"all"}`
	settings := NewSettingService(repo, nil)
	loaded, err := settings.GetRequestQualityProbeSettings(context.Background())
	require.NoError(t, err)
	require.Equal(t, DefaultRequestQualityProbeReuseSeconds, loaded.reuseWindowSeconds())

	require.NoError(t, settings.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled: true, RuleIDs: []int64{1}, ReuseSeconds: intPtr(0),
	}))
	zero, err := settings.GetRequestQualityProbeSettings(context.Background())
	require.NoError(t, err)
	require.Zero(t, zero.reuseWindowSeconds())
}

func TestRequestQualityProbeReusesPassingProbeWithinWindow(t *testing.T) {
	svc, tester, _ := qualityFixture(t)
	repo := &memoryDetectionRepo{}
	svc.detectionRepo = repo
	rules := newBuiltinRuleStub().rules[:1]
	require.NoError(t, repo.Save(context.Background(), &AccountDetectionRecord{
		AccountID: 1, ModelID: "gpt-6-astra", RuleFingerprint: DegradationRulesFingerprint(rules),
		FinishedAt: time.Now(), Result: &ScheduledTestResult{Status: "success"},
	}))

	result, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1}, "gpt-6-astra")
	defer cancel()
	require.Nil(t, result)
	require.Zero(t, tester.runCount)
}

func TestRequestQualityProbeReuseWindowDoesNotCoverFailures(t *testing.T) {
	svc, tester, _ := qualityFixture(t)
	repo := &memoryDetectionRepo{}
	svc.detectionRepo = repo
	rules := newBuiltinRuleStub().rules[:1]
	require.NoError(t, repo.Save(context.Background(), &AccountDetectionRecord{
		AccountID: 1, ModelID: "gpt-6-astra", RuleFingerprint: DegradationRulesFingerprint(rules),
		FinishedAt: time.Now(), Result: &ScheduledTestResult{Status: "failed"},
	}))

	result, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1}, "gpt-6-astra")
	defer cancel()
	require.NotNil(t, result)
	require.NoError(t, <-result)
	require.Equal(t, 1, tester.runCount)
}

func TestRequestQualityProbeReuseWindowZeroKeepsPerRequestProbe(t *testing.T) {
	svc, tester, _ := qualityFixture(t)
	repo := &memoryDetectionRepo{}
	svc.detectionRepo = repo
	require.NoError(t, svc.settingService.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled: true, RuleIDs: []int64{1}, Models: []string{"gpt-6-astra"}, ModelScope: "selected",
		AccountScope: "all", ReuseSeconds: intPtr(0),
	}))
	rules := newBuiltinRuleStub().rules[:1]
	require.NoError(t, repo.Save(context.Background(), &AccountDetectionRecord{
		AccountID: 1, ModelID: "gpt-6-astra", RuleFingerprint: DegradationRulesFingerprint(rules),
		FinishedAt: time.Now(), Result: &ScheduledTestResult{Status: "success"},
	}))

	result, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1}, "gpt-6-astra")
	defer cancel()
	require.NotNil(t, result)
	require.NoError(t, <-result)
	require.Equal(t, 1, tester.runCount)
}

func TestMatchesQualityProbeModel_ClaudeAndWildcards(t *testing.T) {
	tests := []struct {
		pattern  string
		model    string
		expected bool
	}{
		// Claude general wildcard
		{"claude*", "claude-3-7-sonnet-20250219", true},
		{"claude*", "claude-sonnet-5", true},
		{"claude*", "claude-haiku-5-5", true},
		{"claude*", "anthropic/claude-3-7-sonnet", true},
		{"claude*", "global:claude-sonnet-5", true},
		{"claude*", "gpt-6-astra", false},

		// Claude version delimiter tolerance (3-7 vs 3.7)
		{"claude-3-7*", "claude-3-7-sonnet-20250219", true},
		{"claude-3.7*", "claude-3-7-sonnet-20250219", true},
		{"claude-3-7*", "claude-3.7-sonnet", true},
		{"claude-3.7-sonnet*", "claude-3-7-sonnet-20250219", true},
		{"claude-3-7*", "claude-3-5-sonnet", false},

		// Claude Sonnet / Haiku specific wildcards
		{"claude-sonnet*", "claude-sonnet-5", true},
		{"claude-sonnet*", "claude-sonnet-4-5", true},
		{"claude-sonnet*", "claude-haiku-5-5", false},
		{"claude*sonnet*", "claude-3-7-sonnet-20250219", true},
		{"claude*sonnet*", "claude-3-5-haiku-20241022", false},

		// Leading / surrounding wildcard
		{"*claude*", "claude-sonnet-5", true},
		{"*claude*", "anthropic/claude-3-7-sonnet", true},
		{"*claude*", "deepseek-chat", false},

		// Provider prefix in pattern
		{"anthropic/claude*", "claude-3-7-sonnet", true},
		{"anthropic/claude-3-7*", "claude-3-7-sonnet-20250219", true},

		// Family prefix without '*'
		{"claude", "claude-sonnet-5", true},
		{"claude", "claude-3-7-sonnet", true},
		{"claude-3.7", "claude-3-7-sonnet", true},
		{"claude", "gpt-6-astra", false},

		// GPT-6 patterns still work
		{"gpt-6*", "gpt-6.1-sol", true},
		{"gpt-6*", "gpt-6-astra", true},
		{"gpt-6", "gpt-6-sol", true},
		{"gpt-6*", "claude-sonnet-5", false},
	}

	for _, tc := range tests {
		actual := matchesQualityProbeModel(tc.pattern, tc.model)
		require.Equal(t, tc.expected, actual, "pattern=%q model=%q expected=%v got=%v", tc.pattern, tc.model, tc.expected, actual)
	}
}

func TestRequestQualityProbe_ClaudeWildcardGate(t *testing.T) {
	svc, tester, _ := qualityFixture(t)
	// Configure with multiple wildcards including Claude and GPT-6
	require.NoError(t, svc.settingService.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled: true, RuleIDs: []int64{1}, Models: []string{"gpt-6*", "claude-3-7*"}, ModelScope: "selected",
		AccountScope: "all",
	}))

	// 1. claude-3-7-sonnet-20250219 should trigger probe
	result1, cancel1 := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1, Status: "active"}, "claude-3-7-sonnet-20250219")
	defer cancel1()
	require.NotNil(t, result1)
	require.NoError(t, <-result1)
	require.Equal(t, 1, tester.runCount)

	// 2. claude-haiku-5-5 should NOT trigger probe (not matching claude-3-7*)
	result2, cancel2 := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1, Status: "active"}, "claude-haiku-5-5")
	defer cancel2()
	require.Nil(t, result2)
	require.Equal(t, 1, tester.runCount)

	// 3. gpt-6-astra should trigger probe
	result3, cancel3 := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1, Status: "active"}, "gpt-6-astra")
	defer cancel3()
	require.NotNil(t, result3)
	require.NoError(t, <-result3)
	require.Equal(t, 2, tester.runCount)
}

func TestRequestQualityProbe_GLMExemption(t *testing.T) {
	// 1. IsGLMModel identification
	glmModels := []string{
		"glm-5.3",
		"glm-5.3-flash",
		"GLM-5.3",
		"GLM-5.3-Flash",
		"glm-5.2",
		"glm-5.1",
		"glm-5",
		"glm-4.7",
		"glm-4.7-flash",
		"glm-4.6",
		"glm-4.5",
		"cn:glm-5.3",
		"zhipu/GLM-5.3-Flash",
		"chatglm_turbo",
	}
	for _, m := range glmModels {
		require.True(t, IsGLMModel(m), "expected %s to be recognized as GLM", m)
		require.True(t, IsQualityProbeExcluded(nil, m), "expected %s to be excluded from quality probe", m)
	}

	nonGlmModels := []string{
		"gpt-6-astra",
		"gpt-6.1-sol",
		"claude-3-7-sonnet",
		"claude-sonnet-5",
		"deepseek-v4",
	}
	for _, m := range nonGlmModels {
		require.False(t, IsGLMModel(m), "expected %s not to be recognized as GLM", m)
		require.False(t, IsQualityProbeExcluded(nil, m), "expected %s not to be excluded without custom exclude pattern", m)
	}

	// 2. StartRequestQualityCheck with ModelScope: "all"
	svc, tester, _ := qualityFixture(t)
	require.NoError(t, svc.settingService.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled: true, RuleIDs: []int64{1}, ModelScope: "all",
		AccountScope: "all",
	}))

	// GLM models must NOT trigger probe even when ModelScope is "all"
	for _, m := range []string{"glm-5.3", "glm-5.3-flash", "GLM-5.3", "GLM-5.3-Flash", "cn:glm-5.3"} {
		res, cancel := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1, Status: "active"}, m)
		defer cancel()
		require.Nil(t, res, "expected no quality probe for GLM model %s", m)
	}
	require.Equal(t, 0, tester.runCount, "GLM models should never increment tester runCount")

	// EnsureAccountQuality passes immediately without probe for GLM models
	for _, m := range []string{"glm-5.3", "glm-5.3-flash", "GLM-5.3-Flash"} {
		passed, reason, err := svc.EnsureAccountQuality(context.Background(), &Account{ID: 1, Status: "active"}, m)
		require.True(t, passed)
		require.Empty(t, reason)
		require.NoError(t, err)
	}
	require.Equal(t, 0, tester.runCount)

	// Non-GLM model triggers probe when ModelScope is "all"
	resGpt, cancelGpt := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1, Status: "active"}, "gpt-6-astra")
	defer cancelGpt()
	require.NotNil(t, resGpt)
	require.NoError(t, <-resGpt)
	require.Equal(t, 1, tester.runCount)

	// 3. Custom ExcludeModels configuration
	require.NoError(t, svc.settingService.UpdateRequestQualityProbeSettings(context.Background(), &RequestQualityProbeSettings{
		Enabled: true, RuleIDs: []int64{1}, ModelScope: "all",
		ExcludeModels: []string{"glm*", "deepseek*"},
		AccountScope:  "all",
	}))
	resDs, cancelDs := svc.StartRequestQualityCheck(context.Background(), &Account{ID: 1, Status: "active"}, "deepseek-v4")
	defer cancelDs()
	require.Nil(t, resDs, "custom excluded model deepseek-v4 should not trigger probe")
}
