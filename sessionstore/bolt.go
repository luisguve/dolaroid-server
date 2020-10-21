package sessionstore

import (
	"time"

	bolt "go.etcd.io/bbolt"
)

func New() (*bolt.DB, error) {
	return bolt.Open("data/sess.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
}