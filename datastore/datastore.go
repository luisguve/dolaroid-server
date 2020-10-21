package datastore

import (
	bolt "go.etcd.io/bbolt"
)

type Datastore struct {
	usersDB   *bolt.DB
	reviewsDB *bolt.DB
	detailsDB *bolt.DB
}

// Open databases and prepare buckets.
func New() (*Datastore, error) {
	users, err := setupUsersDB()
	if err != nil {
		return nil, err
	}
	reviews, err := setupReviewsDB()
	if err != nil {
		return nil, err
	}
	details, err := setupDetailsDB()
	if err != nil {
		return nil, err
	}
	
	return &Datastore{
		usersDB:   users,
		reviewsDB: reviews,
		detailsDB: details,
	}, nil
}

// Release resources.
func (d *Datastore) Close() error {
	_ = d.usersDB.Close()
	_ = d.reviewsDB.Close()
	_ = d.detailsDB.Close()
	return nil
}
