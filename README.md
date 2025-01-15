# Redission

A Redis-based distributed lock implementation in Go, inspired by Redisson.

## Features

- Distributed Lock with Redis
  - Lock acquisition with timeout
  - Automatic lock renewal (watchdog)
  - Lock reentrant support
  - Safe lock release
  - Configurable logging

## Installation

```bash
go get github.com/huimingz/redission
```

## Quick Start

```go
import (
    "context"
    "time"
    "github.com/redis/go-redis/v9"
    "github.com/huimingz/redission"
)

func main() {
    // Create Redis client
    redisClient := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })
    defer redisClient.Close()

    // Create Redission client
    client := redission.NewClient(redisClient)

    // Create a lock with options
    lock := client.NewLock("my-lock",
        redission.WithWaitTimeout(5*time.Second),  // Wait up to 5s to acquire lock
        redission.WithLeaseTime(30*time.Second),   // Lock expires after 30s
        redission.WithWatchDog(true),              // Auto-renew lock
    )

    // Try to acquire the lock
    ctx := context.Background()
    if err := lock.Lock(ctx); err != nil {
        // Handle lock acquisition failure
        return
    }

    // Don't forget to unlock
    defer lock.Unlock(ctx)

    // Your critical section here
    // ...
}
```

## Lock Options

- `WithWaitTimeout(d time.Duration)`: Maximum time to wait for lock acquisition
- `WithLeaseTime(d time.Duration)`: Lock lease time (expiration)
- `WithWatchDog(enable bool)`: Enable automatic lock renewal
- `WithWatchDogTimeout(d time.Duration)`: Interval for watchdog renewal

## Logging

Redission supports customizable logging through a simple interface:

```go
type Logger interface {
    Debug(ctx context.Context, format string, args ...interface{})
    Info(ctx context.Context, format string, args ...interface{})
    Warn(ctx context.Context, format string, args ...interface{})
    Error(ctx context.Context, format string, args ...interface{})
}
```

You can provide your own logger implementation:

```go
client := redission.NewClient(redisClient, 
    redission.WithLogger(myLogger),
)
```

## Implementation Details

### Lock Mechanism

The distributed lock is implemented using Redis hash structures and Lua scripts to ensure atomicity. The key features are:

1. **Atomic Lock Acquisition**
   - Uses Lua script to check and set lock atomically
   - Supports lock reentrance (same client can acquire lock multiple times)
   - Sets lock expiration to prevent deadlocks

2. **Safe Lock Release**
   - Only the lock owner can release the lock
   - Uses Lua script to verify ownership before deletion

3. **Automatic Lock Renewal**
   - Optional watchdog mechanism to prevent lock expiration
   - Periodically refreshes lock lease time
   - Stops renewal when lock is released or context is cancelled

### Redis Key Structure

- Key: `<lock-name>`
- Type: Hash
- Fields:
  - `owner`: Client identifier
  - TTL: Set using `PEXPIRE`

## Best Practices

1. **Always Use Timeouts**
   ```go
   lock := client.NewLock("my-lock",
       redission.WithWaitTimeout(5*time.Second),
       redission.WithLeaseTime(30*time.Second),
   )
   ```

2. **Use Watchdog for Long Operations**
   ```go
   lock := client.NewLock("my-lock",
       redission.WithWatchDog(true),
       redission.WithWatchDogTimeout(10*time.Second),
   )
   ```

3. **Proper Error Handling**
   ```go
   if err := lock.Lock(ctx); err != nil {
       switch err {
       case redission.ErrLockTimeout:
           // Handle timeout
       case context.DeadlineExceeded:
           // Handle context timeout
       default:
           // Handle other errors
       }
   }
   ```

4. **Use defer for Unlocking**
   ```go
   if err := lock.Lock(ctx); err != nil {
       return err
   }
   defer lock.Unlock(ctx)
   ```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License
