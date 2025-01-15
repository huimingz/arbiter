package arbiter

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/huimingz/arbiter/internal/lua"
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
	logger  Logger

	watchDogCtx    context.Context
	watchDogCancel context.CancelFunc
	watchDogOnce   sync.Once
	watchDogDone   chan struct{}
	
	mu sync.Mutex
}

func newLock(redis *redis.Client, name string, options *LockOptions, logger Logger) Lock {
	return &lockImpl{
		redis:   redis,
		name:    name,
		value:   generateValue(),
		options: options,
		logger:  logger,
		watchDogDone: make(chan struct{}),
	}
}

func (l *lockImpl) Lock(ctx context.Context) error {
	deadline := time.Now().Add(l.options.WaitTimeout)
	l.logger.Debug(ctx, "Attempting to acquire lock: %s", l.name)
	
	attempt := 0
	for {
		attempt++
		acquired, err := l.TryLock(ctx)
		if err != nil {
			l.logger.Error(ctx, "Failed to acquire lock: %s, error: %v", l.name, err)
			return err
		}
		if acquired {
			l.logger.Info(ctx, "Successfully acquired lock: %s", l.name)
			return nil
		}

		if l.options.WaitTimeout > 0 && time.Now().After(deadline) {
			l.logger.Warn(ctx, "Timeout waiting for lock: %s", l.name)
			return ErrLockTimeout
		}

		select {
		case <-ctx.Done():
			l.logger.Debug(ctx, "Context cancelled while waiting for lock: %s", l.name)
			return ctx.Err()
		case <-time.After(100 * time.Millisecond): // retry delay
			continue
		}
	}
}

func (l *lockImpl) TryLock(ctx context.Context) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	leaseTime := l.options.LeaseTime
	if l.options.EnableWatchDog {
		leaseTime = l.options.WatchDogTimeout
	}

	ok, err := l.redis.Eval(ctx, lua.TryLock, []string{l.name}, l.value, leaseTime.Milliseconds()).Bool()
	if err != nil {
		l.logger.Error(ctx, "Error trying to acquire lock: %s", l.name)
		return false, err
	}
	if !ok {
		return false, nil
	}

	if l.options.EnableWatchDog {
		l.logger.Debug(ctx, "Starting watchdog for lock: %s", l.name)
		l.startWatchDog(ctx)
	}

	return true, nil
}

func (l *lockImpl) Unlock(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.logger.Debug(ctx, "Releasing lock: %s", l.name)

	if l.watchDogCancel != nil {
		l.watchDogCancel()
		<-l.watchDogDone
	}

	ok, err := l.redis.Eval(ctx, lua.Unlock, []string{l.name}, l.value).Bool()
	if err != nil {
		l.logger.Error(ctx, "Error releasing lock: %s", l.name)
		return err
	}
	if !ok {
		return ErrLockNotHeld
	}

	l.logger.Info(ctx, "Released lock: %s", l.name)
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
		l.logger.Error(ctx, "Error refreshing lock: %s", l.name)
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
						l.logger.Error(ctx, "Watchdog failed to refresh lock: %s", l.name)
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
