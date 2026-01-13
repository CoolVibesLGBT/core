package types

type Cursor struct {
	Prev *string `json:"prev,omitempty"`
	Next *string `json:"next,omitempty"`
}
