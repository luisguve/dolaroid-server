package main

import (
	"fmt"
	"log"
	"time"
	"strings"

	"github.com/luisguve/scs/v2"
	"github.com/BurntSushi/toml"
	"github.com/alexedwards/scs/boltstore"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/luisguve/dolaroid-server/datastore"
	"github.com/luisguve/dolaroid-server/sessionstore"
)

type srvConfig struct {
	AllowOrigins []string `toml:"allow_origins"`
	Production   bool     `toml:"production"`
	Domain       string   `toml:"domain"`
	Email        string   `toml:"email"`
	SessionLT    int      `toml:"session_lifetime"`
}

func (s srvConfig) validate() error {
	if len(s.AllowOrigins) == 0 {
		return fmt.Errorf("empty origins list")
	}
	return nil
}

func main() {
	data, err := datastore.New()
	if err != nil {
		log.Fatal(err)
	}
	defer data.Close()

	sessStore, err := sessionstore.New()
	if err != nil {
		log.Fatal(err)
	}
	defer sessStore.Close()

	var config srvConfig
	if _, err = toml.DecodeFile("config.toml", &config); err != nil {
		log.Fatal(err)
	}
	if err = config.validate(); err != nil {
		log.Fatal(err)
	}

	session := scs.New()
	session.Store = boltstore.NewWithCleanupInterval(sessStore, 30 * time.Minute)
	session.Lifetime = time.Duration(config.SessionLT) * time.Hour
	r := routes{
		ds: data,
		sess: session,
	}

	app := fiber.New()

	// Wrap handlers with session middleware.
	app.Use("/", func(c *fiber.Ctx) error {
			return r.LoadAndSave(c)
		}, cors.New(cors.Config{
			AllowOrigins: strings.Join(config.AllowOrigins, ","),
			AllowCredentials: true,
		}),
	)

	app.Get("/", r.handleIndex)

	app.Post("/login", r.handleLogin)
	app.Post("/signup", r.handleSignup)
	app.Post("/logout", r.handleLogout)
	app.Post("/location", r.handleLocation)

	app.Get("/review", r.handleGetReview)
	app.Post("/review", r.handlePostReview)

	// Last middleware to match anything
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	// Start server on port 80 on localhost.
	log.Fatal(app.Listen(":8000"))
}
