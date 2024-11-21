package webhandlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/bsm/redislock"
	"github.com/labstack/echo/v4"
	"github.com/onionj/trust/app"
	"github.com/onionj/trust/app/schemas"
	"github.com/onionj/trust/internal/entity"
	"github.com/onionj/trust/pkg/ratelimit"
	"github.com/sirupsen/logrus"
)

// Embed the HTML file directly into the binary
//
//go:embed templates/home.html
var homeHTML string

//go:embed templates/menu.html
var menuHTML string

//go:embed templates/game.html
var gameHTML string

//go:embed templates/notification.html
var notificationHTML string

const DEFAULT_LOBBY_NAME = "trust:default_lobby:name"
const DEFAULT_LOBBY_LOCK = "trust:default_lobby:lock"
const USER_LOCK = "trust:user%d:lock"
const USER_LOCK_BALANCE = "trust:user%d:lock_balance"
const GAME_LOCK = "trust:game%d:lock"
const GAME_INDEX = "trust:game:index"
const GAME_USER_HOUR_LIMIT = "trust:user%d:hour:limit"

type GameHandlers struct {
	server *app.Server
	locker *redislock.Client
}

func NewGameHandlers(server *app.Server) *GameHandlers {
	return &GameHandlers{server: server, locker: redislock.New(server.DB)}
}

func (g *GameHandlers) OpenHome(c echo.Context) error {
	tmpl, err := template.New("home").Parse(homeHTML)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to render home")
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to render home")
	}

	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

