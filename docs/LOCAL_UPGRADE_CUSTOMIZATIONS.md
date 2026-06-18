# Local Upgrade Customizations

This repository carries local `sub2freeApi` behavior that must be preserved when downloading or merging upgraded upstream source.

## Preserve These Behaviors

- Admin account management shows account identifiers in plaintext.
- Admin account management shows upstream `credentials.api_key` fully in plaintext.
- Backend admin account DTOs do not redact `api_key`, but still redact OAuth tokens, cookies, private keys, refresh tokens, and similar secrets.
- Account updates preserve an existing `api_key` when the update payload omits it.
- Client-facing errors include a source prefix:
  - `【sub2freeApi限制】` for local/auth/quota/concurrency/config limits.
  - `【上游错误】` for upstream-originated failures.
- `/responses` streaming errors emit protocol-compatible `response.failed` events with prefixed messages.
- The free service deploys only as `sub2freeApi.service` on port `18382`.

## Files To Check After Upstream Refresh

Account/API Key plaintext:

- `backend/internal/service/account_credentials_redact.go`
- `backend/internal/handler/dto/credentials_redact.go`
- `backend/internal/handler/dto/account_mapper_redact_test.go`
- `frontend/src/components/account/EditAccountModal.vue`
- `frontend/src/views/admin/AccountsView.vue`
- `frontend/src/types/index.ts`
- `frontend/src/i18n/locales/en.ts`
- `frontend/src/i18n/locales/zh.ts`

Error prefixes:

- `backend/internal/pkg/clienterror/`
- `backend/internal/handler/openai_gateway_handler.go`
- `backend/internal/handler/gateway_handler.go`
- `backend/internal/handler/gateway_handler_responses.go`
- `backend/internal/handler/gateway_handler_chat_completions.go`
- `backend/internal/handler/gemini_v1beta_handler.go`
- `backend/internal/handler/stream_error_event.go`
- `backend/internal/server/middleware/middleware.go`
- `backend/internal/server/middleware/api_key_auth_google.go`
- Direct service error writers under `backend/internal/service/`.

## Useful Searches

```bash
rg -n "SensitiveCredentialKeys|PreserveOnMissingCredentialKeys|api_key|apiKeyPlainVisibleHint|type=\"password\"|editApiKey" backend/internal frontend/src
rg -n "clienterror|sub2freeApi限制|上游错误|writeResponsesFailedSSE|AbortWithError|abortWithGoogleError|closeOpenAIClientWS" backend/internal
```

## Focused Verification

From `backend/`:

```bash
go test ./internal/pkg/clienterror ./internal/server/middleware ./internal/handler ./internal/handler/dto ./internal/service -run 'TestPrefix|TestApiKeyAuth|TestAPIKeyAuth|TestOpenAIHandleStreamingAwareError|TestGatewayHandleStreamingAwareError|TestOpenAIEnsureForwardErrorResponse|TestGatewayEnsureForwardErrorResponse|TestOpenAIRecoverResponsesPanic|TestMapResponsesErrorCode|TestConcurrencyErrorResponse|TestRedactCredentials|TestAccountFromServiceShallow|TestMergePreservingSensitiveCreds|TestIsSensitiveCredentialKey|Test.*ErrorPassthrough|TestGatewayHandleErrorResponse|TestOpenAIHandleErrorResponse|TestGeminiWriteGeminiMappedError'
```

From `frontend/`:

```bash
pnpm vitest run src/components/account/__tests__/EditAccountModal.spec.ts src/views/admin/__tests__/AccountsView.bulkEdit.spec.ts
pnpm typecheck
pnpm build
```

## Deploy

```bash
cd /home/third_party/sub2freeApi/backend
CGO_ENABLED=0 go build -tags embed -o /home/third_party/bin/sub2freeApi/sub2freeApi ./cmd/server/
systemctl restart sub2freeApi.service
systemctl status sub2freeApi.service --no-pager
ss -ltnp | rg ':18382'
curl -fsS http://127.0.0.1:18382/ >/dev/null
curl -sS -X POST http://127.0.0.1:18382/v1/responses -H 'Content-Type: application/json' -d '{"model":"gpt-5","input":"hi"}'
```

The missing-auth response should contain `【sub2freeApi限制】`.
