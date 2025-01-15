package redission

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func setupRedis(t *testing.T) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis is not available: %v", err)
	}

	return client
}

func TestLock(t *testing.T) {
	redisClient := setupRedis(t)
	defer redisClient.Close()

	client := NewClient(redisClient)
	ctx := context.Background()

	t.Run("basic lock and unlock", func(t *testing.T) {
		lock := client.NewLock("test-lock")
		
		// Should be able to acquire lock
		err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		// Should be able to unlock
		err = lock.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})

	t.Run("try lock", func(t *testing.T) {
		lock1 := client.NewLock("test-trylock")
		lock2 := client.NewLock("test-trylock")

		// First lock should succeed
		acquired, err := lock1.TryLock(ctx)
		if err != nil {
			t.Fatalf("Failed to try lock: %v", err)
		}
		if !acquired {
			t.Fatal("Should acquire first lock")
		}

		// Second lock should fail
		acquired, err = lock2.TryLock(ctx)
		if err != nil {
			t.Fatalf("Failed to try lock: %v", err)
		}
		if acquired {
			t.Fatal("Should not acquire second lock")
		}

		// Cleanup
		err = lock1.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})

	t.Run("lock with timeout", func(t *testing.T) {
		lock1 := client.NewLock("test-timeout")
		lock2 := client.NewLock("test-timeout",
			WithWaitTimeout(2*time.Second),
		)

		// Acquire first lock
		err := lock1.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire first lock: %v", err)
		}

		// Second lock should timeout
		err = lock2.Lock(ctx)
		if err != ErrLockTimeout {
			t.Fatalf("Expected timeout error, got: %v", err)
		}

		// Cleanup
		err = lock1.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})

	t.Run("watchdog auto refresh", func(t *testing.T) {
		lock := client.NewLock("test-watchdog",
			WithLeaseTime(2*time.Second),
			WithWatchDog(true),
			WithWatchDogTimeout(1*time.Second),
		)

		// Acquire lock
		err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		// Wait for longer than lease time
		time.Sleep(3 * time.Second)

		// Lock should still be valid
		err = lock.Refresh(ctx)
		if err != nil {
			t.Fatalf("Lock should still be valid: %v", err)
		}

		// Cleanup
		err = lock.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})

	t.Run("manual refresh", func(t *testing.T) {
		lock := client.NewLock("test-refresh",
			WithLeaseTime(2*time.Second),
		)

		// Acquire lock
		err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		// Manual refresh
		err = lock.Refresh(ctx)
		if err != nil {
			t.Fatalf("Failed to refresh lock: %v", err)
		}

		// Cleanup
		err = lock.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})
}
