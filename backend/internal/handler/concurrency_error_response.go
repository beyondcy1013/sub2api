package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/clienterr"
)

const statusClientClosedRequest = 499

func concurrencyErrorResponse(err error, slotType string) (int, string, string) {
	var waitQueueFullErr *WaitQueueFullError
	if errors.As(err, &waitQueueFullErr) {
		return http.StatusTooManyRequests, "rate_limit_error",
			"Too many pending requests, please retry later"
	}

	var concurrencyErr *ConcurrencyError
	if errors.As(err, &concurrencyErr) {
		if concurrencyErr.SlotType != "" {
			slotType = concurrencyErr.SlotType
		}
		return http.StatusTooManyRequests, "rate_limit_error",
			clienterr.WithSource(fmt.Sprintf("Concurrency limit exceeded for %s, please retry later", slotType))
	}

	if errors.Is(err, context.Canceled) {
		return statusClientClosedRequest, "api_error", clienterr.WithSource("context canceled")
	}

	return http.StatusServiceUnavailable, "api_error",
		clienterr.WithSource("Service temporarily unavailable, please retry later")
}
