package datastore

import (
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	detailsDataB = "detailsData"
	IncomingDetail = "incoming"
	OutgoingDetail = "outgoing"
)

var (
	ErrNoDetails = fmt.Errorf("There are no details")
)

type Details struct {
	Date string `json:"date"`
	// from/to
	Involved string `json:"involved"`
	Subject string `json:"subject"`
	Notes string `json:"notes"`
}

type DetailsPair struct {
	In  *Details `json:"in"`
	Out *Details `json:"out"`
}

type PostDetails struct {
	Details `json:"detailsData"`
	// either incoming or outcoming
	TypeOfDetail string `json:"typeOfDetail"`
}

// Open database and prepare buckets.
func setupDetailsDB() (*bolt.DB, error) {
	details, err := bolt.Open("data/details.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("Could not open details DB: %v", err)
	}
	err = details.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(detailsDataB))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return details, err
}

func (d *Datastore) CreateDetails(userId string, billInfo BillInfo, pd PostDetails) error {
	id := fmt.Sprintf("%s-%s-%s-%s", userId, billInfo.SerialNumber, billInfo.Value, 
		billInfo.Series)
	return d.detailsDB.Update(func(tx *bolt.Tx) error {
		details := tx.Bucket([]byte(detailsDataB))
		billDetails := []DetailsPair{}
		billDetailsBytes := details.Get([]byte(id))
		if billDetailsBytes != nil {
			if err := json.Unmarshal(billDetailsBytes, &billDetails); err != nil {
				return err
			}
		}
		// Incoming
		if pd.TypeOfDetail == IncomingDetail {
			billDetails = append(billDetails, DetailsPair{
				In: &pd.Details,
			})
		} else {
			// Outgoing
			last := len(billDetails) - 1
			// Basically, if there are no details or if the last detail pair already has
			// an outcoming detail, then create a new detail pair.
			if (len(billDetails) == 0) || (billDetails[last].Out != nil) {
				billDetails = append(billDetails, DetailsPair{
					Out: &pd.Details,
				})
			} else {
				billDetails[last].Out = &pd.Details
			}
		}
		billDetailsBytes, err := json.Marshal(billDetails)
		if err != nil {
			return err
		}
		return details.Put([]byte(id), billDetailsBytes)
	})
}

// Get details
func (d *Datastore) QueryDetails(userId string, billInfo BillInfo) ([]DetailsPair, error) {
	id := fmt.Sprintf("%s-%s-%s-%s", userId, billInfo.SerialNumber, billInfo.Value, 
		billInfo.Series)
	var billDetails []DetailsPair
	err := d.detailsDB.View(func(tx *bolt.Tx) error {
		details := tx.Bucket([]byte(detailsDataB))
		billDetailsBytes := details.Get([]byte(id))
		if billDetailsBytes == nil {
			return ErrNoDetails
		}
		return json.Unmarshal(billDetailsBytes, &billDetails)
	})
	return billDetails, err
}
