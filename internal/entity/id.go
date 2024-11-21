package entity

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type ID string

func (i ID) Type() string {
	return strings.Split(string(i), ":")[0]
}

func (i ID) ID() string {
	return strings.Split(string(i), ":")[1]
}
func (i ID) String() string {
	return string(i)
}

func NewID[T any](entityType string, id T) ID {
	return ID(fmt.Sprintf("%s:%v", entityType, id))
}

// getOrInitID sets the key to 1 if it doesn't exist and then increments it.
func GetOrInitID(rdb *redis.Client, key string) (uint, error) {
	// Attempt to set key to 1 if it doesn't exist
	set, err := rdb.SetNX(context.Background(), key, 1, 0).Result()
	if err != nil {
		return 0, err
	}
	// If key was set to 1 (meaning it didn't exist), return 1
	if set {
		return 1, nil
	}
	// If key exists, increment and return the new value
	i, err := rdb.Incr(context.Background(), key).Result()
	return uint(i), err
}
