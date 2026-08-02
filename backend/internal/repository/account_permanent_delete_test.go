//go:build unit

package repository

import (
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAccountCanBePermanentlyDeleted(t *testing.T) {
	deletedAt := time.Now()

	require.False(t, accountCanBePermanentlyDeleted(nil))
	require.False(t, accountCanBePermanentlyDeleted(&dbent.Account{}))
	require.False(t, accountCanBePermanentlyDeleted(&dbent.Account{
		Extra: map[string]any{"recycled": true},
	}))
	require.True(t, accountCanBePermanentlyDeleted(&dbent.Account{
		Extra: map[string]any{service.AccountDeletedStagingExtraKey: true},
	}))
	require.True(t, accountCanBePermanentlyDeleted(&dbent.Account{
		DeletedAt: &deletedAt,
	}))
}
