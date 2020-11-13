package main

import (
	"log"
	"time"

	"github.com/luisguve/scs/v2"
	"github.com/alexedwards/scs/boltstore"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/luisguve/dolaroid-server/datastore"
	"github.com/luisguve/dolaroid-server/sessionstore"
)

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

	session := scs.New()
	session.Store = boltstore.NewWithCleanupInterval(sessStore, 3 * time.Minute)
	session.Lifetime = 24 * time.Hour
	r := routes{
		ds: data,
		sess: session,
	}

	app := fiber.New()

	// Wrap handlers with session middleware.
	app.Use("/", func(c *fiber.Ctx) error {
		return r.LoadAndSave(c)
	}, cors.New(cors.Config{
		AllowOrigins: "http://localhost:5000",
		AllowCredentials: true,
	}))

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

	// Start server on http://localhost:80
	app.Listen(":80")
}
