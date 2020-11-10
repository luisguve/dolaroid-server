package datastore

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	GoodReview = "Good review"
	BadReview = "Bad review"
	reviewsDataB = "reviewsData"
)

var (
	ErrNoReviews = fmt.Errorf("There are no reviews")
	ErrReviewExists = fmt.Errorf("This user already posted a review of this type")
	ErrUnknownTypeOfReview = fmt.Errorf("Invalid type of review")
)

type BillInfo struct {
	SerialNumber string `query:"sn"     json:"serialNumber"`
	Value        string `query:"value"  json:"value"`
	Series       string `query:"series" json:"series"`
}

type Coords struct {
	Latitude string `json:"latt"`
	Longitude string `json:"longt"`
	City string `json:"city"`
	Region string `json:"region"`
	Country string `json:"country"`
}

type Review struct {
	UserId string `json:"userId"`
	Location Coords `json:"location"`
	Date string `json:"date"`
	Comment string `json:"comment"`
	Defects []string `json:"defects"` // in case of a bad review.
	Rating int `json:"rating"` // in case of a good review.
}

type PostReview struct {
	BillInfo    `json:"billInfo"`
	Review      `json:"review"`
	*PostDetails `json:"details,omitempty"`
	// Regular user or business?
	TypeOfAccount string `json:"typeOfAccount"`
	// Good review or bad review?
	TypeOfReview string `json:"typeOfReview"`
}

type Reviews struct {
	GoodReviews []Review `json:"goodReviews"`
	BadReviews []Review `json:"badReviews"`
}

type GetReviews struct {
	BillInfo `json:"billInfo"`
	UserReviews Reviews `json:"userReviews"`
	BusinessReviews Reviews `json:"businessReviews"`
	Defects []string `json:"defects"`
	Ratings int `json:"ratings"`
	AvgRating int `json:"avgRating"`
}

// Open database and prepare buckets.
func setupReviewsDB() (*bolt.DB, error) {
	reviews, err := bolt.Open("data/reviews.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("Could not open reviews DB: %v", err)
	}
	err = reviews.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(reviewsDataB))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return reviews, err
}

func (d *Datastore) CreateReview(review PostReview) error {
	id := fmt.Sprintf("%s-%s-%s", review.BillInfo.SerialNumber, review.BillInfo.Value,
		review.BillInfo.Series)
	return d.reviewsDB.Update(func(tx *bolt.Tx) error {
		reviews := tx.Bucket([]byte(reviewsDataB))
		bill := GetReviews{}
		billBytes := reviews.Get([]byte(id))
		if billBytes != nil {
			if err := json.Unmarshal(billBytes, &bill); err != nil {
				return err
			}
		} else {
			bill.BillInfo = review.BillInfo
		}
		switch review.TypeOfAccount {
		case TypeRegular, TypeAdmin:
			switch review.TypeOfReview {
			case GoodReview:
				// Allow only 1 review per user
				for _, rv := range bill.UserReviews.GoodReviews {
					if review.UserId == rv.UserId {
						return ErrReviewExists
					}
				}
				// Update ratings and avg rating
				totalReviews := len(bill.UserReviews.GoodReviews)
				totalReviews += len(bill.BusinessReviews.GoodReviews)
				totalReviews++ // Add this review.
				bill.Ratings += review.Review.Rating
				bill.AvgRating = bill.Ratings / totalReviews
				// Append review to this bill
				goodReviews := append(bill.UserReviews.GoodReviews, review.Review)
				bill.UserReviews.GoodReviews = goodReviews
			case BadReview:
				// Allow only 1 review per user
				for _, rv := range bill.UserReviews.BadReviews {
					if review.UserId == rv.UserId {
						return ErrReviewExists
					}
				}

				badReviews := append(bill.UserReviews.BadReviews, review.Review)
				bill.UserReviews.BadReviews = badReviews
				// Update defects; if the defect's not there, append it.
				for _, label1 := range review.Review.Defects {
					found := false
					for _, label2 := range bill.Defects {
						if label1 == label2 {
							found = true
							break
						}
					}
					if !found {
						bill.Defects = append(bill.Defects, label1)
					}
				}
			default:
				log.Printf("%s: %s\n", ErrUnknownTypeOfReview, review.TypeOfReview)
				return ErrUnknownTypeOfReview
			}
		case TypeBusiness:
			switch review.TypeOfReview {
			case GoodReview:
				// Allow only 1 review per user
				for _, rv := range bill.BusinessReviews.GoodReviews {
					if review.UserId == rv.UserId {
						return ErrReviewExists
					}
				}
				// Update ratings and avg rating
				totalReviews := len(bill.UserReviews.GoodReviews)
				totalReviews += len(bill.BusinessReviews.GoodReviews)
				totalReviews++ // Add this review.
				bill.Ratings += review.Review.Rating
				bill.AvgRating = bill.Ratings / totalReviews
				// Append this review to this bill
				goodReviews := append(bill.BusinessReviews.GoodReviews, review.Review)
				bill.BusinessReviews.GoodReviews = goodReviews
			case BadReview:
				// Allow only 1 review per user
				for _, rv := range bill.BusinessReviews.BadReviews {
					if review.UserId == rv.UserId {
						return ErrReviewExists
					}
				}

				badReviews := append(bill.BusinessReviews.BadReviews, review.Review)
				bill.BusinessReviews.BadReviews = badReviews
				// Update defects; if the defect's not there, append it.
				for _, label1 := range review.Review.Defects {
					found := false
					for _, label2 := range bill.Defects {
						if label1 == label2 {
							found = true
							break
						}
					}
					if !found {
						bill.Defects = append(bill.Defects, label1)
					}
				}
			default:
				log.Printf("%s: %s\n", ErrUnknownTypeOfReview, review.TypeOfReview)
				return ErrUnknownTypeOfReview
			}
		default:
			log.Printf("%s: %s\n", ErrUnknownTypeOfAccount, review.TypeOfAccount)
			return ErrUnknownTypeOfAccount
		}
		billBytes, err := json.Marshal(bill)
		if err != nil {
			return err
		}
		return reviews.Put([]byte(id), billBytes)
	})
}

// Get review.
func (d *Datastore) QueryReviews(billInfo BillInfo) (GetReviews, error) {
	id := fmt.Sprintf("%s-%s-%s", billInfo.SerialNumber, billInfo.Value, billInfo.Series)
	bill := GetReviews{}
	err := d.reviewsDB.View(func(tx *bolt.Tx) error {
		reviews := tx.Bucket([]byte(reviewsDataB))
		billBytes := reviews.Get([]byte(id))
		if billBytes == nil {
			return ErrNoReviews
		}
		return json.Unmarshal(billBytes, &bill)
	})
	return bill, err
}
