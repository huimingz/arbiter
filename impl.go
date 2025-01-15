package redission

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/huimingz/redission/internal/lua"
)

var (
	ErrLockNotHeld = errors.New("lock not held")
	ErrLockTimeout = errors.New("lock timeout")
)

type lockImpl struct {
	redis   *redis.Client
	name    string
	value   string
	options *LockOptions

	watchDogCtx    context.Context
	watchDogCancel context.CancelFunc
	watchDogOnce   sync.Once
	watchDogDone   chan struct{}
	
	mu sync.Mutex
}

func newLock(redis *redis.Client, name string, options *LockOptions) Lock {
	return &lockImpl{
		redis:   redis,
		name:    name,
		value:   generateValue(),
		options: options,
		watchDogDone: make(chan struct{}),
	}
}

func (l *lockImpl) Lock(ctx context.Context) error {
	deadline := time.Now().Add(l.options.WaitTimeout)
	for {
		acquired, err := l.TryLock(ctx)
		if err != nil {
			return err
		}
		if acquired {
			return nil
		}

		if l.options.WaitTimeout > 0 && time.Now().After(deadline) {
			return ErrLockTimeout
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond): // retry delay
			continue
		}
	}
}

func (l *lockImpl) TryLock(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Try to acquire the lock
	leaseTime := l.options.LeaseTime
	if l.options.EnableWatchDog {
		leaseTime = l.options.WatchDogTimeout
	}

	ok, err := l.redis.Eval(ctx, lua.TryLock, []string{l.name}, l.value, leaseTime.Milliseconds()).Bool()
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	// Start watchdog if enabled
	if l.options.EnableWatchDog {
		l.startWatchDog(ctx)
	}

	return true, nil
}

func (l *lockImpl) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Stop watchdog if it's running
	if l.watchDogCancel != nil {
		l.watchDogCancel()
		<-l.watchDogDone
	}

	// Release the lock
	ok, err := l.redis.Eval(ctx, lua.Unlock, []string{l.name}, l.value).Bool()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockNotHeld
	}

	return nil
}

func (l *lockImpl) Refresh(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	leaseTime := l.options.LeaseTime
	if l.options.EnableWatchDog {
		leaseTime = l.options.WatchDogTimeout
	}

	ok, err := l.redis.Eval(ctx, lua.Refresh, []string{l.name}, l.value, leaseTime.Milliseconds()).Bool()
	if err != nil {
		return err
	}
	if !ok {
		return ErrLockNotHeld
	}

	return nil
}

func (l *lockImpl) startWatchDog(ctx context.Context) {
	l.watchDogOnce.Do(func() {
		l.watchDogCtx, l.watchDogCancel = context.WithCancel(context.Background())
		
		go func() {
			defer close(l.watchDogDone)
			
			ticker := time.NewTicker(l.options.WatchDogTimeout / 3)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := l.Refresh(ctx); err != nil {
						// If refresh fails, the lock might be lost
						return
					}
				case <-l.watchDogCtx.Done():
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	})
}
