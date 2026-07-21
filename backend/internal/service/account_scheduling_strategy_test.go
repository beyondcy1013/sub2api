package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func schedulingTestAccount(id int64, rate float64, super bool) *Account {
	account := &Account{
		ID:             id,
		RateMultiplier: &rate,
		Extra:          map[string]any{},
	}
	if super {
		account.Extra[SuperPriorityExtraKey] = true
	}
	return account
}

func schedulingTestLoad(account *Account) accountWithLoad {
	return accountWithLoad{account: account, loadInfo: &AccountLoadInfo{AccountID: account.ID}}
}

func accountLoadIDs(items []accountWithLoad) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.account.ID)
	}
	return ids
}

func accountIDs(items []*Account) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func openAISelectionIDs(items []openAIAccountCandidateScore) []int64 {
	ids := make([]int64, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.account.ID)
	}
	return ids
}

func TestFilterByAccountSchedulingPreference_LowestCostIgnoresLegacySuperPriorityOverlay(t *testing.T) {
	superExpensive := schedulingTestAccount(1, 0.8, true)
	ordinaryCheap := schedulingTestAccount(2, 0.1, false)
	ordinaryExpensive := schedulingTestAccount(3, 0.5, false)
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}}

	first := filterByAccountSchedulingPreference([]accountWithLoad{
		schedulingTestLoad(ordinaryExpensive),
		schedulingTestLoad(superExpensive),
		schedulingTestLoad(ordinaryCheap),
	}, cfg)
	require.Equal(t, []int64{2}, accountLoadIDs(first))

	fallback := filterByAccountSchedulingPreference([]accountWithLoad{
		schedulingTestLoad(ordinaryExpensive),
		schedulingTestLoad(ordinaryCheap),
	}, cfg)
	require.Equal(t, []int64{2}, accountLoadIDs(fallback))
}

func TestOrderAccountsBySchedulingPreference_PreservesDefaultOrderWithinTiers(t *testing.T) {
	accounts := []*Account{
		schedulingTestAccount(1, 0.7, false),
		schedulingTestAccount(2, 0.8, true),
		schedulingTestAccount(3, 0.1, false),
		schedulingTestAccount(4, 0.2, true),
	}
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}}

	orderAccountsBySchedulingPreference(accounts, cfg)
	require.Equal(t, []int64{3, 4, 1, 2}, accountIDs(accounts))

	defaultAccounts := []*Account{accounts[3], accounts[2], accounts[1], accounts[0]}
	before := accountIDs(defaultAccounts)
	orderAccountsBySchedulingPreference(defaultAccounts, &config.Config{})
	require.Equal(t, before, accountIDs(defaultAccounts))
}

func TestBuildOpenAISelectionOrder_SuperPriorityFallbackSurvivesTopK(t *testing.T) {
	superAccount := schedulingTestAccount(1, 1, true)
	baseAccount := schedulingTestAccount(2, 1, false)
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyDefault,
	}}
	scheduler := &defaultOpenAIAccountScheduler{service: &OpenAIGatewayService{cfg: cfg}}
	plan := openAIAccountLoadPlan{
		candidates: []openAIAccountCandidateScore{
			{account: baseAccount, loadInfo: &AccountLoadInfo{}, score: 100},
			{account: superAccount, loadInfo: &AccountLoadInfo{}, score: 1},
		},
		topK: 1,
	}

	order := scheduler.buildOpenAISelectionOrder(OpenAIAccountScheduleRequest{}, plan)
	require.Equal(t, []int64{1, 2}, openAISelectionIDs(order))
}

func TestBuildOpenAISelectionOrder_LowestCostFallbackSurvivesTopK(t *testing.T) {
	cheap := schedulingTestAccount(1, 0.1, false)
	expensive := schedulingTestAccount(2, 0.9, false)
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}}
	scheduler := &defaultOpenAIAccountScheduler{service: &OpenAIGatewayService{cfg: cfg}}
	plan := openAIAccountLoadPlan{
		candidates: []openAIAccountCandidateScore{
			{account: expensive, loadInfo: &AccountLoadInfo{}, score: 100},
			{account: cheap, loadInfo: &AccountLoadInfo{}, score: 1},
		},
		topK: 1,
	}

	order := scheduler.buildOpenAISelectionOrder(OpenAIAccountScheduleRequest{}, plan)
	require.Equal(t, []int64{1, 2}, openAISelectionIDs(order))
}

