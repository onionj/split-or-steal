package repository

import (
	"context"
	"testing"

	"github.com/onionj/trust/config"
	"github.com/onionj/trust/db"
	"github.com/onionj/trust/internal/entity"
	"github.com/stretchr/testify/assert"
)

func TestGameRepository(t *testing.T) {
	cfg := config.NewConfig("../../.env")
	cfg.Redis.DB = 1 // Test DB

	redis := db.Init(cfg)
	err := redis.FlushDB(context.Background()).Err()
	assert.NoError(t, err)

	userRepo := NewUserRepository(redis)
	gameRepo := NewGameRepository(redis)

	user := entity.NewUser(10, "Onion", 10000)
	err = userRepo.Save(context.Background(), user)
	assert.NoError(t, err)

	user2 := entity.NewUser(11, "Sarah", 10000)
	err = userRepo.Save(context.Background(), user2)
	assert.NoError(t, err)

	game := entity.NewGame(1, 10, 11)
	err = gameRepo.Save(context.Background(), game)
	assert.NoError(t, err)

	dbGame, err := gameRepo.Get(context.Background(), "game:p10:p11:1")
	assert.NoError(t, err)

	assert.Equal(t, game.Created, dbGame.Created)
	assert.Equal(t, game.Id, dbGame.Id)
	assert.Equal(t, game.P1ID, dbGame.P1ID)
	assert.Equal(t, game.P2ID, dbGame.P2ID)
	assert.Equal(t, game.Rounds, dbGame.Rounds)
	assert.Equal(t, game.TimeLimit, dbGame.TimeLimit)
	assert.Equal(t, game.Coins, dbGame.Coins)
	assert.Equal(t, game.Status, dbGame.Status)

	dbGames, err := gameRepo.Scan(context.Background(), "game*:p10*", 1)
	assert.NoError(t, err)
	assert.Len(t, dbGames, 1)
	assert.Equal(t, game.Created, dbGames[0].Created)
	assert.Equal(t, game.Id, dbGames[0].Id)
	assert.Equal(t, game.P1ID, dbGames[0].P1ID)
	assert.Equal(t, game.P2ID, dbGames[0].P2ID)
	assert.Equal(t, game.Rounds, dbGames[0].Rounds)
	assert.Equal(t, game.TimeLimit, dbGames[0].TimeLimit)
	assert.Equal(t, game.Coins, dbGames[0].Coins)
	assert.Equal(t, game.Status, dbGames[0].Status)

	game.Status = entity.Completed
	game.R1P1Decision = entity.Share
	game.R1P2Decision = entity.Steal
	game.R1Winner = entity.P2
	game.R1Status = entity.Completed
	game.R1Rewards = 0

	err = gameRepo.Save(context.Background(), game)
	assert.NoError(t, err)

	dbGameNew, err := gameRepo.Get(context.Background(), "game:p10:p11:1")
	assert.NoError(t, err)

	assert.Equal(t, game.Created, dbGameNew.Created)
	assert.Equal(t, game.Id, dbGameNew.Id)
	assert.Equal(t, game.P1ID, dbGameNew.P1ID)
	assert.Equal(t, game.P2ID, dbGameNew.P2ID)
	assert.Equal(t, game.Rounds, dbGameNew.Rounds)
	assert.Equal(t, game.TimeLimit, dbGameNew.TimeLimit)
	assert.Equal(t, game.Coins, dbGameNew.Coins)
	assert.Equal(t, entity.Completed, dbGameNew.Status)
	assert.Equal(t, entity.Share, dbGameNew.R1P1Decision)
	assert.Equal(t, entity.Steal, dbGameNew.R1P2Decision)
	assert.Equal(t, entity.P2, dbGameNew.R1Winner)
	assert.Equal(t, entity.Completed, dbGameNew.R1Status)
	assert.Equal(t, 0, dbGameNew.R1Rewards)
}
