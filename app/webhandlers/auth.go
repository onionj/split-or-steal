package webhandlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/labstack/echo/v4"
	"github.com/onionj/trust/app"
	"github.com/sirupsen/logrus"
)

type authHandlers struct {
	server *app.Server
}

func NewAuthHandlers(server *app.Server) *authHandlers {
	return &authHandlers{server: server}
}

func (a authHandlers) AuthorizeMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		initData := c.Request().Header.Get("Authorization")

		isValid, err := app.ValidateWebAppInputData(initData)
		if err != nil {
			return err
		}
		if !isValid {
			return showNotification(c, "Invalid InitData")
		}
		parsed, _ := url.ParseQuery(initData)

		user_data := make(map[string]interface{})
		json.Unmarshal([]byte(parsed.Get("user")), &user_data)

		user_id := int64(user_data["id"].(float64))
		// user_id := int64(790311667)

		user, err := a.server.UserRepo.Get(
			context.Background(),
			fmt.Sprintf("user:%d", user_id))

		if err == nil {
			c.Set("user", user)
		} else {
			logrus.Error("get data from user repo err: ", err)
			return c.String(244, "error")
		}

		return next(c)

	}
}
