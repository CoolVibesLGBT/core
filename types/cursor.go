package types

type Cursor struct {
	Prev     *string  `json:"prev,omitempty"`
	Next     *string  `json:"next,omitempty"`
	Distance *float64 `json:"distance,omitempty"`
}
