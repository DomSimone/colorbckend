package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// RedisClient represents a Redis client connection
type RedisClient struct {
	*redis.Client
	logger zerolog.Logger
}

// NewRedisClient creates a new Redis client connection
func NewRedisClient(cfg RedisConfig) (*RedisClient, error) {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Override with environment variables if they exist
	if host := getEnv("REDIS_HOST", ""); host != "" {
		cfg.Host = host
	}
	if port := getEnv("REDIS_PORT", ""); port != "" {
		fmt.Sscanf(port, "%d", &cfg.Port)
	}
	if password := getEnv("REDIS_PASSWORD", ""); password != "" {
		cfg.Password = password
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: log.Writer()}).With().Timestamp().Logger()
	logger.Info().Msg("Connected to Redis")

	return &RedisClient{
		Client: client,
		logger: logger,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// GetClient returns the underlying redis.Client
func (r *RedisClient) GetClient() *redis.Client {
	return r.Client
}

// SetWithExpiration sets a key with an expiration time
func (r *RedisClient) SetWithExpiration(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return r.Client.Set(ctx, key, value, expiration).Err()
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	return r.Client.Get(ctx, key).Result()
}

// Delete deletes a key
func (r *RedisClient) Delete(ctx context.Context, key string) error {
	return r.Client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// Set sets a key without expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}) error {
	return r.Client.Set(ctx, key, value, 0).Err()
}

// Increment increments a key by 1
func (r *RedisClient) Increment(ctx context.Context, key string) error {
	return r.Client.Incr(ctx, key).Err()
}

// Decrement decrements a key by 1
func (r *RedisClient) Decrement(ctx context.Context, key string) error {
	return r.Client.Decr(ctx, key).Err()
}

// SetExpiration sets the expiration time for a key
func (r *RedisClient) SetExpiration(ctx context.Context, key string, expiration time.Duration) error {
	return r.Client.Expire(ctx, key, expiration).Err()
}

// GetExpiration gets the expiration time for a key
func (r *RedisClient) GetExpiration(ctx context.Context, key string) (time.Duration, error) {
	return r.Client.TTL(ctx, key).Result()
}

// Keys returns all keys matching a pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	return r.Client.Keys(ctx, pattern).Result()
}

// FlushDB clears the current database
func (r *RedisClient) FlushDB(ctx context.Context) error {
	return r.Client.FlushDB(ctx).Err()
}

// getEnv returns the value of the environment variable or the default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}