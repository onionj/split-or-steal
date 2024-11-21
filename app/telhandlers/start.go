package telhandlers

import (
	tele "gopkg.in/telebot.v4"

	"github.com/onionj/trust/app"
)

type StartHandlers struct {
	server *app.Server
}

func NewStartHandlers(server *app.Server) *StartHandlers {
	return &StartHandlers{server: server}
}

func (s *StartHandlers) Start(c tele.Context) error {
	selector := &tele.ReplyMarkup{}
	selector.Inline(selector.Row(selector.WebApp(
		"🎮 Open App",
		&tele.WebApp{URL: s.server.Config.HTTP.ExposeAddress},
	)))
	return c.Send("Open App:", selector)
}
