package schemas

import "github.com/onionj/trust/internal/entity"

type GameShortReport struct {
	CompetitorName     string
	CompetitorAvatarId string
	CompetitorCoins    string
	YourCoins          string
}

type MenuData struct {
	User            entity.User
	GameShortReport []GameShortReport
}

type GameData struct {
	Competitor     entity.User
	Game           entity.Game
	GameResults    map[string]string
	GameResultsSum string
}
