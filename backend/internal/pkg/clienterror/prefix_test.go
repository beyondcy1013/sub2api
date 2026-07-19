package clienterror

import "testing"

func TestPrefix(t *testing.T) {
	if err := Configure("free"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := Configure("main"); err != nil {
			t.Fatal(err)
		}
	})
	tests := []struct {
		name    string
		errType string
		message string
		want    string
	}{
		{
			name:    "local rate limit",
			errType: "rate_limit_error",
			message: "Concurrency limit exceeded for account, please retry later",
			want:    "【sub2freeApi限制】 Concurrency limit exceeded for account, please retry later",
		},
		{
			name:    "upstream type",
			errType: "upstream_error",
			message: "Concurrency limit exceeded for account, please retry later",
			want:    "【上游错误】 Concurrency limit exceeded for account, please retry later",
		},
		{
			name:    "upstream message",
			errType: "rate_limit_error",
			message: "Upstream rate limit exceeded, please retry later",
			want:    "【上游错误】 Upstream rate limit exceeded, please retry later",
		},
		{
			name:    "idempotent local",
			errType: "rate_limit_error",
			message: "【sub2freeApi限制】 Too many pending requests, please retry later",
			want:    "【sub2freeApi限制】 Too many pending requests, please retry later",
		},
		{
			name:    "replaces legacy local prefix",
			errType: "rate_limit_error",
			message: "【" + "本" + "项目限制】 Too many pending requests, please retry later",
			want:    "【sub2freeApi限制】 Too many pending requests, please retry later",
		},
		{
			name:    "idempotent upstream",
			errType: "upstream_error",
			message: "【上游错误】 Upstream request failed",
			want:    "【上游错误】 Upstream request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Prefix(tt.errType, tt.message); got != tt.want {
				t.Fatalf("Prefix() = %q, want %q", got, tt.want)
			}
		})
	}
}
