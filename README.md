# Redission

Redission is a Redis-based distributed lock implementation in Go, inspired by Java's Redisson. It provides a robust and feature-rich distributed locking mechanism with support for automatic lock renewal (watchdog mechanism).

## Features

- Distributed locking with Redis
- Automatic lock renewal (watchdog mechanism)
- Configurable wait timeout
- Configurable lease time
- Manual lock refresh support
- Context support for better control
- Thread-safe implementation
- Lua script ensures atomic operations
- Inspired by Java's Redisson implementation

## Installation

```bash
go get github.com/huimingz/redission
```

## Quick Start

```go
package main

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
    
    // Create lock client
    client := redission.NewClient(redisClient)
    
    // Create a new lock with options
    lock := client.NewLock("my-lock",
        redission.WithWaitTimeout(5*time.Second),    // Wait up to 5 seconds to acquire lock
        redission.WithLeaseTime(30*time.Second),     // Lock expires after 30 seconds
        redission.WithWatchDog(true),                // Enable auto-renewal
    )
    
    ctx := context.Background()
    
    // Try to acquire the lock
    if err := lock.Lock(ctx); err != nil {
        panic(err)
    }
    
    // Don't forget to unlock
    defer lock.Unlock(ctx)
    
    // Your business logic here
    // ...
}
```

## Advanced Usage

### TryLock

```go
// Try to acquire the lock without waiting
acquired, err := lock.TryLock(ctx)
if err != nil {
    panic(err)
}
if !acquired {
    // Lock is held by someone else
    return
}
```

### Manual Refresh

```go
// Manually extend the lock's lease time
if err := lock.Refresh(ctx); err != nil {
    panic(err)
}
```

### Watch Dog (Automatic Renewal)

```go
// Create a lock with watch dog enabled
lock := client.NewLock("my-lock",
    redission.WithWatchDog(true),              // Enable auto-renewal
    redission.WithWatchDogTimeout(30*time.Second), // Renewal interval
)
```

## Configuration Options

- `WithWaitTimeout(d time.Duration)`: Set maximum time to wait for lock acquisition
- `WithLeaseTime(d time.Duration)`: Set lock expiration time
- `WithWatchDog(enable bool)`: Enable/disable automatic lock renewal
- `WithWatchDogTimeout(d time.Duration)`: Set watch dog renewal interval

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Author

HuimingZ
