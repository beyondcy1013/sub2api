#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/../backend"
go test -race ./internal/modules/autodetection
go test -tags unit ./internal/service ./internal/handler/admin ./internal/repository \
  -run 'Test(Automatic|ScheduledTestRunner|RateLimitService_RecoverAccountAfterSuccessfulTest|UpstreamBillingProbe|AccountRepositoryListDue)'
cd ../frontend
if command -v pnpm >/dev/null 2>&1 && [[ "$(pnpm --version)" == 9.* ]]; then
  detection_pnpm=(pnpm)
else
  detection_pnpm=(npm exec --yes --package=pnpm@9.15.9 -- pnpm)
fi
"${detection_pnpm[@]}" vitest run \
  src/modules/qualitycontrol/__tests__/AutomaticDetectionPolicies.spec.ts \
  src/modules/qualitycontrol/__tests__/QualityManagementDialog.spec.ts \
  src/components/account/__tests__/SchedulingRulesModal.spec.ts
