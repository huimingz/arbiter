package redission

import "context"

// Lock represents a distributed lock interface
type Lock interface {
	// Lock acquires the lock, blocking until it succeeds or ctx is done
	// It has a Watch Dog mechanism that automatically extends the lock every 30s
	// until unlock or ctx is done.
	Lock(ctx context.Context) error

	// TryLock attempts to acquire the lock and returns immediately
	// It has a Watch Dog mechanism that automatically extends the lock every 30s
	// until unlock or ctx is done.
	TryLock(ctx context.Context) (bool, error)

	// Unlock releases the lock
	Unlock(ctx context.Context) error

	// Refresh manually extends the lock's lease time
	Refresh(ctx context.Context) error
}
