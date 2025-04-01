package serviceproxy

import (
	"fmt"
	"testing"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
)

func TestTokenCache(t *testing.T) {
	// Create a new cache with 1 second TTL for testing
	cache := newTokenCache(time.Second)

	// Test data
	testToken := "test-token"
	testUserInfo := &authenticationv1.UserInfo{
		Username: "test-user",
		Groups:   []string{"test-group"},
	}

	// Test Set and Get
	t.Run("Set and Get valid token", func(t *testing.T) {
		cache.Set(testToken, testUserInfo, 0)
		userInfo, exists := cache.Get(testToken)
		if !exists {
			t.Error("Expected token to exist in cache")
		}
		if userInfo.Username != testUserInfo.Username {
			t.Errorf("Expected username %s, got %s", testUserInfo.Username, userInfo.Username)
		}
		if len(userInfo.Groups) != len(testUserInfo.Groups) {
			t.Errorf("Expected %d groups, got %d", len(testUserInfo.Groups), len(userInfo.Groups))
		}
	})

	// Test expired token
	t.Run("Get expired token", func(t *testing.T) {
		// Wait for token to expire
		time.Sleep(time.Second * 2)
		_, exists := cache.Get(testToken)
		if exists {
			t.Error("Expected expired token to not exist in cache")
		}
	})

	// Test non-existent token
	t.Run("Get non-existent token", func(t *testing.T) {
		_, exists := cache.Get("non-existent-token")
		if exists {
			t.Error("Expected non-existent token to not exist in cache")
		}
	})

	// Test custom TTL
	t.Run("Set with custom TTL", func(t *testing.T) {
		customTTL := 2 * time.Second
		cache.Set(testToken, testUserInfo, customTTL)
		time.Sleep(time.Second)
		_, exists := cache.Get(testToken)
		if !exists {
			t.Error("Expected token to still exist in cache")
		}
		time.Sleep(time.Second * 2)
		_, exists = cache.Get(testToken)
		if exists {
			t.Error("Expected token to be expired")
		}
	})

	// Test concurrent access
	t.Run("Concurrent access", func(t *testing.T) {
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				cache.Set(testToken, testUserInfo, time.Second)
				_, _ = cache.Get(testToken)
				done <- true
			}()
		}
		for i := 0; i < 10; i++ {
			<-done
		}
		// No panic should occur
	})
}

func TestTokenCacheCleanup(t *testing.T) {
	// Create a new cache with very short TTL for testing cleanup
	cache := newTokenCache(100 * time.Millisecond)

	// Add multiple tokens
	for i := 0; i < 5; i++ {
		cache.Set(fmt.Sprintf("token-%d", i), &authenticationv1.UserInfo{
			Username: fmt.Sprintf("user-%d", i),
		}, 0)
	}

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	// Check if all tokens are cleaned up
	for i := 0; i < 5; i++ {
		_, exists := cache.Get(fmt.Sprintf("token-%d", i))
		if exists {
			t.Errorf("Token %d should have been cleaned up", i)
		}
	}
}
