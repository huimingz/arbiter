package arbiter

import "time"

// LockOptions defines the options for lock configuration
type LockOptions struct {
	// WaitTimeout specifies how long to wait for lock acquisition
	WaitTimeout time.Duration

	// LeaseTime specifies the lock expiration time
	LeaseTime time.Duration

	// EnableWatchDog enables automatic lock renewal
	EnableWatchDog bool

	// WatchDogTimeout specifies the watchdog timeout (only valid when EnableWatchDog is true)
	WatchDogTimeout time.Duration
}

// Option is a function type for setting lock options
type Option func(*LockOptions)

// WithWaitTimeout sets the wait timeout
func WithWaitTimeout(timeout time.Duration) Option {
	return func(o *LockOptions) {
		o.WaitTimeout = timeout
	}
}

// WithLeaseTime sets the lease time
func WithLeaseTime(leaseTime time.Duration) Option {
	return func(o *LockOptions) {
		o.LeaseTime = leaseTime
	}
}

// WithWatchDog enables or disables the watchdog
func WithWatchDog(enable bool) Option {
	return func(o *LockOptions) {
		o.EnableWatchDog = enable
	}
}

// WithWatchDogTimeout sets the watchdog timeout
func WithWatchDogTimeout(timeout time.Duration) Option {
	return func(o *LockOptions) {
		o.WatchDogTimeout = timeout
	}
}

// defaultOptions returns the default lock options
func defaultOptions() *LockOptions {
	return &LockOptions{
		WaitTimeout:     0,                // no wait timeout by default
		LeaseTime:       30 * time.Second, // 30 seconds lease time by default
		EnableWatchDog:  false,            // watchdog disabled by default
		WatchDogTimeout: 30 * time.Second, // 30 seconds watchdog timeout by default
	}
}
