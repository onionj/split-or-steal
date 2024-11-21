package repository

import (
	"context"
	"testing"

	"github.com/onionj/trust/config"
	"github.com/onionj/trust/db"
	"github.com/onionj/trust/internal/entity"
	"github.com/stretchr/testify/assert"
)

func TestUserRepository(t *testing.T) {
	cfg := config.NewConfig("../../.env")
	cfg.Redis.DB = 1 // Test DB

	redis := db.Init(cfg)
	err := redis.FlushDB(context.Background()).Err()
	assert.NoError(t, err)

	userRepo := NewUserRepository(redis)

	user := entity.NewUser(10, "Onion", 10000)
	err = userRepo.Save(context.Background(), user)
	assert.NoError(t, err)

	new_user, err := userRepo.Get(context.Background(), "user:10")
	assert.NoError(t, err)
	assert.Equal(t, user.Id, new_user.Id)
	assert.Equal(t, user.DisplayName, new_user.DisplayName)
	assert.Equal(t, user.Balance, new_user.Balance)

	users, err := userRepo.Scan(context.Background(), "user:1*", 0)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, user.Id, users[0].Id)
	assert.Equal(t, user.DisplayName, users[0].DisplayName)
	assert.Equal(t, user.Balance, users[0].Balance)

	user2 := entity.NewUser(11, "Sarah", 10000)
	err = userRepo.Save(context.Background(), user2)
	assert.NoError(t, err)

	users2, err := userRepo.Scan(context.Background(), "user:1*", 0)
	assert.NoError(t, err)
	assert.Len(t, users2, 2)

}
