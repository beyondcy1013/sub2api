package xai

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// CLI client identity required by cli-chat-proxy billing endpoints.
	CLITokenAuthHeader     = "x-xai-token-auth"
	CLITokenAuthValue      = "xai-grok-cli"
	CLIClientVersionHeader = "x-grok-client-version"
	// CLIClientVersion is the one place the pinned Grok CLI version lives. The
	// repository and service layers build their own client identity from it, so
	// one bump here covers OAuth traffic and billing probes together.
	// Keep in sync with https://x.ai/cli/stable.
	CLIClientVersion = "1.0.46"

	BillingWeeklyPath  = "/billing?format=credits"
	BillingMonthlyPath = "/billing"

	SuperGrokLimitCents      = 15_000  // $150.00
	SuperGrokHeavyLimitCents = 150_000 // $1,500.00

	SubscriptionPath = "/user?include=subscription"
)

// BillingPeriod describes the current weekly/monthly window.
type BillingPeriod struct {
	Type  string `json:"type,omitempty"`
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// BillingProductUsage is per-product usage inside the weekly credits window.
type BillingProductUsage struct {
	Product      string          `json:"product,omitempty"`
	UsagePercent *float64        `json:"usagePercent,omitempty"`
	Used         json.RawMessage `json:"used,omitempty"`
	Total        json.RawMessage `json:"total,omitempty"`
	Remaining    json.RawMessage `json:"remaining,omitempty"`
}

// BillingConfig is the nested config object from /v1/billing responses.
// Weekly (`?format=credits`) and monthly (`/billing`) share this shape; absolute
// money fields typically appear on the credits (prepaid/on-demand) or monthly
// (limit/used) responses.
type BillingConfig struct {
	CurrentPeriod        *BillingPeriod        `json:"currentPeriod,omitempty"`
	CreditUsagePercent   *float64              `json:"creditUsagePercent,omitempty"`
	ProductUsage         []BillingProductUsage `json:"productUsage,omitempty"`
	SubscriptionTier     string                `json:"subscriptionTier,omitempty"`
	SubscriptionStatus   string                `json:"subscriptionStatus,omitempty"`
	Credits              json.RawMessage       `json:"credits,omitempty"`
	IncludedCredits      json.RawMessage       `json:"includedCredits,omitempty"`
	SubscriptionCredits  json.RawMessage       `json:"subscriptionCredits,omitempty"`
	WeeklyCredits        json.RawMessage       `json:"weeklyCredits,omitempty"`
	SharedPool           json.RawMessage       `json:"sharedPool,omitempty"`
	MonthlyLimit         json.RawMessage       `json:"monthlyLimit,omitempty"`
	Used                 json.RawMessage       `json:"used,omitempty"`
	OnDemandCap          json.RawMessage       `json:"onDemandCap,omitempty"`
	OnDemandUsed         json.RawMessage       `json:"onDemandUsed,omitempty"`
	PrepaidBalance       json.RawMessage       `json:"prepaidBalance,omitempty"`
	IsUnifiedBillingUser bool                  `json:"isUnifiedBillingUser,omitempty"`
	TopUpMethod          string                `json:"topUpMethod,omitempty"`
	BillingPeriodStart   string                `json:"billingPeriodStart,omitempty"`
	BillingPeriodEnd     string                `json:"billingPeriodEnd,omitempty"`
}

// BillingPayload is the top-level body from /v1/billing.
type BillingPayload struct {
	Config *BillingConfig `json:"config,omitempty"`
}

type SubscriptionPayload struct {
	Email              string           `json:"email,omitempty"`
	User               map[string]any   `json:"user,omitempty"`
	HasGrokCodeAccess  bool             `json:"hasGrokCodeAccess,omitempty"`
	SubscriptionTier   string           `json:"subscriptionTier,omitempty"`
	SubscriptionStatus string           `json:"subscriptionStatus,omitempty"`
	Subscription       map[string]any   `json:"subscription,omitempty"`
	Subscriptions      []map[string]any `json:"subscriptions,omitempty"`
}

type SubscriptionSummary struct {
	Email              string `json:"email,omitempty"`
	SubscriptionTier   string `json:"subscription_tier,omitempty"`
	SubscriptionStatus string `json:"subscription_status,omitempty"`
	HasGrokCodeAccess  bool   `json:"has_grok_code_access,omitempty"`
	StatusCode         int    `json:"status_code,omitempty"`
}

// BillingProductSummary is a normalized product usage row for UI.
type BillingProductSummary struct {
	Product      string   `json:"product"`
	UsagePercent *float64 `json:"usage_percent,omitempty"`
	Used         *float64 `json:"used,omitempty"`
	Total        *float64 `json:"total,omitempty"`
	Remaining    *float64 `json:"remaining,omitempty"`
}

// BillingSummary is the merged weekly + monthly billing view.
// Cents fields remain the authoritative monthly numbers; dollar fields are the
// operator-facing absolute money view (prepaid / on-demand / monthly $).
type BillingSummary struct {
	PeriodType         string                  `json:"period_type,omitempty"` // weekly | monthly | unknown
	UsagePercent       *float64                `json:"usage_percent,omitempty"`
	PeriodStart        string                  `json:"period_start,omitempty"`
	PeriodEnd          string                  `json:"period_end,omitempty"`
	ProductUsage       []BillingProductSummary `json:"product_usage,omitempty"`
	MonthlyLimitCents  *float64                `json:"monthly_limit_cents,omitempty"`
	UsedCents          *float64                `json:"used_cents,omitempty"`
	IncludedUsedCents  *float64                `json:"included_used_cents,omitempty"`
	BillingPeriodStart string                  `json:"billing_period_start,omitempty"`
	BillingPeriodEnd   string                  `json:"billing_period_end,omitempty"`
	UsedPercent        *float64                `json:"used_percent,omitempty"`
	// Absolute money (USD). Prepaid/on-demand come from credits probe as dollars.
	// MonthlyLimit/MonthlyUsed are cents/100 for consistent $ display.
	PrepaidBalance       *float64 `json:"prepaid_balance,omitempty"`
	MonthlyLimit         *float64 `json:"monthly_limit,omitempty"`
	MonthlyUsed          *float64 `json:"monthly_used,omitempty"`
	OnDemandCap          *float64 `json:"on_demand_cap,omitempty"`
	OnDemandUsed         *float64 `json:"on_demand_used,omitempty"`
	TopUpMethod          string   `json:"top_up_method,omitempty"`
	IsUnifiedBillingUser bool     `json:"is_unified_billing_user,omitempty"`
	SubscriptionTier     string   `json:"subscription_tier,omitempty"`
	SubscriptionStatus   string   `json:"subscription_status,omitempty"`
	Email                string   `json:"email,omitempty"`
	WeeklyUsed           *float64 `json:"weekly_used,omitempty"`
	WeeklyTotal          *float64 `json:"weekly_total,omitempty"`
	WeeklyRemaining      *float64 `json:"weekly_remaining,omitempty"`
	HasGrokCodeAccess    bool     `json:"has_grok_code_access,omitempty"`
	Plan                 string   `json:"plan,omitempty"` // SuperGrok | SuperGrok Heavy | ""
	StatusCode           int      `json:"status_code,omitempty"`
	WeeklyStatusCode     int      `json:"weekly_status_code,omitempty"`
	MonthlyStatusCode    int      `json:"monthly_status_code,omitempty"`
	Source               string   `json:"source,omitempty"`
	FetchedAt            string   `json:"fetched_at,omitempty"`
	UpdatedAt            string   `json:"updated_at,omitempty"`
	WeeklyUpdatedAt      string   `json:"weekly_updated_at,omitempty"`
	MonthlyUpdatedAt     string   `json:"monthly_updated_at,omitempty"`
	Partial              bool     `json:"partial,omitempty"`
	FailedWindows        []string `json:"failed_windows,omitempty"`
}

// BuildBillingURL builds weekly or monthly billing URL against the CLI chat proxy.
func BuildBillingURL(formatCredits bool) string {
	base := strings.TrimRight(DefaultCLIBaseURL, "/")
	if formatCredits {
		return base + BillingWeeklyPath
	}
	return base + BillingMonthlyPath
}

// BuildBillingURLWithValidator builds the weekly or monthly billing URL against
// the caller-resolved base URL, applying the caller's outbound URL trust policy
// first. Accounts forwarding through a custom upstream keep their billing
// probes on the same upstream.
func BuildBillingURLWithValidator(baseURL string, formatCredits bool, validator BaseURLValidator) (string, error) {
	validatedBaseURL, err := validatedBaseURLWithValidator(baseURL, validator)
	if err != nil {
		return "", fmt.Errorf("invalid base url: %w", err)
	}
	if formatCredits {
		return validatedBaseURL + BillingWeeklyPath, nil
	}
	return validatedBaseURL + BillingMonthlyPath, nil
}

func BuildSubscriptionURLWithValidator(baseURL string, validator BaseURLValidator) (string, error) {
	validatedBaseURL, err := validatedBaseURLWithValidator(baseURL, validator)
	if err != nil {
		return "", fmt.Errorf("invalid base url: %w", err)
	}
	return validatedBaseURL + SubscriptionPath, nil
}

// ApplyCLIBillingHeaders sets Authorization + CLI identity headers for billing GETs.
func ApplyCLIBillingHeaders(req *http.Request, accessToken string) {
	if req == nil {
		return
	}
	token := strings.TrimSpace(accessToken)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(CLITokenAuthHeader, CLITokenAuthValue)
	req.Header.Set(CLIClientVersionHeader, CLIClientVersion)
	req.Header.Set("User-Agent", CLIUserAgent(CLIClientVersion))
	req.Header.Set("x-grok-client-mode", CLIClientMode)
}

// ApplyCLISubscriptionHeaders adds the identity headers that the CLI proxy
// requires for /v1/user. Missing user identity commonly turns into 403 even
// when the same bearer token works for inference traffic.
func ApplyCLISubscriptionHeaders(req *http.Request, accessToken, userID string) {
	if req == nil {
		return
	}
	ApplyCLIBillingHeaders(req, accessToken)
	if userID = strings.TrimSpace(userID); userID != "" {
		req.Header.Set("X-UserID", userID)
		req.Header.Set("x-authenticateresponse", "authenticate-response")
	}
}

// ParseBillingPayload unmarshals a billing API response body.
func ParseBillingPayload(body []byte) (*BillingPayload, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty billing body")
	}
	var payload BillingPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func ParseSubscriptionSummary(body []byte) (*SubscriptionSummary, error) {
	var payload SubscriptionPayload
	if len(body) == 0 {
		return nil, fmt.Errorf("empty subscription body")
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	subscription := payload.Subscription
	if subscription == nil && payload.User != nil {
		subscription = payload.User
	}
	if subscription == nil {
		for _, item := range payload.Subscriptions {
			if status, _ := item["status"].(string); strings.EqualFold(strings.TrimSpace(status), "SUBSCRIPTION_STATUS_ACTIVE") {
				subscription = item
				break
			}
		}
	}
	getString := func(keys ...string) string {
		object := subscription
		if object == nil {
			object = map[string]any{}
		}
		for _, key := range keys {
			if value, ok := object[key]; ok {
				if text, _ := value.(string); strings.TrimSpace(text) != "" {
					return strings.TrimSpace(text)
				}
			}
		}
		return ""
	}
	return &SubscriptionSummary{
		Email:              strings.TrimSpace(payload.Email),
		SubscriptionTier:   firstNonEmptyString(strings.TrimSpace(payload.SubscriptionTier), getString("tier", "subscriptionTier", "subscription_tier")),
		SubscriptionStatus: getString("status", "subscriptionStatus", "subscription_status"),
		HasGrokCodeAccess:  payload.HasGrokCodeAccess,
	}, nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// BuildBillingSummary normalizes a billing config into a UI-friendly summary.
func BuildBillingSummary(config *BillingConfig) *BillingSummary {
	if config == nil {
		return nil
	}
	summary := &BillingSummary{}
	period := config.CurrentPeriod
	periodType := resolvePeriodType(period)
	creditUsage := cloneFloat(config.CreditUsagePercent)

	// Weekly period bounds must not fall back to monthly billing period ends —
	// that would park accounts on a multi-week horizon when weekly UsagePercent
	// is high (scheduler seven_day uses PeriodEnd).
	periodStart := ""
	periodEnd := ""
	if period != nil {
		periodStart = strings.TrimSpace(period.Start)
		periodEnd = strings.TrimSpace(period.End)
	}

	products := make([]BillingProductSummary, 0, len(config.ProductUsage))
	for _, item := range config.ProductUsage {
		product := strings.TrimSpace(item.Product)
		if product == "" {
			continue
		}
		used, total, remaining := creditAmountsFromRaw(item.Used, item.Total, item.Remaining)
		products = append(products, BillingProductSummary{
			Product:      product,
			UsagePercent: cloneFloat(item.UsagePercent),
			Used:         used,
			Total:        total,
			Remaining:    remaining,
		})
	}

	monthlyLimit := parseCentValue(config.MonthlyLimit)
	used := parseCentValue(config.Used)
	// Absolute money on credits responses is dollar-denominated ({"val": 12}).
	// Monthly limit/used are cents (same as MonthlyLimitCents / UsedCents).
	prepaid := parseCentValue(config.PrepaidBalance)
	onDemandCap := parseCentValue(config.OnDemandCap)
	onDemandUsed := parseCentValue(config.OnDemandUsed)
	weeklyUsed, weeklyTotal, weeklyRemaining := creditAmounts(
		config.Credits, config.IncludedCredits, config.SubscriptionCredits,
		config.WeeklyCredits, config.SharedPool,
	)
	billingStart := strings.TrimSpace(config.BillingPeriodStart)
	billingEnd := strings.TrimSpace(config.BillingPeriodEnd)

	var includedUsed *float64
	if used != nil {
		if monthlyLimit != nil && *monthlyLimit > 0 {
			v := math.Min(*used, *monthlyLimit)
			includedUsed = &v
		} else {
			includedUsed = cloneFloat(used)
		}
	}

	var usedPercent *float64
	if monthlyLimit != nil && *monthlyLimit > 0 && includedUsed != nil {
		v := (*includedUsed / *monthlyLimit) * 100
		usedPercent = &v
	}

	hasWeekly := creditUsage != nil || periodType == "weekly" || len(products) > 0 ||
		prepaid != nil || onDemandCap != nil || onDemandUsed != nil ||
		weeklyUsed != nil || weeklyTotal != nil || weeklyRemaining != nil
	hasMonthly := monthlyLimit != nil || used != nil || (!hasWeekly && billingEnd != "")
	if !hasWeekly && !hasMonthly {
		return nil
	}

	if hasWeekly {
		if periodType == "unknown" {
			periodType = "weekly"
		}
		summary.PeriodType = periodType
		summary.UsagePercent = creditUsage
		summary.PeriodStart = periodStart
		summary.PeriodEnd = periodEnd
	} else {
		// Monthly-only: do not put monthly % into UsagePercent (weekly bar field).
		// Frontend weekly bar only renders when PeriodType == weekly.
		summary.PeriodType = "monthly"
		summary.PeriodStart = billingStart
		summary.PeriodEnd = billingEnd
	}
	summary.ProductUsage = products
	summary.MonthlyLimitCents = monthlyLimit
	summary.UsedCents = used
	summary.IncludedUsedCents = includedUsed
	if hasMonthly {
		summary.BillingPeriodStart = billingStart
		summary.BillingPeriodEnd = billingEnd
	}
	summary.UsedPercent = usedPercent
	summary.PrepaidBalance = prepaid
	if onDemandCap != nil {
		summary.OnDemandCap = onDemandCap
	}
	if onDemandUsed != nil {
		summary.OnDemandUsed = onDemandUsed
	}
	// Expose monthly cents as dollars for UI absolute rows.
	if monthlyLimit != nil {
		v := *monthlyLimit / 100
		summary.MonthlyLimit = &v
	}
	if used != nil {
		v := *used / 100
		summary.MonthlyUsed = &v
	}
	summary.TopUpMethod = strings.TrimSpace(config.TopUpMethod)
	summary.IsUnifiedBillingUser = config.IsUnifiedBillingUser
	summary.SubscriptionTier = strings.TrimSpace(config.SubscriptionTier)
	summary.SubscriptionStatus = strings.TrimSpace(config.SubscriptionStatus)
	summary.WeeklyUsed = weeklyUsed
	summary.WeeklyTotal = weeklyTotal
	summary.WeeklyRemaining = weeklyRemaining
	summary.Plan = resolvePlan(monthlyLimit)
	return summary
}

// MergeBillingProbeResult updates successful billing domains while retaining
// the previous value for any domain that could not be refreshed.
func MergeBillingProbeResult(previous, weekly, monthly *BillingSummary, weeklyOK, monthlyOK bool) *BillingSummary {
	var out BillingSummary
	if previous != nil {
		out = *previous
		previousUpdatedAt := previous.UpdatedAt
		if previousUpdatedAt == "" {
			previousUpdatedAt = previous.FetchedAt
		}
		if out.WeeklyUpdatedAt == "" && (out.UsagePercent != nil || len(out.ProductUsage) > 0) {
			out.WeeklyUpdatedAt = previousUpdatedAt
		}
		if out.MonthlyUpdatedAt == "" && (out.MonthlyLimitCents != nil || out.UsedPercent != nil) {
			out.MonthlyUpdatedAt = previousUpdatedAt
		}
	}
	now := time.Now().UTC().Format(time.RFC3339)

	if weeklyOK && weekly != nil {
		out.PeriodType = weekly.PeriodType
		out.UsagePercent = weekly.UsagePercent
		out.PeriodStart = weekly.PeriodStart
		out.PeriodEnd = weekly.PeriodEnd
		out.ProductUsage = weekly.ProductUsage
		// Absolute prepaid / on-demand usually ride the credits (weekly) response.
		if weekly.PrepaidBalance != nil {
			out.PrepaidBalance = weekly.PrepaidBalance
		}
		if weekly.OnDemandCap != nil {
			out.OnDemandCap = weekly.OnDemandCap
		}
		if weekly.OnDemandUsed != nil {
			out.OnDemandUsed = weekly.OnDemandUsed
		}
		if weekly.TopUpMethod != "" {
			out.TopUpMethod = weekly.TopUpMethod
		}
		if weekly.IsUnifiedBillingUser {
			out.IsUnifiedBillingUser = true
		}
		if weekly.SubscriptionTier != "" {
			out.SubscriptionTier = weekly.SubscriptionTier
		}
		if weekly.SubscriptionStatus != "" {
			out.SubscriptionStatus = weekly.SubscriptionStatus
		}
		if weekly.WeeklyUsed != nil {
			out.WeeklyUsed = weekly.WeeklyUsed
		}
		if weekly.WeeklyTotal != nil {
			out.WeeklyTotal = weekly.WeeklyTotal
		}
		if weekly.WeeklyRemaining != nil {
			out.WeeklyRemaining = weekly.WeeklyRemaining
		}
		out.WeeklyUpdatedAt = now
	}
	if monthlyOK && monthly != nil {
		if out.PeriodType == "" {
			out.PeriodType = "monthly"
		}
		out.MonthlyLimitCents = monthly.MonthlyLimitCents
		out.UsedCents = monthly.UsedCents
		out.IncludedUsedCents = monthly.IncludedUsedCents
		out.BillingPeriodStart = monthly.BillingPeriodStart
		out.BillingPeriodEnd = monthly.BillingPeriodEnd
		out.UsedPercent = monthly.UsedPercent
		out.MonthlyLimit = monthly.MonthlyLimit
		out.MonthlyUsed = monthly.MonthlyUsed
		// Monthly probe may also carry on-demand cap when credits omitted it.
		if monthly.OnDemandCap != nil && out.OnDemandCap == nil {
			out.OnDemandCap = monthly.OnDemandCap
		}
		if monthly.OnDemandUsed != nil && out.OnDemandUsed == nil {
			out.OnDemandUsed = monthly.OnDemandUsed
		}
		if out.SubscriptionTier == "" {
			out.SubscriptionTier = monthly.SubscriptionTier
		}
		if out.SubscriptionStatus == "" {
			out.SubscriptionStatus = monthly.SubscriptionStatus
		}
		out.Plan = monthly.Plan
		out.MonthlyUpdatedAt = now
	}

	out.Partial = !weeklyOK || !monthlyOK
	out.FailedWindows = nil
	if !weeklyOK {
		out.FailedWindows = append(out.FailedWindows, "weekly")
	}
	if !monthlyOK {
		out.FailedWindows = append(out.FailedWindows, "monthly")
	}
	if !weeklyOK && !monthlyOK && previous == nil {
		return nil
	}
	return &out
}

// StampBillingSummary sets fetch metadata.
func StampBillingSummary(summary *BillingSummary, statusCode int, source string) *BillingSummary {
	if summary == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	summary.StatusCode = statusCode
	summary.Source = source
	summary.FetchedAt = now
	summary.UpdatedAt = now
	return summary
}

func (s *BillingSummary) MergeSubscription(subscription *SubscriptionSummary) *BillingSummary {
	if s == nil || subscription == nil {
		return s
	}
	if subscription.Email != "" && s.Plan == "" {
		s.Plan = subscription.SubscriptionTier
	}
	if tier := strings.TrimSpace(subscription.SubscriptionTier); tier != "" {
		s.SubscriptionTier = tier
		s.Plan = tier
	}
	if subscription.Email != "" {
		s.Email = subscription.Email
	}
	if subscription.SubscriptionStatus != "" {
		s.SubscriptionStatus = subscription.SubscriptionStatus
	}
	if subscription.HasGrokCodeAccess {
		s.HasGrokCodeAccess = true
	}
	return s
}

func resolvePeriodType(period *BillingPeriod) string {
	if period == nil {
		return "unknown"
	}
	raw := strings.ToLower(strings.TrimSpace(period.Type))
	if strings.Contains(raw, "weekly") {
		return "weekly"
	}
	if strings.Contains(raw, "monthly") {
		return "monthly"
	}
	return "unknown"
}

func resolvePlan(monthlyLimitCents *float64) string {
	if monthlyLimitCents == nil {
		return ""
	}
	// Allow small float noise.
	limit := math.Round(*monthlyLimitCents)
	switch limit {
	case SuperGrokLimitCents:
		return "SuperGrok"
	case SuperGrokHeavyLimitCents:
		return "SuperGrok Heavy"
	default:
		return ""
	}
}

func parseCentValue(raw json.RawMessage) *float64 {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	// Object form: {"val": 123}
	var obj struct {
		Val any `json:"val"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil && obj.Val != nil {
		return anyToFloat(obj.Val)
	}
	// Bare number / string
	var n any
	if err := json.Unmarshal(raw, &n); err != nil {
		return nil
	}
	return anyToFloat(n)
}

func creditAmounts(values ...json.RawMessage) (*float64, *float64, *float64) {
	for _, raw := range values {
		if len(raw) == 0 || string(raw) == "null" {
			continue
		}
		var value any
		if err := json.Unmarshal(raw, &value); err != nil {
			continue
		}
		if used, total, remaining, ok := creditAmountsFromValue(value); ok {
			return used, total, remaining
		}
	}
	return nil, nil, nil
}

func creditAmountsFromRaw(usedRaw, totalRaw, remainingRaw json.RawMessage) (*float64, *float64, *float64) {
	used := parseCentValue(usedRaw)
	total := parseCentValue(totalRaw)
	remaining := parseCentValue(remainingRaw)
	if used == nil && total == nil && remaining == nil {
		return nil, nil, nil
	}
	if used == nil && total != nil && remaining != nil {
		v := math.Max(*total-*remaining, 0)
		used = &v
	}
	if remaining == nil && total != nil && used != nil {
		v := math.Max(*total-*used, 0)
		remaining = &v
	}
	return used, total, remaining
}

func creditAmountsFromValue(value any) (*float64, *float64, *float64, bool) {
	switch item := value.(type) {
	case []any:
		for _, child := range item {
			if used, total, remaining, ok := creditAmountsFromValue(child); ok {
				return used, total, remaining, true
			}
		}
	case map[string]any:
		total := firstNumericValue(item, "total", "limit", "cap", "allocation", "amount")
		used := firstNumericValue(item, "used", "spent", "consumed", "usage")
		remaining := firstNumericValue(item, "remaining", "balance", "left")
		if total != nil || used != nil || remaining != nil {
			if used == nil && total != nil && remaining != nil {
				v := math.Max(*total-*remaining, 0)
				used = &v
			}
			if remaining == nil && total != nil && used != nil {
				v := math.Max(*total-*used, 0)
				remaining = &v
			}
			return used, total, remaining, true
		}
		for _, key := range []string{"bags", "items"} {
			if child, ok := item[key]; ok {
				if used, total, remaining, ok := creditAmountsFromValue(child); ok {
					return used, total, remaining, true
				}
			}
		}
	}
	return nil, nil, nil, false
}

func firstNumericValue(object map[string]any, keys ...string) *float64 {
	for _, key := range keys {
		if raw, ok := object[key]; ok && raw != nil {
			if value := numericValue(raw); value != nil {
				return value
			}
		}
	}
	return nil
}

func numericValue(value any) *float64 {
	switch item := value.(type) {
	case float64:
		if !math.IsNaN(item) && !math.IsInf(item, 0) {
			return &item
		}
	case map[string]any:
		return firstNumericValue(item, "val", "value", "amount")
	case string:
		var parsed float64
		if err := json.Unmarshal([]byte(item), &parsed); err == nil &&
			!math.IsNaN(parsed) && !math.IsInf(parsed, 0) {
			return &parsed
		}
	}
	return nil
}

func anyToFloat(v any) *float64 {
	switch n := v.(type) {
	case float64:
		return &n
	case float32:
		f := float64(n)
		return &f
	case int:
		f := float64(n)
		return &f
	case int64:
		f := float64(n)
		return &f
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return nil
		}
		return &f
	case string:
		s := strings.TrimSpace(n)
		if s == "" {
			return nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil
		}
		return &f
	default:
		return nil
	}
}

func cloneFloat(v *float64) *float64 {
	if v == nil {
		return nil
	}
	f := *v
	return &f
}
