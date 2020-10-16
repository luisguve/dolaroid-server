package main

import (
	"errors"
	"log"
	"time"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html"
	"github.com/luisguve/dolaroid-server/datastore"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Name string `json:"username" form:"username"`
	Pass string `json:"password" form:"password"`
}

func handleIndex(c *fiber.Ctx) error {
	username := c.Cookies("session")
	if username == "" {
		return c.SendFile("./static/index.html")
	}
	return c.Render("index-user", fiber.Map{
		"Name": username,
	})
}

type routes struct {
	ds *datastore.Datastore
}

func (r routes) handleSignup(c *fiber.Ctx) error {
	user := datastore.User{}
	if err := c.BodyParser(&user); err != nil {
		return err
	}
	pass, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	user = datastore.User{
		Username: user.Username,
		Password: string(pass),
		Type: datastore.TypeRegular,
	}
	_, err = r.ds.CreateUser(user)
	if err != nil {
		if errors.Is(err, datastore.ErrUsernameAlreadyTaken) {
			c.SendString("Username "+ user.Username +" already taken")
			return c.SendStatus(http.StatusOK)
		}
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	// find a way to integrate gorilla cookies with fiber; the user id should be
	// the value of the cookie, but the username also has to be stored in the
	// session.
	cookie := &fiber.Cookie{
		Name: "session",
		Value: user.Username,
		Expires: time.Now().Add(24 * time.Hour),
	}

	// Set cookie
	c.Cookie(cookie)
	return c.SendStatus(http.StatusOK)
}

func (r routes) handleLogin(c *fiber.Ctx) error {
	user := datastore.User{}
	if err := c.BodyParser(&user); err != nil {
		return err
	}
	if user.Username == "" || user.Password == "" {
		return c.SendStatus(http.StatusBadRequest)
	}
	pass := user.Password
	user, err := r.ds.User(user.Username)
	if err != nil {
		if errors.Is(err, datastore.ErrUsernameNotExists) {
			c.SendString("User unregistered")
			return c.SendStatus(http.StatusUnauthorized)
		}
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(pass))
	if err != nil {
		c.SendString("Invalid credentials")
		return c.SendStatus(http.StatusUnauthorized)
	}

	cookie := &fiber.Cookie{
		Name: "session",
		Value: user.Username,
		Expires: time.Now().Add(24 * time.Hour),
	}

	// Set cookie
	c.Cookie(cookie)
	return c.SendStatus(http.StatusOK)
}

func (r routes) handleLogout(c *fiber.Ctx) error {
	user := c.Cookies("session")
	if user == "" {
		c.SendString("User not logged in")
		return c.SendStatus(http.StatusUnauthorized)
	}
	cookie := &fiber.Cookie{
		Name: "session",
		Expires: time.Now().Add(-(2 * time.Hour)),
	}
	c.Cookie(cookie)
	return c.SendStatus(http.StatusOK)
}

func main() {
	var (
		r routes
		err error
	)
	r.ds, err = datastore.New()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize standard Go html template engine
	engine := html.New("./static", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/", handleIndex)

	app.Post("/login", r.handleLogin)
	app.Post("/signup", r.handleSignup)
	app.Post("/logout", r.handleLogout)

	// Last middleware to match anything
	app.Use(func(c *fiber.Ctx) error {
		return c.SendStatus(404) // => 404 "Not Found"
	})

	// Start server on http://localhost:8000
	app.Listen(":8000")
}