package repository

import (
	"github.com/onionj/trust/internal/entity"
	"github.com/redis/go-redis/v9"
)

var _ UserRepository = (*userRepository)(nil) // implement check

type userRepository struct {
	redis *redis.Client
	CommonBehaviorRepository[entity.User]
}

func NewUserRepository(redis *redis.Client) UserRepository {
	return &userRepository{
		CommonBehaviorRepository: NewCommonBehavior[entity.User](redis),
	}
}