func TestBuildOpenAISelectionOrder_SuperPriorityPrecedesSubscriptionPreference(t *testing.T) {
	superAccount := schedulingTestAccount(1, 1, true)
	superAccount.Platform = PlatformOpenAI
	superAccount.Type = AccountTypeAPIKey
	subscriptionAccount := schedulingTestAccount(2, 1, false)
	subscriptionAccount.Platform = PlatformOpenAI
	subscriptionAccount.Type = AccountTypeOAuth
	subscriptionAccount.Credentials = map[string]any{"plan_type": "plus"}
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyDefault,
	}}
	scheduler := &defaultOpenAIAccountScheduler{service: &OpenAIGatewayService{cfg: cfg}}
	plan := openAIAccountLoadPlan{
		candidates: []openAIAccountCandidateScore{
			{account: subscriptionAccount, loadInfo: &AccountLoadInfo{}, score: 100},
			{account: superAccount, loadInfo: &AccountLoadInfo{}, score: 1},
		},
		topK: 1,
	}

	order := scheduler.buildOpenAISelectionOrder(OpenAIAccountScheduleRequest{SubscriptionPriority: true}, plan)
	require.Equal(t, []int64{1, 2}, openAISelectionIDs(order))
}

func TestOpenAILegacyWithoutLoadBatch_FallsBackAfterCheapestAccountIsFull(t *testing.T) {
	superRate := 0.8
	cheapRate := 0.1
	accounts := []Account{
		{
			ID:             11,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeAPIKey,
			Status:         StatusActive,
			Schedulable:    true,
			Concurrency:    1,
			RateMultiplier: &superRate,
			Extra:          map[string]any{SuperPriorityExtraKey: true},
		},
		{
			ID:             12,
			Platform:       PlatformOpenAI,
			Type:           AccountTypeAPIKey,
			Status:         StatusActive,
			Schedulable:    true,
			Concurrency:    1,
			RateMultiplier: &cheapRate,
		},
	}
	var acquired []int64
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	service := &OpenAIGatewayService{
		accountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:       &schedulerTestGatewayCache{},
		cfg:         cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{
			acquireResults: map[int64]bool{11: true, 12: false},
			acquiredIDs:    &acquired,
		}),
	}
	groupID := int64(1)

	selection, err := service.selectAccountWithLoadAwareness(
		context.Background(),
		&groupID,
		PlatformOpenAI,
		"",
		"",
		nil,
		false,
		OpenAIEndpointCapabilityChatCompletions,
		false,
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.True(t, selection.Acquired)
	require.Equal(t, int64(11), selection.Account.ID)
	require.Equal(t, []int64{12, 11}, acquired)
}

func TestAccountSchedulingRate_UsesManualSourceByDefault(t *testing.T) {
	rate := 0.35
	account := &Account{RateMultiplier: &rate}

	got, known, source := account.SchedulingRate(time.Now())

	require.True(t, known)
	require.Equal(t, 0.35, got)
	require.Equal(t, SchedulingRateSourceManual, source)
}

func TestAccountSchedulingRate_UsesFreshUpstreamSnapshotAndPeakMultiplier(t *testing.T) {
	rate := 0.9
	receivedAt := time.Date(2026, 7, 22, 1, 0, 0, 0, time.UTC)
	freshUntil := receivedAt.Add(6 * time.Hour)
	account := &Account{
		RateMultiplier: &rate,
		Extra: map[string]any{
			SchedulingRateSourceExtraKey: SchedulingRateSourceUpstream,
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      UpstreamBillingProbeStatusOK,
				"received_at": receivedAt,
				"fresh_until": freshUntil,
				"data": map[string]any{
					"billing_scope":            "token",
					"resolved_rate_multiplier": 0.4,
					"peak_rate_enabled":        true,
					"peak_start":               "09:00",
					"peak_end":                 "18:00",
					"peak_rate_multiplier":     1.5,
					"timezone":                 "Asia/Shanghai",
				},
			},
		},
	}

	got, known, source := account.SchedulingRate(time.Date(2026, 7, 22, 4, 0, 0, 0, time.UTC))

	require.True(t, known)
	require.InDelta(t, 0.6, got, 1e-9)
	require.Equal(t, SchedulingRateSourceUpstream, source)
}

