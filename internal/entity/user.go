package entity

import (
	"math/rand"
	"time"
)

type User struct {
	Id              int64  `json:"id" redis:"id"`
	Created         int64  `json:"created" redis:"created"`
	DisplayName     string `json:"display_name" redis:"display_name"`
	Balance         int    `json:"balance" redis:"balance"`
	AvatarID        int    `json:"avatar_id" redis:"avatar_id"`
	HourLimit       int    `json:"hour_limit" redis:"hour_limit"`
	LastGamesResult string `json:"last_games_result" redis:"last_games_result"`
}

var avatar_ids = [11]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}

func NewUser(id int64, displayName string, Balance int) User {
	return User{
		Id:              id,
		Created:         time.Now().Unix(),
		DisplayName:     displayName,
		Balance:         Balance,
		AvatarID:        avatar_ids[rand.Intn(len(avatar_ids))],
		HourLimit:       10,
		LastGamesResult: "",
	}

}

func (u User) Table() string {
	return "user"
}

func (u User) EntityID() ID {
	return NewID("user", u.Id)
}
