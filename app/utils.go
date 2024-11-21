package app

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/onionj/trust/config"
	"github.com/onionj/trust/internal/entity"

	"github.com/sirupsen/logrus"
)

type J map[string]any

func ValidateWebAppInputData(inputData string) (bool, error) {
	initData, err := url.ParseQuery(inputData)
	if err != nil {
		logrus.WithError(err).Errorln("couldn't parse web app input data")
		return false, err
	}

	dataCheckString := make([]string, 0, len(initData))
	for k, v := range initData {
		if k == "hash" {
			continue
		}
		if len(v) > 0 {
			dataCheckString = append(dataCheckString, fmt.Sprintf("%s=%s", k, v[0]))
		}
	}

	sort.Strings(dataCheckString)

	secret := hmac.New(sha256.New, []byte("WebAppData"))
	secret.Write([]byte(config.GlobalConfig.Telegram.Token))

	hHash := hmac.New(sha256.New, secret.Sum(nil))
	hHash.Write([]byte(strings.Join(dataCheckString, "\n")))

	hash := hex.EncodeToString(hHash.Sum(nil))
	if initData.Get("hash") != hash {
		return false, nil
	}

	return true, nil
}

func ResponseOk(code int, data any) any {
	return J{
		"ok":   true,
		"code": code,
		"data": data,
	}
}
func ResponseError(code int, data any) any {
	return J{
		"ok":   false,
		"code": code,
		"data": data,
	}
}

func GetUserFromCtx(c echo.Context) entity.User {
	return c.Get("user").(entity.User)
}