func TestAccountSchedulingRate_StaleOrUnsupportedUpstreamIsUnknown(t *testing.T) {
	for _, status := range []string{UpstreamBillingProbeStatusUnsupported, UpstreamBillingProbeStatusFailed} {
		account := &Account{Extra: map[string]any{
			SchedulingRateSourceExtraKey: SchedulingRateSourceUpstream,
			UpstreamBillingProbeExtraKey: map[string]any{
				"status":      status,
				"received_at": time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
				"fresh_until": time.Date(2026, 7, 21, 1, 0, 0, 0, time.UTC),
			},
		}}
		_, known, source := account.SchedulingRate(time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC))
		require.False(t, known)
		require.Equal(t, SchedulingRateSourceUpstream, source)
	}
}

func TestOrderAccountLoadsBySchedulingPreference_PutsUnknownRatesLast(t *testing.T) {
	knownRate := 0.4
	unknown := &Account{ID: 1, RateMultiplier: func() *float64 { v := 0.1; return &v }(), Extra: map[string]any{
		SchedulingRateSourceExtraKey: SchedulingRateSourceUpstream,
		UpstreamBillingProbeExtraKey: map[string]any{"status": UpstreamBillingProbeStatusUnsupported},
	}}
	cheap := &Account{ID: 2, RateMultiplier: &knownRate}
	expensive := &Account{ID: 3, RateMultiplier: func() *float64 { v := 0.8; return &v }()}
	items := []accountWithLoad{schedulingTestLoad(unknown), schedulingTestLoad(expensive), schedulingTestLoad(cheap)}
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{BaseStrategy: AccountSchedulingStrategyLowestCost}}

	orderAccountLoadsBySchedulingPreference(items, cfg)

	require.Equal(t, []int64{2, 3, 1}, accountLoadIDs(items))
}

func TestOpenAIAdvancedLowestCost_BypassesMovableSessionSticky(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	groupID := int64(88)
	expensiveRate, cheapRate := 0.9, 0.1
	accounts := []Account{
		{ID: 881, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}, RateMultiplier: &expensiveRate},
		{ID: 882, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}, RateMultiplier: &cheapRate},
	}
	cache := &schedulerTestGatewayCache{sessionBindings: map[string]int64{"openai:cost-session": 881}}
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{BaseStrategy: AccountSchedulingStrategyLowestCost}}
	cfg.RunMode = config.RunModeSimple
	cfg.Gateway.OpenAIWS.LBTopK = 2
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:              cache,
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true", "false"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}

	scheduler := &defaultOpenAIAccountScheduler{service: svc, stats: newOpenAIAccountRuntimeStats()}
	stickySelection, _, stickyErr := scheduler.selectBySessionHash(context.Background(), OpenAIAccountScheduleRequest{
		GroupID: &groupID, SessionHash: "cost-session", RequestedModel: "gpt-5.1", Platform: PlatformOpenAI,
		RequiredTransport: OpenAIUpstreamTransportAny,
	})
	require.NoError(t, stickyErr)
	require.NotNil(t, stickySelection)
	require.Equal(t, int64(881), stickySelection.Account.ID)
	if stickySelection.ReleaseFunc != nil {
		stickySelection.ReleaseFunc()
	}
	selection, decision, err := scheduler.Select(context.Background(), OpenAIAccountScheduleRequest{
		GroupID:           &groupID,
		SessionHash:       "cost-session",
		RequestedModel:    "gpt-5.1",
		RequiredTransport: OpenAIUpstreamTransportAny,
		Platform:          PlatformOpenAI,
		StickyWeighted:    false,
	})

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.Equal(t, int64(882), selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIAdvancedLowestCost_PreservesStrictPreviousResponseAffinity(t *testing.T) {
	resetOpenAIAdvancedSchedulerSettingCacheForTest()
	ctx := context.Background()
	groupID := int64(89)
	expensiveRate, cheapRate := 0.9, 0.1
	expensive := Account{ID: 891, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}, RateMultiplier: &expensiveRate, Extra: map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}}
	cheap := Account{ID: 892, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Concurrency: 1, GroupIDs: []int64{groupID}, RateMultiplier: &cheapRate}
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{BaseStrategy: AccountSchedulingStrategyLowestCost}}
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.StickyResponseIDTTLSeconds = 3600
	svc := &OpenAIGatewayService{
		accountRepo:        schedulerTestOpenAIAccountRepo{accounts: []Account{expensive, cheap}},
		cache:              &schedulerTestGatewayCache{},
		cfg:                cfg,
		rateLimitService:   newOpenAIAdvancedSchedulerRateLimitService("true"),
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{}),
	}
	require.NoError(t, svc.getOpenAIWSStateStore().BindResponseAccount(ctx, groupID, "resp_cost_strict", expensive.ID, time.Hour))

	selection, decision, err := svc.SelectAccountWithScheduler(
		ctx, &groupID, "resp_cost_strict", "cost-session", "gpt-5.1", nil,
		OpenAIUpstreamTransportAny, false,
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.Equal(t, expensive.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerPreviousResponse, decision.Layer)
	require.True(t, decision.StickyPreviousHit)
	if selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestGatewayLegacyWithoutLoadBatch_FallsBackAfterCheapestAccountIsFull(t *testing.T) {
	superRate := 0.8
	cheapRate := 0.1
	accounts := []Account{
		{
			ID:             21,
			Platform:       PlatformAnthropic,
			Status:         StatusActive,
			Schedulable:    true,
			Priority:       2,
			Concurrency:    1,
			RateMultiplier: &superRate,
			Extra:          map[string]any{SuperPriorityExtraKey: true},
		},
		{
			ID:             22,
			Platform:       PlatformAnthropic,
			Status:         StatusActive,
			Schedulable:    true,
			Priority:       1,
			Concurrency:    1,
			RateMultiplier: &cheapRate,
		},
	}
	repo := schedulerTestOpenAIAccountRepo{accounts: accounts}
	var acquired []int64
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	service := &GatewayService{
		accountRepo: repo,
		cache:       &schedulerTestGatewayCache{},
		cfg:         cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{
			acquireResults: map[int64]bool{21: true, 22: false},
			acquiredIDs:    &acquired,
		}),
	}

	ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAnthropic)
	selection, err := service.SelectAccountWithLoadAwareness(
		ctx,
		nil,
		"",
		"claude-3-5-sonnet-20241022",
		nil,
		"",
		0,
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.True(t, selection.Acquired)
	require.Equal(t, int64(21), selection.Account.ID)
	require.Equal(t, []int64{22, 21}, acquired)
}

