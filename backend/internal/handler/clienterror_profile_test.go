package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/clienterror"
	"github.com/stretchr/testify/require"
)

func useFreeClientErrorProfile(t *testing.T) {
	t.Helper()
	require.NoError(t, clienterror.Configure("free"))
	t.Cleanup(func() { require.NoError(t, clienterror.Configure("main")) })
}
