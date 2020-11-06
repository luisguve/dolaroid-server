package main

import (
	"fmt"
	"encoding/gob"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/luisguve/scs/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/luisguve/dolaroid-server/datastore"
	"github.com/luisguve/dolaroid-server/geocode"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessKey = "sess"
	locationKey = "location"
)

type routes struct {
	ds   *datastore.Datastore
	sess *scs.SessionManager
}

// Structure of message to be sent in JSON format when requesting "/".
type initMsg struct {
	IsLoggedIn bool `json:"isLoggedIn"`
	User       *datastore.User `json:"session,omitempty"`
	SendLocation bool `json:"sendLocation"`
}

func init() {
	gob.Register(datastore.User{})
	gob.Register(coords{})
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
	// The location of the user is unknown?
	if !(r.sess.Exists(c.Context(), locationKey)) {
		res.SendLocation = true
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

	res := initMsg{
		IsLoggedIn: true,
		User: &user,
	}
	// The location of the user is unknown?
	if !(r.sess.Exists(c.Context(), locationKey)) {
		res.SendLocation = true
	}

	return c.JSON(res)
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

	res := initMsg{
		IsLoggedIn: true,
		User: &user,
	}
	// The location of the user is unknown?
	if !(r.sess.Exists(c.Context(), locationKey)) {
		res.SendLocation = true
	}

	return c.JSON(res)
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

type coords struct {
	Latitude string `json:"latt"`
	Longitude string `json:"longt"`
	City string `json:"city"`
	Region string `json:"region"`
	Country string `json:"country"`
}

func (r routes) handleLocation(c *fiber.Ctx) error {
	location := coords{}
	if err := c.BodyParser(&location); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	if (location.Latitude == "") || (location.Longitude == "") {
		log.Println("Received empty location")
		return c.SendStatus(http.StatusBadRequest)
	}
	query := fmt.Sprintf("%s,%s", location.Latitude, location.Longitude)
	// Get info about the city.
	if err := geocode.Locate(query, &location); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	query = fmt.Sprintf("%s,%s,%s", location.City, location.Region, location.Country)
	// Get coords of the city
	if err := geocode.Locate(query, &location); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}

	r.sess.Put(c.Context(), locationKey, location)
	return c.SendStatus(http.StatusOK)
}

// Reviews for authenticated users; full data.
type fullReview struct {
	BillInfo        datastore.BillInfo `json:"billInfo"`
	UserReviews     datastore.Reviews `json:"userReviews"`
	BusinessReviews datastore.Reviews `json:"businessReviews"`
	Defects         []string `json:"defects"`
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
	// Store serial number in variable.
	originalSerial := billInfo.SerialNumber
	// Remove whitespaces from serial number.
	billInfo.SerialNumber = strings.Replace(billInfo.SerialNumber, " ", "", -1)
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
		billInfo.SerialNumber = originalSerial
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
	// Assign back the serial number with spaces.
	billInfo.SerialNumber = originalSerial
	res := fullReview{
		BillInfo:        billInfo,
		UserReviews:     reviews.UserReviews,
		BusinessReviews: reviews.BusinessReviews,
		Defects:         reviews.Defects,
		AvgRating:       reviews.AvgRating,
		Details:         details,
		GoodReviews:     good,
		BadReviews:      bad,
	}
	return c.JSON(res)
}

func (r routes) handlePostReview(c *fiber.Ctx) error {
	// Isn't the user logged in?
	if !(r.sess.Exists(c.Context(), sessKey)) {
		return c.SendStatus(http.StatusUnauthorized)
	}
	// The location of the user is unknown?
	if !(r.sess.Exists(c.Context(), locationKey)) {
		c.SendString("Location required")
		return c.SendStatus(http.StatusBadRequest)
	}
	sessVal := r.sess.Get(c.Context(), sessKey)
	session, ok := sessVal.(datastore.User)
	if !ok {
		log.Printf("Failed type assertion to datastore.User from %t.\n", sessVal)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	locationVal := r.sess.Get(c.Context(), locationKey)
	location, ok := locationVal.(string)
	if !ok {
		log.Printf("Failed type assertion to string from %#v.\n", locationVal)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}

	review := datastore.PostReview{}
	if err := c.BodyParser(&review); err != nil {
		log.Println(err)
		c.SendString("Something went wrong")
		return c.SendStatus(http.StatusInternalServerError)
	}
	// Set user id from session data.
	review.Review.UserId = session.Id
	// Set type of account from session data.
	review.TypeOfAccount = session.Type
	// Set user location from session data.
	review.Review.Location = location
	// Remove whitespaces from serial number.
	review.BillInfo.SerialNumber = strings.Replace(review.BillInfo.SerialNumber, " ", "", -1)

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
