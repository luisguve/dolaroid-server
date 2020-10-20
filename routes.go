package main

import (
	"log"
	"errors"
	"net/http"
	"encoding/gob"

	"github.com/luisguve/scs/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/luisguve/dolaroid-server/datastore"
	"golang.org/x/crypto/bcrypt"
)

const sessKey = "sess"

type routes struct {
	ds   *datastore.Datastore
	sess *scs.SessionManager
}

type sess struct {
	Username string `json:"username"`
	UserId   string `json:"userId"`
	Type     string `json:"typeOfAccount"`
}

// Structure of message to be sent in JSON format when requesting "/".
type initMsg struct {
	IsLoggedIn bool `json:"isLoggedIn"`
	Session    *sess `json:"session,omitempty"`
}

func init() {
	gob.Register(sess{})
}

// Check whether the user is logged in and return its data: username, user id and
// type of account in JSON format. 
func (r routes) handleIndex(c *fiber.Ctx) error {
	res := initMsg{}
	// Isn't the user logged in?
	if !(r.sess.Exists(c.Context(), sessKey)) {
		return c.JSON(res)
	}
	sessVal := r.sess.Get(c.Context(), sessKey)
	session, ok := sessVal.(sess)
	if !ok {
		log.Printf("Failed type assertion to sess from %t.\n", sessVal)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	res = initMsg{
		IsLoggedIn: true,
		Session:    &session,
	}
	return c.JSON(res)
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
	id, err := r.ds.CreateUser(user)
	if err != nil {
		if errors.Is(err, datastore.ErrUsernameAlreadyTaken) {
			c.SendString("Username "+ user.Username +" already taken")
			return c.SendStatus(http.StatusConflict)
		}
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}

	r.sess.Put(c.Context(), sessKey, sess{
		Username: user.Username,
		UserId:   id,
		Type:     datastore.TypeRegular,
	})
	c.SendString(id)
	return c.SendStatus(http.StatusCreated)
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
		c.SendString("Invalid username or password")
		return c.SendStatus(http.StatusUnauthorized)
	}

	r.sess.Put(c.Context(), sessKey, sess{
		Username: user.Username,
		UserId:   user.Id,
		Type:     user.Type,
	})

	c.SendString(user.Id)
	return c.SendStatus(http.StatusOK)
}

func (r routes) handleLogout(c *fiber.Ctx) error {
	if r.sess.Exists(c.Context(), sessKey) {
		sessVal := r.sess.Get(c.Context(), sessKey)
		if _, ok := sessVal.(sess); !ok {
			log.Printf("Failed type assertion to sess from %t.\n", sessVal)
			c.SendString("User not logged in")
			return c.SendStatus(http.StatusUnauthorized)
		}
		r.sess.Remove(c.Context(), sessKey)
	}
	return c.SendStatus(http.StatusOK)
}
