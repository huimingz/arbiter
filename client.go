package arbiter

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/redis/go-redis/v9"
)

const (
	defaultKeyPrefix = "arbiter:"
)

// Client represents a distributed lock client
type Client struct {
	redis  *redis.Client
	logger Logger
	prefix string
}

// ClientOption is a function type for setting client options
type ClientOption func(*Client)

// WithLogger sets the logger for the client
func WithLogger(logger Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithKeyPrefix sets a custom prefix for Redis keys
func WithKeyPrefix(prefix string) ClientOption {
	return func(c *Client) {
		c.prefix = prefix
	}
}

// NewClient creates a new distributed lock client
func NewClient(redis *redis.Client, opts ...ClientOption) *Client {
	c := &Client{
		redis:  redis,
		logger: newDefaultLogger(),
		prefix: defaultKeyPrefix,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NewLock creates a new distributed lock instance
func (c *Client) NewLock(name string, opts ...Option) Lock {
	options := defaultOptions()
	for _, opt := range opts {
		opt(options)
	}

	return newLock(c.redis, fmt.Sprintf("%s%s", c.prefix, name), options, c.logger)
}

// generateValue generates a random string as lock value
func generateValue() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		panic(err) // This should never happen
	}
	return base64.StdEncoding.EncodeToString(b)
}
