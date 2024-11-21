package entity

import (
	"fmt"
	"time"
)

// Define constants for each status
const (
	Active     string = "active"
	Completed  string = "completed"
	InProgress string = "in_progress"
)

const (
	Share string = "share"
	Steal string = "steal"
)

const (
	P1     string = "p1"
	P2     string = "p2"
	P1P2   string = "p1p2"
	Server string = "server"
)

// Game represents a game instance between two players
type Game struct {
	Id        uint   `json:"id" redis:"id"`                 // Id
	Created   uint   `json:"created" redis:"created"`       // Initial time
	P1ID      int64  `json:"p1_id" redis:"p1_id"`           // Foreign key to User
	P2ID      int64  `json:"p2_id" redis:"p2_id"`           // Foreign key to User
	Rounds    int    `json:"rounds" redis:"rounds"`         // Total number of rounds
	TimeLimit int    `json:"time_limit" redis:"time_limit"` // Decision time limit in seconds
	Coins     int    `json:"coins" redis:"coins"`           // Total coins
	Status    string `json:"status" redis:"status"`         // 'active' or 'completed'
	MaxSteal  int    `json:"max_steal" redis:"max_steal"`   // max steal per player

	R1P1Decision string `json:"r1_p1_decision" redis:"r1_p1_decision"` // '' or 'share' or 'steal'
	R1P2Decision string `json:"r1_p2_decision" redis:"r1_p2_decision"` // '' or 'share' or 'steal'
	R1Winner     string `json:"r1_winner" redis:"r1_winner"`           // '' or 'p1' or 'p2' or 'p1p2' or 'server'
	R1Status     string `json:"r1_status" redis:"r1_status"`           // 'in_progress' or 'completed'
	R1Rewards    int    `json:"r1_rewards" redis:"r1_rewards"`         // server rewards wen winner is p1p2

	R2P1Decision string `json:"r2_p1_decision" redis:"r2_p1_decision"` // '' or 'share' or 'steal'
	R2P2Decision string `json:"r2_p2_decision" redis:"r2_p2_decision"` // '' or 'share' or 'steal'
	R2Winner     string `json:"r2_winner" redis:"r2_winner"`           // '' or 'p1' or 'p2' or 'p1p2' or 'server'
	R2Status     string `json:"r2_status" redis:"r2_status"`           // 'in_progress' or 'completed'
	R2Rewards    int    `json:"r2_rewards" redis:"r2_rewards"`         // server rewards wen winner is p1p2

	R3P1Decision string `json:"r3_p1_decision" redis:"r3_p1_decision"` // '' or 'share' or 'steal'
	R3P2Decision string `json:"r3_p2_decision" redis:"r3_p2_decision"` // '' or 'share' or 'steal'
	R3Winner     string `json:"r3_winner" redis:"r3_winner"`           // '' or 'p1' or 'p2' or 'p1p2' or 'server'
	R3Status     string `json:"r3_status" redis:"r3_status"`           // 'in_progress' or 'completed'
	R3Rewards    int    `json:"r3_rewards" redis:"r3_rewards"`         // server rewards wen winner is p1p2

	R4P1Decision string `json:"r4_p1_decision" redis:"r4_p1_decision"` // '' or 'share' or 'steal'
	R4P2Decision string `json:"r4_p2_decision" redis:"r4_p2_decision"` // '' or 'share' or 'steal'
	R4Winner     string `json:"r4_winner" redis:"r4_winner"`           // '' or 'p1' or 'p2' or 'p1p2' or 'server'
	R4Status     string `json:"r4_status" redis:"r4_status"`           // 'in_progress' or 'completed'
	R4Rewards    int    `json:"r4_rewards" redis:"r4_rewards"`         // server rewards wen winner is p1p2
}

func NewGame(GameID uint, p1ID int64, p2ID int64) Game {
	return Game{
		Id:        GameID,
		Created:   uint(time.Now().Unix()),
		P1ID:      p1ID,
		P2ID:      p2ID,
		Rounds:    4,
		TimeLimit: 120,
		Coins:     400,
		Status:    Active,
		MaxSteal:  4,
	}
}

func (Game) Table() string {
	return "game"
}

func (g Game) EntityID() ID {
	return NewID(fmt.Sprintf("game:p%d:p%d", g.P1ID, g.P2ID), g.Id)
}