func TestGatewayLegacyWithoutLoadBatch_WaitsOnCheapestAccountAfterAllAccountsAreFull(t *testing.T) {
	superRate := 0.8
	cheapRate := 0.1
	accounts := []Account{
		{
			ID:             31,
			Platform:       PlatformAnthropic,
			Status:         StatusActive,
			Schedulable:    true,
			Priority:       2,
			Concurrency:    1,
			RateMultiplier: &superRate,
			Extra:          map[string]any{SuperPriorityExtraKey: true},
		},
		{
			ID:             32,
			Platform:       PlatformAnthropic,
			Status:         StatusActive,
			Schedulable:    true,
			Priority:       1,
			Concurrency:    1,
			RateMultiplier: &cheapRate,
		},
	}
	var acquired []int64
	cfg := &config.Config{SuperPriority: config.SuperPriorityConfig{
		Mode:         superPriorityModeSuperPriority,
		BaseStrategy: AccountSchedulingStrategyLowestCost,
	}}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	service := &GatewayService{
		accountRepo: schedulerTestOpenAIAccountRepo{accounts: accounts},
		cache:       &schedulerTestGatewayCache{},
		cfg:         cfg,
		concurrencyService: NewConcurrencyService(schedulerTestConcurrencyCache{
			acquireResults: map[int64]bool{31: false, 32: false},
			acquiredIDs:    &acquired,
		}),
	}
	ctx := context.WithValue(context.Background(), ctxkey.ForcePlatform, PlatformAnthropic)

	selection, err := service.SelectAccountWithLoadAwareness(
		ctx,
		nil,
		"",
		"claude-3-5-sonnet-20241022",
		nil,
		"",
		0,
	)

	require.NoError(t, err)
	require.NotNil(t, selection)
	require.False(t, selection.Acquired)
	require.NotNil(t, selection.WaitPlan)
	require.Equal(t, int64(32), selection.Account.ID)
	require.Equal(t, int64(32), selection.WaitPlan.AccountID)
	require.Equal(t, []int64{32, 31}, acquired)
}
