package serviceproxy

import (
	"context"
	"sync"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
)

type tokenCacheEntry struct {
	userInfo *authenticationv1.UserInfo
	expiry   time.Time
}

type tokenCache struct {
	cache map[string]tokenCacheEntry
	mu    sync.RWMutex
	// Default cache TTL, can be adjusted based on actual token validity period
	defaultTTL time.Duration
}

func newTokenCache(defaultTTL time.Duration) *tokenCache {
	if defaultTTL == 0 {
		defaultTTL = 10 * time.Minute // Default cache duration is 5 minutes
	}

	cache := &tokenCache{
		cache:      make(map[string]tokenCacheEntry),
		defaultTTL: defaultTTL,
	}

	// Start a goroutine to periodically clean up expired cache entries
	go cache.periodicCleanup(context.Background())

	return cache
}

func (tc *tokenCache) Get(token string) (*authenticationv1.UserInfo, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	entry, exists := tc.cache[token]
	if !exists {
		return nil, false
	}

	// Check if the entry has expired
	if time.Now().After(entry.expiry) {
		return nil, false
	}

	return entry.userInfo, true
}

func (tc *tokenCache) Set(token string, userInfo *authenticationv1.UserInfo, ttl time.Duration) {
	if ttl == 0 {
		ttl = tc.defaultTTL
	}

	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache[token] = tokenCacheEntry{
		userInfo: userInfo,
		expiry:   time.Now().Add(ttl),
	}
}

func (tc *tokenCache) periodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(tc.defaultTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tc.cleanup()
		}
	}
}

func (tc *tokenCache) cleanup() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	now := time.Now()
	for token, entry := range tc.cache {
		if now.After(entry.expiry) {
			delete(tc.cache, token)
		}
	}
}