// Serve the menu page
func (g *GameHandlers) OpenMenu(c echo.Context) error {
	user := app.GetUserFromCtx(c)

	gamesReportRaw := strings.Split(user.LastGamesResult, "|")
	slices.Reverse(gamesReportRaw)

	filteredGamesReport := gamesReportRaw[:0]
	for _, game := range gamesReportRaw {
		if game != "" {
			filteredGamesReport = append(filteredGamesReport, game)
		}
	}

	if len(filteredGamesReport) > 6 {
		filteredGamesReport = filteredGamesReport[:6]
	}

	gameShortReports := make([]schemas.GameShortReport, len(filteredGamesReport))

	for idx, shortReport := range filteredGamesReport {

		// "|%d:%d:%d:%d" game.Id, p1, game.P2ID, p2
		shortReportData := strings.Split(shortReport, ":")
		gameShortReports[idx].YourCoins = shortReportData[1]
		gameShortReports[idx].CompetitorCoins = shortReportData[3]

		competitor, err := g.server.UserRepo.Get(context.Background(), fmt.Sprintf("user:%s", shortReportData[2])) // TODO Get all in one pipe
		if err == nil {
			gameShortReports[idx].CompetitorName = competitor.DisplayName
			gameShortReports[idx].CompetitorAvatarId = fmt.Sprint(competitor.AvatarID)
		}
	}

	tmpl, err := template.New("menu").Parse(menuHTML)
	if err != nil {
		logrus.Error("Failed to render menu ", err)
		return c.JSON(http.StatusInternalServerError, "Failed to render menu")
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, schemas.MenuData{User: user, GameShortReport: gameShortReports})
	if err != nil {
		logrus.Error("Failed to render menu ", err)
		return c.JSON(http.StatusInternalServerError, "Failed to render menu")
	}

	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

// Start And Serve the game
func (g *GameHandlers) StartGame(c echo.Context) error {
	user := app.GetUserFromCtx(c)
	ctx := context.Background()

	// Lock User ID
	userLock, err := g.locker.Obtain(
		ctx,
		fmt.Sprintf(USER_LOCK, user.Id),
		30*time.Second,
		&redislock.Options{
			RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(1*time.Second), 31)},
	)
	if err != nil {
		logrus.Error("user lock error ", err)
		return errors.New("server error")
	}
	defer userLock.Release(ctx)

	// Check if user is in any Active Game
	dbGames, err := g.server.GameRepo.Scan(ctx, fmt.Sprintf("game:*p%d*", user.Id), 1000) // TODO: filter on status?
	if err != nil {
		logrus.Error("GameRepo.Scan error ", err)
		return errors.New("server error")
	}
	if len(dbGames) > 0 {
		for _, game := range dbGames {
			if game.Status == entity.Active {
				return renderGamePage(c, g, user, game)
			}
		}
	}

	isLimited, _ := ratelimit.IsLimited(g.server.DB, fmt.Sprintf(GAME_USER_HOUR_LIMIT, int(user.Id)), user.HourLimit, time.Hour)
	if isLimited {
		logrus.Warn("User Limited")
		return showNotification(c, "You've reached your game limit for this hour and can't start a new game just yet. Please try again in an hour to continue playing!!")
	}

	// Lock Default lobby
	lock, err := g.locker.Obtain(
		ctx,
		DEFAULT_LOBBY_LOCK,
		5*time.Second,
		&redislock.Options{
			RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(500*time.Millisecond), 5)},
	)
	if err != nil {
		logrus.Error("default lobby lock error ", err)
		return c.JSON(http.StatusInternalServerError, "default lobby error (0)")
	}
	defer lock.Release(ctx)

	// Check if any user is in global lobby
	defaultLobby, _ := g.server.DB.Get(ctx, DEFAULT_LOBBY_NAME).Result()

	if defaultLobby != "" {
		defaultLobbyUserID, err := strconv.ParseInt(defaultLobby, 10, 64)
		if err != nil {
			logrus.Error("defaultLobbyUserID parse error ", err)
			return c.JSON(http.StatusInternalServerError, "default lobby error (1)")
		}
		if defaultLobbyUserID == user.Id {
			return c.JSON(http.StatusInternalServerError, "default lobby error (2)")
		}

		g.server.DB.Del(ctx, DEFAULT_LOBBY_NAME)
		new_game_id, err := entity.GetOrInitID(g.server.DB, GAME_INDEX)
		if err != nil {
			logrus.Error("save new game error ", err)
			return c.JSON(http.StatusInternalServerError, "default lobby error (3)")
		}

		newGame := entity.NewGame(new_game_id, user.Id, defaultLobbyUserID)
		err = g.server.GameRepo.Save(ctx, newGame)
		if err != nil {
			logrus.Error("save new game error (1) ", err)
			return c.JSON(http.StatusInternalServerError, "default lobby error (4)")
		}

		ratelimit.BurnToken(g.server.DB, fmt.Sprintf(GAME_USER_HOUR_LIMIT, int(user.Id)))
		return renderGamePage(c, g, user, newGame)
	}

	// Set current user to default lobby
	g.server.DB.Set(ctx, DEFAULT_LOBBY_NAME, fmt.Sprint(user.Id), 30*time.Second)
	lock.Release(ctx)

	// Wait for Game
	for i := 0; i < 62; i++ {
		time.Sleep(500 * time.Millisecond)

		dbGames, _ := g.server.GameRepo.Scan(ctx, fmt.Sprintf("game*:p%d*", user.Id), 1000) // TODO: filter on status?
		if len(dbGames) > 0 {
			for _, game := range dbGames {
				if game.Status == entity.Active {
					ratelimit.BurnToken(g.server.DB, fmt.Sprintf(GAME_USER_HOUR_LIMIT, int(user.Id)))
					return renderGamePage(c, g, user, game)
				}
			}
		}
	}

	return showNotification(c, "No active game found.")
}

func (g *GameHandlers) GetGameUpdate(c echo.Context) error {
	user := app.GetUserFromCtx(c)
	gameId := c.Param("gameID")

	dbGames, err := g.server.GameRepo.Scan(context.Background(), fmt.Sprintf("game:*p%d*:%s", user.Id, gameId), 1)
	if err != nil {
		logrus.Error("GameRepo.Scan error ", err)
		return errors.New("server error")
	}
	if len(dbGames) < 1 {
		return showNotification(c, "Game Not Found.")
	}

	return renderGamePage(c, g, user, dbGames[0])
}

