package service

import (
	"net/http"
	"strings"
)

var upstreamModelNotFoundKeywords = []string{"model not found", "unknown model", "not found"}

func isUpstreamModelNotFoundError(statusCode int, body []byte) bool {
	if statusCode != http.StatusNotFound {
		return false
	}
	normalized := normalizeModelNotFoundBody(body)
	if normalized == "" || !strings.Contains(normalized, "model") {
		return false
	}
	return containsModelNotFoundKeyword(normalized)
}

func isModelNotFoundError(statusCode int, body []byte) bool {
	return isUpstreamModelNotFoundError(statusCode, body) || statusCode == http.StatusNotFound
}

// openAICodexPlanGatedModelPhrase matches the deterministic Codex 400 returned
// when a ChatGPT OAuth account's plan cannot serve the requested model, e.g.
// {"detail":"The 'gpt-5.6-sol' model is not supported when using Codex with a ChatGPT account."}
// The phrase is compared against the normalized body (lowercased, "_"/"-"
// folded to spaces), so it also matches the same message embedded in
// error.message-style payloads.
const openAICodexPlanGatedModelPhrase = "model is not supported when using codex"

// isOpenAICodexPlanGatedModelError reports whether the upstream response is the
// deterministic Codex rejection of a plan-gated model on a ChatGPT account.
// Unlike transient failures, retrying the same account cannot succeed until the
// account's plan changes, so callers should treat it like model-not-found and
// cool the (account, model) pair down instead of re-selecting the account.
func isOpenAICodexPlanGatedModelError(statusCode int, body []byte) bool {
	if statusCode != http.StatusBadRequest {
		return false
	}
	normalized := normalizeModelNotFoundBody(body)
	if normalized == "" {
		return false
	}
	return strings.Contains(normalized, openAICodexPlanGatedModelPhrase)
}

// openAIModelNotAllowed403Phrase matches upstream 403 responses that reject a
// model the account's upstream group cannot serve, e.g. NewAPI-style
// {"code":"model_not_allowed","message":"当前分组不支持模型「claude-fable-5」。支持的模型：..."}
// or generic English "model is not allowed". Unlike an account health 403,
// retrying the same account cannot succeed for that model, but the account is
// fully healthy for every other supported model. Compared against the
// normalized body (lowercased, "_"/"-" folded to spaces), matching the
// deterministic phrases used across upstream vendors.
const openAIModelNotAllowed403Phrase = "model is not allowed"
const openAIModelNotAllowedGroupZhPhrase = "不支持模型"

// isOpenAIModelNotAllowed403Error reports whether a 403 is a model-routing
// rejection rather than an account health problem. Such 403s must NOT freeze
// the whole account; instead the (account, model) pair is cooled down via
// SetModelRateLimit, mirroring 404 model-not-found handling.
func isOpenAIModelNotAllowed403Error(statusCode int, body []byte) bool {
	if statusCode != http.StatusForbidden {
		return false
	}
	normalized := normalizeModelNotFoundBody(body)
	if normalized == "" {
		return false
	}
	// Vendor-specific Chinese "group does not support model" rejection.
	if strings.Contains(string(body), openAIModelNotAllowedGroupZhPhrase) {
		return true
	}
	// Structured code field, e.g. NewAPI {"code":"model_not_allowed"}.
	if strings.Contains(normalized, "code model not allowed") {
		return true
	}
	// English phrases: contiguous "model is not allowed" or a body that
	// mentions both "model" and "not allowed" (the model name may sit between
	// them, e.g. "The model claude-fable-5 is not allowed").
	if strings.Contains(normalized, openAIModelNotAllowed403Phrase) || strings.Contains(normalized, "model not allowed") {
		return true
	}
	return strings.Contains(normalized, "model") && strings.Contains(normalized, "not allowed")
}

func containsModelNotFoundKeyword(normalizedBody string) bool {
	if normalizedBody == "" {
		return false
	}
	for _, keyword := range upstreamModelNotFoundKeywords {
		if strings.Contains(normalizedBody, keyword) {
			return true
		}
	}
	return false
}

func normalizeModelNotFoundBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	normalized := strings.ToLower(string(body))
	normalized = strings.NewReplacer("_", " ", "-", " ", "\n", " ", "\r", " ", "\t", " ").Replace(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}
