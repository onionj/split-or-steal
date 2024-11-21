package repository

import (
	"context"
	"errors"

	"github.com/onionj/trust/internal/entity"
)

var (
	ErrNotFound = errors.New("entity not found")
)

type CommonBehaviorRepository[T entity.DBModel] interface {
	Save(ctx context.Context, model T) error
	Keys(ctx context.Context, pattern string) []string
	Scan(ctx context.Context, pattern string, limit int) ([]T, error)
	Get(ctx context.Context, key string) (T, error)
	// add more common behavior
}

type UserRepository interface {
	CommonBehaviorRepository[entity.User]
}

type GameRepository interface {
	CommonBehaviorRepository[entity.Game]
}