func (g *GameHandlers) GameChoice(c echo.Context) error {
	user := app.GetUserFromCtx(c)
	ctx := context.Background()

	gameId, err := strconv.Atoi(c.Param("gameID"))
	if err != nil {
		return showNotification(c, "Invalid game ID.")
	}

	roundId := c.Param("roundID")
	if roundId != "1" && roundId != "2" && roundId != "3" && roundId != "4" {
		return showNotification(c, "Invalid round id.")
	}

	choice := c.Param("choice")
	if choice != entity.Share && choice != entity.Steal {
		return showNotification(c, "Invalid choice.")
	}

	dbGamesNames := g.server.GameRepo.Keys(ctx, fmt.Sprintf("game:*p%d*:%d", user.Id, gameId))

	if len(dbGamesNames) != 1 {
		return showNotification(c, "Active Game Not Found.")
	}

	gameLock, err := g.locker.Obtain(
		ctx,
		fmt.Sprintf(GAME_LOCK, gameId),
		5*time.Second,
		&redislock.Options{
			RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(100*time.Millisecond), 100)},
	)
	if err != nil {
		logrus.Error("game lock error ", err)
		return c.JSON(http.StatusInternalServerError, "game error (0)")
	}
	defer gameLock.Release(ctx)

	game, err := g.server.GameRepo.Get(ctx, dbGamesNames[0])
	if err != nil {
		logrus.Error("game get error ", err)
		return c.JSON(http.StatusInternalServerError, "game error (1)")
	}

	if choice == entity.Steal {
		playerChoices := []string{}
		if game.P1ID == user.Id {
			playerChoices = []string{game.R1P1Decision, game.R2P1Decision, game.R3P1Decision, game.R4P1Decision}
		} else if game.P2ID == user.Id {
			playerChoices = []string{game.R1P2Decision, game.R2P2Decision, game.R3P2Decision, game.R4P2Decision}
		}

		StealsCount := 0

		for _, choice := range playerChoices {
			if choice == entity.Steal {
				StealsCount += 1
			}
		}
		if StealsCount >= game.MaxSteal {
			return showNotification(c, "You can not steal anymore")
		}
	}

	if roundId == "1" {
		if game.P1ID == user.Id && game.R1P1Decision == "" {
			game.R1P1Decision = choice
		} else if game.P2ID == user.Id && game.R1P2Decision == "" {
			game.R1P2Decision = choice
		}
	} else if roundId == "2" {
		if game.P1ID == user.Id && game.R2P1Decision == "" && game.R1P1Decision != "" {
			game.R2P1Decision = choice
		} else if game.P2ID == user.Id && game.R2P2Decision == "" && game.R1P2Decision != "" {
			game.R2P2Decision = choice
		}
	} else if roundId == "3" {
		if game.P1ID == user.Id && game.R3P1Decision == "" && game.R2P1Decision != "" {
			game.R3P1Decision = choice
		} else if game.P2ID == user.Id && game.R3P2Decision == "" && game.R2P2Decision != "" {
			game.R3P2Decision = choice
		}
	} else if roundId == "4" {
		if game.P1ID == user.Id && game.R4P1Decision == "" && game.R3P1Decision != "" {
			game.R4P1Decision = choice
		} else if game.P2ID == user.Id && game.R4P2Decision == "" && game.R3P2Decision != "" {
			game.R4P2Decision = choice
		}
	}

	err = g.updateGameResults(game)
	if err != nil {
		logrus.Error(gameId, " game not saved", err)
		return showNotification(c, "Game Not Saved!.")
	}

	return renderGamePage(c, g, user, game)
}

