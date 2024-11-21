package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/onionj/trust/internal/entity"
	"github.com/onionj/trust/pkg/maptostruct"
)

type commonBehavior[T entity.DBModel] struct {
	redis *redis.Client
}

func NewCommonBehavior[T entity.DBModel](redis *redis.Client) CommonBehaviorRepository[T] {
	return &commonBehavior[T]{
		redis: redis,
	}
}

func (c commonBehavior[T]) Save(ctx context.Context, model T) error {
	_, err := c.redis.HSet(ctx, model.EntityID().String(), model).Result()
	return err
}

// Get retrieves a specific key from Redis and converts it into the struct T
func (c commonBehavior[T]) Get(ctx context.Context, key string) (T, error) {
	// Fetch all fields for the key using HGetAll
	hash, err := c.redis.HGetAll(ctx, key).Result()
	if err != nil {
		return *new(T), fmt.Errorf("failed to retrieve key %s: %v", key, err)
	}

	// If the key doesn't exist, HGetAll will return an empty map
	if len(hash) == 0 {
		return *new(T), ErrNotFound
	}

	// Convert the hash map back to the struct
	var model T
	if err := maptostruct.MapToStruct(hash, &model); err != nil {
		return *new(T), fmt.Errorf("failed to map redis hash to struct: %v", err)
	}

	return model, nil
}

// Keys retrieves all keys matching the pattern
func (c commonBehavior[T]) Keys(ctx context.Context, pattern string) []string {
	return c.redis.Keys(context.Background(), pattern).Val()
}

// Scan retrieves all keys matching the pattern and fetches their associated values using a pipeline
func (c commonBehavior[T]) Scan(ctx context.Context, pattern string, limit int) ([]T, error) {
	var allKeys []string
	var cursor uint64

	// Use SCAN to retrieve keys in chunks
	for {
		keys, nextCursor, err := c.redis.Scan(ctx, cursor, pattern, 1000).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan keys: %v", err)
		}
		allKeys = append(allKeys, keys...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}

		if limit > 0 && len(allKeys) >= limit {
			break
		}
	}

	// Early return if no keys were found
	if len(allKeys) == 0 {
		return []T{}, nil
	}

	// Initialize slice to store results
	results := make([]T, len(allKeys))

	// Use pipelining to retrieve all the keys' values in parallel
	pipe := c.redis.Pipeline()
	cmds := make([]*redis.MapStringStringCmd, len(allKeys))

	for i, key := range allKeys {
		// Fetch all fields of the hash using HGetAll for each key
		cmds[i] = pipe.HGetAll(ctx, key)
	}

	// Execute all pipelined commands
	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute pipeline: %v", err)
	}

	// Process results
	for i, cmd := range cmds {
		hash, err := cmd.Result()
		if err != nil {
			return nil, fmt.Errorf("failed to get hash for key %s: %v", allKeys[i], err)
		}

		// Convert the hash map back to the struct
		var model T
		if err := maptostruct.MapToStruct(hash, &model); err != nil {
			return nil, fmt.Errorf("failed to map redis hash to struct: %v", err)
		}
		results[i] = model
	}

	return results, nil
}
