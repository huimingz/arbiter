package arbiter

import (
	"context"
	stderrors "errors"
	"sync"
	"sync/atomic"
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

		err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		err = lock.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})

	t.Run("try lock", func(t *testing.T) {
		lock1 := client.NewLock("test-trylock")
		lock2 := client.NewLock("test-trylock")

		acquired, err := lock1.TryLock(ctx)
		if err != nil {
			t.Fatalf("Failed to try lock: %v", err)
		}
		if !acquired {
			t.Fatal("Should acquire first lock")
		}

		acquired, err = lock2.TryLock(ctx)
		if err != nil {
			t.Fatalf("Failed to try lock: %v", err)
		}
		if acquired {
			t.Fatal("Should not acquire second lock")
		}

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

		err := lock1.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire first lock: %v", err)
		}

		err = lock2.Lock(ctx)
		if err != ErrLockTimeout {
			t.Fatalf("Expected timeout error, got: %v", err)
		}

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

		err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		time.Sleep(3 * time.Second)

		err = lock.Refresh(ctx)
		if err != nil {
			t.Fatalf("Lock should still be valid: %v", err)
		}

		err = lock.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})

	t.Run("manual refresh", func(t *testing.T) {
		lock := client.NewLock("test-refresh",
			WithLeaseTime(2*time.Second),
		)

		err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		err = lock.Refresh(ctx)
		if err != nil {
			t.Fatalf("Failed to refresh lock: %v", err)
		}

		err = lock.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})
}

func TestConcurrentLock(t *testing.T) {
	redisClient := setupRedis(t)
	defer redisClient.Close()

	client := NewClient(redisClient)
	ctx := context.Background()

	t.Run("concurrent lock acquisition", func(t *testing.T) {
		const (
			numGoroutines = 10
			numIterations = 20
		)

		var (
			wg           sync.WaitGroup
			successCount atomic.Int32
			lockHolder   atomic.Int32
			errors       = make(chan error, numGoroutines*numIterations)
		)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				lock := client.NewLock("test-concurrent",
					WithWaitTimeout(5*time.Second),
					WithLeaseTime(1*time.Second),
				)

				for j := 0; j < numIterations; j++ {
					err := lock.Lock(ctx)
					if err != nil {
						if err != ErrLockTimeout {
							errors <- err
						}
						continue
					}

					// Ensure that only one goroutine holds the lock at any given time.
					// If the lockHolder is not zero, it means that some other goroutine
					// is holding the lock, which is unexpected.
					prev := lockHolder.Add(1)
					if prev > 1 {
						errors <- stderrors.New("multiple lock holders detected")
					}

					time.Sleep(100 * time.Millisecond)

					lockHolder.Add(-1)
					successCount.Add(1)

					if err := lock.Unlock(ctx); err != nil {
						errors <- err
					}
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		for err := range errors {
			t.Errorf("Concurrent lock error: %v", err)
		}

		if successCount.Load() == 0 {
			t.Error("No goroutine succeeded in acquiring the lock")
		}
		t.Logf("Successfully acquired lock %d times", successCount.Load())
	})

	t.Run("concurrent lock and refresh", func(t *testing.T) {
		const numGoroutines = 5
		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines)

		lock1 := client.NewLock("test-concurrent-refresh",
			WithLeaseTime(2*time.Second),
		)

		if err := lock1.Lock(ctx); err != nil {
			t.Fatalf("Failed to acquire initial lock: %v", err)
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				lock2 := client.NewLock("test-concurrent-refresh",
					WithWaitTimeout(1*time.Second),
				)

				if acquired, err := lock2.TryLock(ctx); err != nil {
					errors <- err
				} else if acquired {
					errors <- stderrors.New("lock should not be acquired")
				}
			}(i)
		}

		go func() {
			for i := 0; i < 3; i++ {
				time.Sleep(1 * time.Second)
				if err := lock1.Refresh(ctx); err != nil {
					errors <- err
					return
				}
			}
		}()

		wg.Wait()

		if err := lock1.Unlock(ctx); err != nil {
			t.Errorf("Failed to release lock: %v", err)
		}

		close(errors)
		for err := range errors {
			t.Errorf("Concurrent refresh error: %v", err)
		}
	})
}

func TestCustomKeyPrefix(t *testing.T) {
	redisClient := setupRedis(t)
	defer redisClient.Close()

	customPrefix := "test-prefix:"
	client := NewClient(redisClient, WithKeyPrefix(customPrefix))
	ctx := context.Background()

	t.Run("verify key prefix", func(t *testing.T) {
		lockName := "test-lock"
		lock := client.NewLock(lockName)

		err := lock.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire lock: %v", err)
		}

		// Verify the key exists with custom prefix
		exists, err := redisClient.Exists(ctx, customPrefix+lockName).Result()
		if err != nil {
			t.Fatalf("Failed to check key existence: %v", err)
		}
		if exists != 1 {
			t.Errorf("Expected key %s to exist", customPrefix+lockName)
		}

		// Clean up
		err = lock.Unlock(ctx)
		if err != nil {
			t.Fatalf("Failed to release lock: %v", err)
		}
	})

	t.Run("different prefixes don't conflict", func(t *testing.T) {
		client1 := NewClient(redisClient, WithKeyPrefix("prefix1:"))
		client2 := NewClient(redisClient, WithKeyPrefix("prefix2:"))
		
		lockName := "same-lock"
		lock1 := client1.NewLock(lockName)
		lock2 := client2.NewLock(lockName)

		// Acquire first lock
		err := lock1.Lock(ctx)
		if err != nil {
			t.Fatalf("Failed to acquire first lock: %v", err)
		}

		// Try to acquire second lock with different prefix
		acquired, err := lock2.TryLock(ctx)
		if err != nil {
			t.Fatalf("Failed to try second lock: %v", err)
		}
		if !acquired {
			t.Error("Second lock should be acquired as it uses different prefix")
		}

		// Clean up
		if err := lock1.Unlock(ctx); err != nil {
			t.Fatalf("Failed to release first lock: %v", err)
		}
		if err := lock2.Unlock(ctx); err != nil {
			t.Fatalf("Failed to release second lock: %v", err)
		}
	})
}