// Helper function to update game results with user choices
func (g *GameHandlers) updateGameResults(game entity.Game) error {
	err := g.server.GameRepo.Save(context.Background(), game)
	if err != nil {
		logrus.Error(game.Id, " game not saved", err)
		return err
	}

	if game.R1Status != entity.Completed {
		if game.R1P1Decision == entity.Share && game.R1P2Decision == entity.Share {
			game.R1Winner = entity.P1P2
			game.R1Status = entity.Completed
			game.R1Rewards = game.Coins / 40
		} else if game.R1P1Decision == entity.Steal && game.R1P2Decision == entity.Steal {
			game.R1Winner = entity.Server
			game.R1Status = entity.Completed
		} else if game.R1P1Decision == entity.Share && game.R1P2Decision == entity.Steal {
			game.R1Winner = entity.P2
			game.R1Status = entity.Completed
		} else if game.R1P1Decision == entity.Steal && game.R1P2Decision == entity.Share {
			game.R1Winner = entity.P1
			game.R1Status = entity.Completed
		}
	}
	if game.R2Status != entity.Completed {
		if game.R2P1Decision == entity.Share && game.R2P2Decision == entity.Share {
			game.R2Winner = entity.P1P2
			game.R2Status = entity.Completed
			game.R2Rewards = game.Coins / 40
		} else if game.R2P1Decision == entity.Steal && game.R2P2Decision == entity.Steal {
			game.R2Winner = entity.Server
			game.R2Status = entity.Completed
		} else if game.R2P1Decision == entity.Share && game.R2P2Decision == entity.Steal {
			game.R2Winner = entity.P2
			game.R2Status = entity.Completed
		} else if game.R2P1Decision == entity.Steal && game.R2P2Decision == entity.Share {
			game.R2Winner = entity.P1
			game.R2Status = entity.Completed
		}
	}
	if game.R3Status != entity.Completed {
		if game.R3P1Decision == entity.Share && game.R3P2Decision == entity.Share {
			game.R3Winner = entity.P1P2
			game.R3Status = entity.Completed
			game.R3Rewards = game.Coins / 40
		} else if game.R3P1Decision == entity.Steal && game.R3P2Decision == entity.Steal {
			game.R3Winner = entity.Server
			game.R3Status = entity.Completed
		} else if game.R3P1Decision == entity.Share && game.R3P2Decision == entity.Steal {
			game.R3Winner = entity.P2
			game.R3Status = entity.Completed
		} else if game.R3P1Decision == entity.Steal && game.R3P2Decision == entity.Share {
			game.R3Winner = entity.P1
			game.R3Status = entity.Completed
		}
	}
	if game.R4Status != entity.Completed {
		if game.R4P1Decision == entity.Share && game.R4P2Decision == entity.Share {
			game.R4Winner = entity.P1P2
			game.R4Status = entity.Completed
			game.R4Rewards = game.Coins / 40
		} else if game.R4P1Decision == entity.Steal && game.R4P2Decision == entity.Steal {
			game.R4Winner = entity.Server
			game.R4Status = entity.Completed
		} else if game.R4P1Decision == entity.Share && game.R4P2Decision == entity.Steal {
			game.R4Winner = entity.P2
			game.R4Status = entity.Completed
		} else if game.R4P1Decision == entity.Steal && game.R4P2Decision == entity.Share {
			game.R4Winner = entity.P1
			game.R4Status = entity.Completed
		}
	}

	// calculate rewards and player coins
	server := 0
	p1 := 0
	p2 := 0

	if game.Status != entity.Completed && game.R4Status == entity.Completed {
		game.Status = entity.Completed

		gameWinners := []string{game.R1Winner, game.R2Winner, game.R3Winner, game.R4Winner}
		gameRewards := []int{game.R1Rewards, game.R2Rewards, game.R3Rewards, game.R4Rewards}
		perRoundCoin := game.Coins / game.Rounds

		for idx, winner := range gameWinners {
			if winner == entity.Server {
				server += perRoundCoin
			} else if winner == entity.P1 {
				p1 += perRoundCoin
			} else if winner == entity.P2 {
				p2 += perRoundCoin
			} else if winner == entity.P1P2 {
				p2 += (perRoundCoin / 2)
				p1 += (perRoundCoin / 2)
				if gameRewards[idx] > 1 {
					p1 += gameRewards[idx] / 2
					p2 += gameRewards[idx] / 2
				}
			}
		}

		// save p1 balance
		p1BalanceLock, err := g.locker.Obtain(
			context.Background(),
			fmt.Sprintf(USER_LOCK_BALANCE, game.P1ID),
			5*time.Second,
			&redislock.Options{
				RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(100*time.Millisecond), 10)},
		)
		if err != nil {
			logrus.Error("user balance lock error ", err)
		} else {
			defer p1BalanceLock.Release(context.Background())
			p1User, err := g.server.UserRepo.Get(context.Background(), fmt.Sprintf("user:%d", game.P1ID))
			if err != nil {
				logrus.Error("Get p1 error ", err)
			} else {
				p1User.Balance += p1
				p1User.LastGamesResult += fmt.Sprintf("|%d:%d:%d:%d", game.Id, p1, game.P2ID, p2)
				// TODO: Remove very Old Games
				g.server.UserRepo.Save(context.Background(), p1User)
			}
		}

		// save p2 balance
		p2BalanceLock, err := g.locker.Obtain(
			context.Background(),
			fmt.Sprintf(USER_LOCK_BALANCE, game.P2ID),
			5*time.Second,
			&redislock.Options{
				RetryStrategy: redislock.LimitRetry(redislock.LinearBackoff(100*time.Millisecond), 10)},
		)
		if err != nil {
			logrus.Error("user balance lock error ", err)
		} else {
			defer p2BalanceLock.Release(context.Background())
			p2User, err := g.server.UserRepo.Get(context.Background(), fmt.Sprintf("user:%d", game.P2ID))
			if err != nil {
				logrus.Error("Get p2 error ", err)
			} else {
				p2User.Balance += p2
				p2User.LastGamesResult += fmt.Sprintf("|%d:%d:%d:%d", game.Id, p2, game.P1ID, p1)
				// TODO: Remove very Old Games
				g.server.UserRepo.Save(context.Background(), p2User)
			}
		}
	}

	return g.server.GameRepo.Save(context.Background(), game)
}

