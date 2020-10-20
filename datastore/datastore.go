package datastore

import (
	"encoding/json"
	"time"
	"fmt"

	bolt "go.etcd.io/bbolt"
	"github.com/satori/go.uuid"
)

const (
	usersDataB = "usersData"
	usernameToIdMappingsB = "usernameToId"
)

var (
	ErrUsernameAlreadyTaken = fmt.Errorf("Username already taken")
	ErrUsernameNotExists = fmt.Errorf("Username not found")
)

type Datastore struct {
	usersDB *bolt.DB
}

func New() (*Datastore, error) {
	users, err := bolt.Open("users.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
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
	if err != nil {
		return nil, err
	}
	return &Datastore{
		usersDB: users,
	}, nil
}

func (d *Datastore) Close() error {
	return d.usersDB.Close()
}

const (
	TypeAdmin = "admin"
	TypeRegular = "regular"
	TypeBusiness = "business"
)

type User struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Type     string // type of account - admin, regular or business.
	Id       string
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
