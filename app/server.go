package app

import (
	"fmt"
	"log"
	"time"

	"github.com/onionj/trust/config"
	"github.com/onionj/trust/db"
	"github.com/onionj/trust/internal/repository"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
	tele "gopkg.in/telebot.v4"
)

type Server struct {
	Echo     *echo.Echo
	TeleBot  *tele.Bot
	DB       *redis.Client
	Config   config.ConfigT
	UserRepo repository.UserRepository
	GameRepo repository.GameRepository
}

func NewServer(cfg config.ConfigT) *Server {

	bot, err := tele.NewBot(tele.Settings{
		Token:  cfg.Telegram.Token,
		Poller: &tele.LongPoller{Timeout: 20 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
	}
	redis := db.Init(cfg)

	userRepo := repository.NewUserRepository(redis)
	gameRepo := repository.NewGameRepository(redis)

	return &Server{
		Echo:     echo.New(),
		TeleBot:  bot,
		DB:       redis,
		Config:   cfg,
		UserRepo: userRepo,
		GameRepo: gameRepo,
	}
}

func (server *Server) Start() error {
	go server.TeleBot.Start()
	fmt.Println(server.Config.HTTP.Host + ":" + server.Config.HTTP.Port)
	return server.Echo.Start(server.Config.HTTP.Host + ":" + server.Config.HTTP.Port)
}
