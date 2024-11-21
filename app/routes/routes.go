package routes

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v4"

	"github.com/onionj/trust/app"
	"github.com/onionj/trust/app/telhandlers"
	"github.com/onionj/trust/app/webhandlers"
	"github.com/onionj/trust/internal/entity"
	"github.com/onionj/trust/internal/repository"
)

func MountWebRoutes(server *app.Server, embeddedFiles embed.FS) {
	gameHandler := webhandlers.NewGameHandlers(server)
	authHandler := webhandlers.NewAuthHandlers(server)

	server.Echo.Use(middleware.Recover())
	server.Echo.Use(middleware.GzipWithConfig(middleware.GzipConfig{}))
	server.Echo.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			stop := time.Now()

			method := c.Request().Method
			uri := c.Request().RequestURI
			code := c.Response().Status
			if err != nil {
				// Echo sets the status code based on the error
				// You can inspect the error to determine the status if needed
				if he, ok := err.(*echo.HTTPError); ok {
					code = he.Code
				} else {
					// Default to 500 if it's not an HTTPError
					code = http.StatusInternalServerError
				}
			}
			latency := stop.Sub(start)

			log := fmt.Sprintf("%s %-12v %s %-15v  %d",
				stop.Format(time.RFC3339),
				latency,
				method,
				uri,
				code,
			)
			if code < 400 {
				logrus.Info(log)
			} else {
				logrus.Error(log)
			}

			return err
		}
	})

	// Create a subdirectory view of the embedded assets
	staticFiles, err := fs.Sub(embeddedFiles, "assets")
	if err != nil {
		server.Echo.Logger.Fatal(err)
	}

	// Serve static files and favicon from embedded assets
	server.Echo.StaticFS("/static", staticFiles)
	server.Echo.FileFS("/favicon.ico", "assets/favicon.ico", embeddedFiles)

	game := server.Echo.Group("")
	game.GET("/", gameHandler.OpenHome)
	game.GET("/menu", gameHandler.OpenMenu, authHandler.AuthorizeMiddleware)
	game.GET("/game", gameHandler.StartGame, authHandler.AuthorizeMiddleware)
	game.GET("/game-update/:gameID", gameHandler.GetGameUpdate, authHandler.AuthorizeMiddleware)
	game.GET("/game-choice/:gameID/:roundID/:choice", gameHandler.GameChoice, authHandler.AuthorizeMiddleware)
}

// Telegram user count per year
// Year	Users (millions)
// 2014	35
// 2015	50
// 2016	80
// 2017	150
// 2018	200
// 2019	300
// 2020	400
// 2021	550
// 2022	700
// 2023	800
// 2024	+900
func coinPerAccountAge(userID int64) int {
	switch {
	case userID < 80_000_000: // 2016
		return 20_000
	case userID < 400_000_000: // 2020
		return 10_000
	case userID < 800_000_000: // 2023
		return 5_000
	case userID < 1_000_000_000: // 2024
		return 2_500
	default:
		return 900
	}
}

func MountTelegramRoutes(server *app.Server) {

	server.TeleBot.Use(func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {
			logrus.Info("tel: Username:", c.Sender().Username, " ID:", c.Sender().ID, " Text:", c.Text())

			user, err := server.UserRepo.Get(
				context.Background(),
				fmt.Sprintf("user:%d", c.Sender().ID))

			if err == nil {
				c.Set("user", user)

			} else if errors.Is(err, repository.ErrNotFound) {
				err := server.UserRepo.Save(context.Background(),
					entity.NewUser(c.Sender().ID,
						fmt.Sprintf("%s %s", c.Sender().FirstName, c.Sender().LastName),
						coinPerAccountAge(c.Sender().ID),
					))
				if err != nil {
					logrus.Error("save user err: ", err)
					return err
				}
				user, err := server.UserRepo.Get(
					context.Background(),
					fmt.Sprintf("user:%d", c.Sender().ID))

				if err != nil {
					logrus.Error("get user after save err: ", err)
					return err
				}
				c.Set("user", user)
				c.Reply(fmt.Sprintf("🎉 You Win %d Coins!", coinPerAccountAge(c.Sender().ID)))

			} else {
				logrus.Error("get data from user repo err: ", err)
				return err
			}

			return next(c)
		}
	})

	startHandlers := telhandlers.NewStartHandlers(server)
	server.TeleBot.Handle("/start", startHandlers.Start)
}
