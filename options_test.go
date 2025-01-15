package redission

import (
	"testing"
	"time"
)

func TestLockOptions(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		expected LockOptions
	}{
		{
			name: "default options",
			opts: []Option{},
			expected: LockOptions{
				WaitTimeout:     0,
				LeaseTime:      30 * time.Second,
				EnableWatchDog: false,
				WatchDogTimeout: 30 * time.Second,
			},
		},
		{
			name: "custom wait timeout",
			opts: []Option{
				WithWaitTimeout(5 * time.Second),
			},
			expected: LockOptions{
				WaitTimeout:     5 * time.Second,
				LeaseTime:      30 * time.Second,
				EnableWatchDog: false,
				WatchDogTimeout: 30 * time.Second,
			},
		},
		{
			name: "custom lease time",
			opts: []Option{
				WithLeaseTime(10 * time.Second),
			},
			expected: LockOptions{
				WaitTimeout:     0,
				LeaseTime:      10 * time.Second,
				EnableWatchDog: false,
				WatchDogTimeout: 30 * time.Second,
			},
		},
		{
			name: "enable watchdog",
			opts: []Option{
				WithWatchDog(true),
				WithWatchDogTimeout(20 * time.Second),
			},
			expected: LockOptions{
				WaitTimeout:     0,
				LeaseTime:      30 * time.Second,
				EnableWatchDog: true,
				WatchDogTimeout: 20 * time.Second,
			},
		},
		{
			name: "multiple options",
			opts: []Option{
				WithWaitTimeout(5 * time.Second),
				WithLeaseTime(10 * time.Second),
				WithWatchDog(true),
				WithWatchDogTimeout(20 * time.Second),
			},
			expected: LockOptions{
				WaitTimeout:     5 * time.Second,
				LeaseTime:      10 * time.Second,
				EnableWatchDog: true,
				WatchDogTimeout: 20 * time.Second,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := defaultOptions()
			for _, opt := range tt.opts {
				opt(options)
			}

			if options.WaitTimeout != tt.expected.WaitTimeout {
				t.Errorf("WaitTimeout = %v, want %v", options.WaitTimeout, tt.expected.WaitTimeout)
			}
			if options.LeaseTime != tt.expected.LeaseTime {
				t.Errorf("LeaseTime = %v, want %v", options.LeaseTime, tt.expected.LeaseTime)
			}
			if options.EnableWatchDog != tt.expected.EnableWatchDog {
				t.Errorf("EnableWatchDog = %v, want %v", options.EnableWatchDog, tt.expected.EnableWatchDog)
			}
			if options.WatchDogTimeout != tt.expected.WatchDogTimeout {
				t.Errorf("WatchDogTimeout = %v, want %v", options.WatchDogTimeout, tt.expected.WatchDogTimeout)
			}
		})
	}
}
