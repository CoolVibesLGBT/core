package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Address struct {
	Road        string
	Suburb      string
	District    string
	Town        string
	Province    string
	Region      string
	Postcode    string
	Country     string
	CountryCode string
}

func ReverseGeocode(lat, lon float64) (Address, error) {
	address := Address{}

	url := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?lat=%f&lon=%f&format=json",
		lat, lon,
	)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return address, err
	}

	req.Header.Set("User-Agent", "CoolVibes/1.0 (info@coolvibes.lgbt)")

	resp, err := client.Do(req)
	if err != nil {
		return address, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return address, err
	}

	addr, ok := result["address"].(map[string]interface{})
	if !ok {
		return address, nil
	}

	if v, ok := addr["road"].(string); ok {
		address.Road = v
	}
	if v, ok := addr["suburb"].(string); ok {
		address.Suburb = v
	}
	if v, ok := addr["city_district"].(string); ok {
		address.District = v
	}
	if v, ok := addr["town"].(string); ok {
		address.Town = v
	}
	if address.Town == "" {
		if v, ok := addr["city"].(string); ok {
			address.Town = v
		}
	}
	if v, ok := addr["province"].(string); ok {
		address.Province = v
	}
	if v, ok := addr["region"].(string); ok {
		address.Region = v
	}
	if v, ok := addr["postcode"].(string); ok {
		address.Postcode = v
	}
	if v, ok := addr["country"].(string); ok {
		address.Country = v
	}
	if v, ok := addr["country_code"].(string); ok {
		address.CountryCode = v
	}

	return address, nil
}
