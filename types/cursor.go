package types

type Cursor struct {
	Prev *string `json:"prev,omitempty"`
	Next *string `json:"next,omitempty"`
}

type PlaceFilters struct {
	Category  *string
	Name      *string
	City      *string
	Country   *string
	Latitude  *float64
	Longitude *float64
	Cursor    *int64
	Limit     int
}
