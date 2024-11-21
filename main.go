package main

import (
	"embed"
	"log"

	server "github.com/onionj/trust/app"
	"github.com/onionj/trust/app/routes"
	"github.com/onionj/trust/config"
)

//go:embed assets
var embeddedFiles embed.FS

func main() {
	cfg := config.NewConfig(".env")
	app := server.NewServer(cfg)
	routes.MountWebRoutes(app, embeddedFiles)
	routes.MountTelegramRoutes(app)

	err := app.Start()
	if err != nil {
		log.Fatal(err)
	}
}
