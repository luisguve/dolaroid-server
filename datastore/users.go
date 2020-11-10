package datastore

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/satori/go.uuid"
	bolt "go.etcd.io/bbolt"
)

const (
	usersDataB = "usersData"
	usernameToIdMappingsB = "usernameToId"
	TypeAdmin = "admin"
	TypeRegular = "regular"
	TypeBusiness = "business"
)

var (
	ErrUsernameAlreadyTaken = fmt.Errorf("Username already taken")
	ErrUsernameNotExists = fmt.Errorf("Username not found")
	ErrUnknownTypeOfAccount = fmt.Errorf("Invalid type of account")
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password,omitempty"`
	Type     string `json:"typeOfAccount"`// type of account - admin, regular or business.
	Id       string `json:"userId"`
}

// Open database and prepare buckets.
func setupUsersDB() (*bolt.DB, error) {
	users, err := bolt.Open("data/users.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("Could not open users DB: %v", err)
	}
	err = users.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(usersDataB))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		_, err = tx.CreateBucketIfNotExists([]byte(usernameToIdMappingsB))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return users, err
}

func (d *Datastore) CreateUser(user User) (string, error) {
	var id string
	err := d.usersDB.Update(func(tx *bolt.Tx) error {
		usernames := tx.Bucket([]byte(usernameToIdMappingsB))
		userExists := usernames.Get([]byte(user.Username))
		if userExists != nil {
			return ErrUsernameAlreadyTaken
		}
		id = uuid.NewV4().String()
		err := usernames.Put([]byte(user.Username), []byte(id))
		if err != nil {
			return err
		}
		user.Id = id
		buf, err := json.Marshal(user)
		if err != nil {
			return err
		}
		usersData := tx.Bucket([]byte(usersDataB))
		return usersData.Put([]byte(id), buf)
	})
	return id, err
}

// Get a user.
func (d *Datastore) User(username string) (User, error) {
	var user User
	err := d.usersDB.View(func(tx *bolt.Tx) error {
		usernames := tx.Bucket([]byte(usernameToIdMappingsB))
		id := usernames.Get([]byte(username))
		if id == nil {
			return ErrUsernameNotExists
		}
		usersData := tx.Bucket([]byte(usersDataB))
		buf := usersData.Get(id)
		return json.Unmarshal(buf, &user)
	})
	return user, err
}
