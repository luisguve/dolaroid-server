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

// Structure of message to be sent in JSON format when requesting "/".
type initMsg struct {
	IsLoggedIn bool `json:"isLoggedIn"`
	User       *datastore.User `json:"session,omitempty"`
}

func init() {
	gob.Register(datastore.User{})
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
	session, ok := sessVal.(datastore.User)
	if !ok {
		log.Printf("Failed type assertion to datastore.User from %t.\n", sessVal)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	res = initMsg{
		IsLoggedIn: true,
		User:       &session,
	}
	return c.JSON(res)
}

func (r routes) handleSignup(c *fiber.Ctx) error {
	user := datastore.User{}
	if err := c.BodyParser(&user); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
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
	user.Id, err = r.ds.CreateUser(user)
	if err != nil {
		if errors.Is(err, datastore.ErrUsernameAlreadyTaken) {
			c.SendString("Username "+ user.Username +" already taken")
			return c.SendStatus(http.StatusConflict)
		}
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}

	// Don't store password in session
	user.Password = ""

	r.sess.Put(c.Context(), sessKey, user)
	c.SendString(user.Id)
	return c.SendStatus(http.StatusCreated)
}

func (r routes) handleLogin(c *fiber.Ctx) error {
	user := datastore.User{}
	if err := c.BodyParser(&user); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
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

	// Don't store password in session
	user.Password = ""

	r.sess.Put(c.Context(), sessKey, user)

	c.SendString(user.Id)
	return c.SendStatus(http.StatusOK)
}

func (r routes) handleLogout(c *fiber.Ctx) error {
	if r.sess.Exists(c.Context(), sessKey) {
		sessVal := r.sess.Get(c.Context(), sessKey)
		if _, ok := sessVal.(datastore.User); !ok {
			log.Printf("Failed type assertion to datastore.User from %t.\n", sessVal)
			c.SendString("User not logged in")
			return c.SendStatus(http.StatusUnauthorized)
		}
		r.sess.Remove(c.Context(), sessKey)
	}
	return c.SendStatus(http.StatusOK)
}

// Reviews for authenticated users; full data.
type fullReview struct {
	BillInfo        datastore.BillInfo `json:"billInfo"`
	UserReviews     datastore.Reviews `json:"userReviews"`
	BusinessReviews datastore.Reviews `json:"businessReviews"`
	Defects         map[string]string `json:"defects"`
	AvgRating       int `json:"avgRating"`
	Details         []datastore.DetailsPair `json:"details"`
	GoodReviews     int `json:"goodReviews"`
	BadReviews      int `json:"badReviews"`
}

// Reviews for unauthenticated users; very limited data.
type basicReview struct {
	BillInfo datastore.BillInfo `json:"billInfo"`
	GoodReviews int `json:"goodReviews"`
	BadReviews  int `json:"badReviews"`
}

func (r routes) handleGetReview(c *fiber.Ctx) error {
	billInfo := datastore.BillInfo{}
	if err := c.QueryParser(&billInfo); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	reviews, err := r.ds.QueryReviews(billInfo)
	if err != nil {
		if !errors.Is(err, datastore.ErrNoReviews) {
			log.Println(err)
			c.SendString("Something went wrong")
			return c.SendStatus(http.StatusInternalServerError)
		}
	}
	good := len(reviews.UserReviews.GoodReviews) + len(reviews.BusinessReviews.GoodReviews)
	bad := len(reviews.UserReviews.BadReviews) + len(reviews.BusinessReviews.BadReviews)
	// Isn't the user logged in?
	if !(r.sess.Exists(c.Context(), sessKey)) {
		return c.JSON(basicReview{
			BillInfo:    billInfo,
			GoodReviews: good,
			BadReviews:  bad,
		})
	}
	sessVal := r.sess.Get(c.Context(), sessKey)
	session, ok := sessVal.(datastore.User)
	if !ok {
		log.Printf("Failed type assertion to datastore.User from %t.\n", sessVal)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	details, err := r.ds.QueryDetails(session.Id, billInfo)
	if err != nil {
		if !errors.Is(err, datastore.ErrNoDetails) {
			log.Println(err)
			c.SendString("Something went wrong")
			return c.SendStatus(http.StatusInternalServerError)
		}
	}
	return c.JSON(fullReview{
		BillInfo:        billInfo,
		UserReviews:     reviews.UserReviews,
		BusinessReviews: reviews.BusinessReviews,
		Defects:         reviews.Defects,
		AvgRating:       reviews.AvgRating,
		Details:         details,
		GoodReviews:     good,
		BadReviews:      bad,
	})
}

func (r routes) handlePostReview(c *fiber.Ctx) error {
	// Isn't the user logged in?
	if !(r.sess.Exists(c.Context(), sessKey)) {
		return c.SendStatus(http.StatusUnauthorized)
	}
	sessVal := r.sess.Get(c.Context(), sessKey)
	session, ok := sessVal.(datastore.User)
	if !ok {
		log.Printf("Failed type assertion to datastore.User from %t.\n", sessVal)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}

	review := datastore.PostReview{}
	if err := c.BodyParser(&review); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	review.TypeOfAccount = session.Type

	if err := r.ds.CreateReview(review); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	if review.PostDetails != nil {
		if err := r.ds.CreateDetails(session.Id, review.BillInfo, *review.PostDetails); err != nil {
			log.Println(err)
			c.SendString("Something went wrong")
			return c.SendStatus(http.StatusInternalServerError)
		}
	}
	return c.SendStatus(http.StatusCreated)
}
