package cli

import (
	"context"
	"sync"
	"time"
)

const cliSyncCacheTTL = 10 * time.Minute

var (
	cliSyncMu      sync.RWMutex
	cliSyncCached  *cliSyncResponse
	cliSyncCachedAt time.Time
)

func invalidateCLISyncCache() {
	cliSyncMu.Lock()
	cliSyncCached = nil
	cliSyncCachedAt = time.Time{}
	cliSyncMu.Unlock()
}

// getCLISyncCached returns sync v2 payload; refresh bypasses TTL cache.
func getCLISyncCached(ctx context.Context, refresh bool) (*cliSyncResponse, error) {
	if refresh {
		invalidateCLISyncCache()
	}
	cliSyncMu.RLock()
	if cliSyncCached != nil && time.Since(cliSyncCachedAt) < cliSyncCacheTTL {
		out := *cliSyncCached
		cliSyncMu.RUnlock()
		return &out, nil
	}
	cliSyncMu.RUnlock()

	resp, err := callCLISync(ctx)
	if err != nil {
		return nil, formatOpsfleetAPIError(err, "/api/cli/sync")
	}
	cliSyncMu.Lock()
	cliSyncCached = resp
	cliSyncCachedAt = time.Now()
	cliSyncMu.Unlock()
	return resp, nil
}
