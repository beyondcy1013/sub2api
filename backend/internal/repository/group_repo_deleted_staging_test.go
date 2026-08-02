package repository

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGroupAccountCountsExcludeDeletedStagingRows(t *testing.T) {
	for _, predicate := range []string{
		groupAccountVisibleSQL,
		groupAccountAvailableSQL,
		groupAccountTemporarilyLimitedSQL,
	} {
		require.Contains(t, predicate, "extra -> 'deleted'")
	}
}
