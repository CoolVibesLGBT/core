package geoip

import (
	"fmt"

	"github.com/oschwald/maxminddb-golang"
)

func Open() (*maxminddb.Reader, error) {
	paths := []string{
		"./static/data/GeoLite2-City.mmdb",
		"../static/data/GeoLite2-City.mmdb",
		"../../static/data/GeoLite2-City.mmdb",
	}
	var lastErr error
	for _, path := range paths {
		reader, err := maxminddb.Open(path)
		if err == nil {
			return reader, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("open GeoLite2-City.mmdb: %w", lastErr)
}
