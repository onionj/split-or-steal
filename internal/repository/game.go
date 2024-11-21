package repository

import (
	"github.com/onionj/trust/internal/entity"
	"github.com/redis/go-redis/v9"
)

var _ GameRepository = (*gameRepository)(nil) // implement check

type gameRepository struct {
	redis *redis.Client
	CommonBehaviorRepository[entity.Game]
}

func NewGameRepository(redis *redis.Client) GameRepository {
	return &gameRepository{
		CommonBehaviorRepository: NewCommonBehavior[entity.Game](redis),
	}
}
