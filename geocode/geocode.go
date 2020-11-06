package geocode

import (
	"encoding/json"
	"net/http"
	"net/url"
)

const (
	geocodeToken = "459526325677843455651x14755"
)

func Locate(location string, v interface{}) error {
	baseURL, _ := url.Parse("https://geocode.xyz")

	baseURL.Path += "/"

	params := url.Values{}
	params.Add("auth", geocodeToken)
	// Query
	params.Add("location", location)
	params.Add("json", "1")

	baseURL.RawQuery = params.Encode()

	req, _ := http.NewRequest("GET", baseURL.String(), nil)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(&v)
}
