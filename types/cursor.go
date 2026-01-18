package types

type Cursor struct {
	Prev *string `json:"prev,omitempty"`
	Next *string `json:"next,omitempty"`
}

type Filters struct {
	Search    *string
	Category  *string
	Name      *string
	City      *string
	Country   *string
	Latitude  *float64
	Longitude *float64
	Cursor    *int64
	Limit     int
}