// Helper function to render the game page
func renderGamePage(c echo.Context, g *GameHandlers, user entity.User, game entity.Game) error {
	competitorId := int64(0)
	if game.P1ID != user.Id {
		competitorId = game.P1ID
	} else {
		competitorId = game.P2ID
	}

	competitor, err := g.server.UserRepo.Get(context.Background(), fmt.Sprintf("user:%d", competitorId))
	if err != nil {
		logrus.Error("cant find competitor", err)
		return c.JSON(http.StatusInternalServerError, "cant find competitor")
	}

	gameSum := c.QueryParam("gameSum")

	perRoundCoins := game.Coins / game.Rounds
	gameResults := map[string]string{
		"gameId":                   fmt.Sprint(game.Id),
		"gameStatus":               fmt.Sprint(game.Status),
		"round1Coins":              fmt.Sprint(perRoundCoins),
		"round2Coins":              fmt.Sprint(perRoundCoins),
		"round3Coins":              fmt.Sprint(perRoundCoins),
		"round4Coins":              fmt.Sprint(perRoundCoins),
		"round1YourDecision":       "",
		"round2YourDecision":       "",
		"round3YourDecision":       "",
		"round4YourDecision":       "",
		"round1CompetitorDecision": "",
		"round2CompetitorDecision": "",
		"round3CompetitorDecision": "",
		"round4CompetitorDecision": "",
		"round1Result":             "0",
		"round2Result":             "0",
		"round3Result":             "0",
		"round4Result":             "0",
		"AllRoundResult":           "0",
		"AllRoundCoins":            fmt.Sprint(game.Coins),
		"ActiveRound":              "1",
		"StealActive":              "true",
		"StealCount":               fmt.Sprint(game.MaxSteal),
	}

	newGameSum := ""

	for range 100 {

		if game.P1ID == user.Id {
			gameResults["round1YourDecision"] = game.R1P1Decision
			if game.R1P1Decision != "" {
				gameResults["round1CompetitorDecision"] = game.R1P2Decision
			}
			gameResults["round2YourDecision"] = game.R2P1Decision
			if game.R2P1Decision != "" {
				gameResults["round2CompetitorDecision"] = game.R2P2Decision
			}
			gameResults["round3YourDecision"] = game.R3P1Decision
			if game.R3P1Decision != "" {
				gameResults["round3CompetitorDecision"] = game.R3P2Decision
			}
			gameResults["round4YourDecision"] = game.R4P1Decision
			if game.R4P1Decision != "" {
				gameResults["round4CompetitorDecision"] = game.R4P2Decision
			}

			if game.R1Winner == entity.P1 {
				gameResults["round1Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R1Winner == entity.P2 || game.R1Winner == entity.Server {
				gameResults["round1Result"] = "0"
			} else if game.R1Winner == entity.P1P2 {
				gameResults["round1Result"] = fmt.Sprint((perRoundCoins + game.R1Rewards) / 2)
			}

			if game.R2Winner == entity.P1 {
				gameResults["round2Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R2Winner == entity.P2 || game.R2Winner == entity.Server {
				gameResults["round2Result"] = "0"
			} else if game.R2Winner == entity.P1P2 {
				gameResults["round2Result"] = fmt.Sprint((perRoundCoins + game.R2Rewards) / 2)
			}

			if game.R3Winner == entity.P1 {
				gameResults["round3Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R3Winner == entity.P2 || game.R3Winner == entity.Server {
				gameResults["round3Result"] = "0"
			} else if game.R3Winner == entity.P1P2 {
				gameResults["round3Result"] = fmt.Sprint((perRoundCoins + game.R3Rewards) / 2)
			}

			if game.R4Winner == entity.P1 {
				gameResults["round4Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R4Winner == entity.P2 || game.R4Winner == entity.Server {
				gameResults["round4Result"] = "0"
			} else if game.R4Winner == entity.P1P2 {
				gameResults["round4Result"] = fmt.Sprint((perRoundCoins + game.R4Rewards) / 2)
			}
		}

		if game.P2ID == user.Id {
			gameResults["round1YourDecision"] = game.R1P2Decision
			if game.R1P2Decision != "" {
				gameResults["round1CompetitorDecision"] = game.R1P1Decision
			}
			gameResults["round2YourDecision"] = game.R2P2Decision
			if game.R2P2Decision != "" {
				gameResults["round2CompetitorDecision"] = game.R2P1Decision
			}
			gameResults["round3YourDecision"] = game.R3P2Decision
			if game.R3P2Decision != "" {
				gameResults["round3CompetitorDecision"] = game.R3P1Decision
			}
			gameResults["round4YourDecision"] = game.R4P2Decision
			if game.R4P2Decision != "" {
				gameResults["round4CompetitorDecision"] = game.R4P1Decision
			}

			if game.R1Winner == entity.P2 {
				gameResults["round1Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R1Winner == entity.P1 || game.R1Winner == entity.Server {
				gameResults["round1Result"] = "0"
			} else if game.R1Winner == entity.P1P2 {
				gameResults["round1Result"] = fmt.Sprint((perRoundCoins + game.R1Rewards) / 2)
			}

			if game.R2Winner == entity.P2 {
				gameResults["round2Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R2Winner == entity.P1 || game.R2Winner == entity.Server {
				gameResults["round2Result"] = "0"
			} else if game.R2Winner == entity.P1P2 {
				gameResults["round2Result"] = fmt.Sprint((perRoundCoins + game.R2Rewards) / 2)
			}

			if game.R3Winner == entity.P2 {
				gameResults["round3Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R3Winner == entity.P1 || game.R3Winner == entity.Server {
				gameResults["round3Result"] = "0"
			} else if game.R3Winner == entity.P1P2 {
				gameResults["round3Result"] = fmt.Sprint((perRoundCoins + game.R3Rewards) / 2)
			}

			if game.R4Winner == entity.P2 {
				gameResults["round4Result"] = fmt.Sprint(perRoundCoins)
			} else if game.R4Winner == entity.P1 || game.R4Winner == entity.Server {
				gameResults["round4Result"] = "0"
			} else if game.R4Winner == entity.P1P2 {
				gameResults["round4Result"] = fmt.Sprint((perRoundCoins + game.R4Rewards) / 2)
			}
		}

		r1s, _ := strconv.Atoi(gameResults["round1Result"])
		r2s, _ := strconv.Atoi(gameResults["round2Result"])
		r3s, _ := strconv.Atoi(gameResults["round3Result"])
		r4s, _ := strconv.Atoi(gameResults["round4Result"])
		finalRoundsResult := 0

		if r1s > 0 {
			finalRoundsResult += r1s
		}
		if r2s > 0 {
			finalRoundsResult += r2s
		}
		if r3s > 0 {
			finalRoundsResult += r3s
		}
		if r4s > 0 {
			finalRoundsResult += r4s
		}
		gameResults["AllRoundResult"] = fmt.Sprint(finalRoundsResult)

		if gameResults["round4YourDecision"] != "" {
			gameResults["ActiveRound"] = "-1"
		} else if gameResults["round3YourDecision"] != "" {
			gameResults["ActiveRound"] = "4"
		} else if gameResults["round2YourDecision"] != "" {
			gameResults["ActiveRound"] = "3"
		} else if gameResults["round1YourDecision"] != "" {
			gameResults["ActiveRound"] = "2"
		}

		playerChoices := []string{}
		if game.P1ID == user.Id {
			playerChoices = []string{game.R1P1Decision, game.R2P1Decision, game.R3P1Decision, game.R4P1Decision}
		} else if game.P2ID == user.Id {
			playerChoices = []string{game.R1P2Decision, game.R2P2Decision, game.R3P2Decision, game.R4P2Decision}
		}
		StealsCount := 0

		for _, choice := range playerChoices {
			if choice == entity.Steal {
				StealsCount += 1
			}
		}
		gameResults["StealCount"] = fmt.Sprint(game.MaxSteal - StealsCount)
		if game.MaxSteal <= StealsCount {
			gameResults["StealActive"] = "false"
		}

		// Generate data hash for long poling algorithm
		shaHash := sha256.New()
		shaHash.Write([]byte(
			gameResults["gameId"] +
				gameResults["gameStatus"] +
				gameResults["round1Coins"] +
				gameResults["round2Coins"] +
				gameResults["round3Coins"] +
				gameResults["round4Coins"] +
				gameResults["round1YourDecision"] +
				gameResults["round2YourDecision"] +
				gameResults["round3YourDecision"] +
				gameResults["round4YourDecision"] +
				gameResults["round1CompetitorDecision"] +
				gameResults["round2CompetitorDecision"] +
				gameResults["round3CompetitorDecision"] +
				gameResults["round4CompetitorDecision"] +
				gameResults["round1Result"] +
				gameResults["round2Result"] +
				gameResults["round3Result"] +
				gameResults["round4Result"] +
				gameResults["AllRoundResult"] +
				gameResults["AllRoundCoins"] +
				gameResults["ActiveRound"]))
		newGameSum = hex.EncodeToString(shaHash.Sum(nil))

		if newGameSum != gameSum {
			break
		} else { // TODO Create a Global Query Pipe for these query type
			time.Sleep(500 * time.Millisecond)
			game, _ = g.server.GameRepo.Get(
				context.Background(),
				fmt.Sprintf("game:p%d:p%d:%d", game.P1ID, game.P2ID, game.Id))
		}
	}
	tmpl, err := template.New("game").Parse(gameHTML)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "failed to render game (1)")
	}

	var buf bytes.Buffer
	err = tmpl.Execute(
		&buf,
		schemas.GameData{Game: game, Competitor: competitor, GameResults: gameResults, GameResultsSum: newGameSum})
	if err != nil {
		logrus.Error("render game page error: ", err)
		return c.JSON(http.StatusInternalServerError, "failed to render game (2)")
	}

	return c.HTMLBlob(http.StatusOK, buf.Bytes())
}

// Helper function to render notification in top of page
func showNotification(c echo.Context, text string) error {
	tmpl, err := template.New("notification").Parse(notificationHTML)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to render home")
	}

	c.Response().Header().Set("HX-Retarget", "#notification-container")
	c.Response().Header().Set("HX-Reswap", "innerHTML")
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, text)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, "Failed to render home")
	}

	return c.HTMLBlob(245, buf.Bytes())
}
