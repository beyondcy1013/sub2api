package clienterror

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClientErrorProfileMain(t *testing.T) {
	require.NoError(t, Configure("main"))
	t.Cleanup(func() { require.NoError(t, Configure("main")) })

	require.Equal(t, "sub2api", Source)
	require.Equal(t, "local failure", Local("local failure"))
	require.Equal(t, "upstream failure", Upstream("upstream failure"))
	require.Equal(t, "failure", WithSource("failure"))
}

func TestClientErrorProfileFree(t *testing.T) {
	require.NoError(t, Configure("free"))
	t.Cleanup(func() { require.NoError(t, Configure("main")) })

	require.Equal(t, "sub2freeApi", Source)
	require.Equal(t, "【sub2freeApi限制】 local failure", Local("local failure"))
	require.Equal(t, "【上游错误】 upstream failure", Upstream("upstream failure"))
	require.Equal(t, "failure (source: sub2freeApi)", WithSource("failure"))
}

func TestClientErrorProfileRejectsUnknownValue(t *testing.T) {
	require.NoError(t, Configure("main"))
	require.Error(t, Configure("enterprise"))
	require.Equal(t, "sub2api", Source)
}
