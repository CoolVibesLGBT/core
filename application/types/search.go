package types

// GlobalSearchResult groups the public read models used by the search screen.
type GlobalSearchResult struct {
	Query  string              `json:"query"`
	Users  []PublicUserSummary `json:"users"`
	Events []PublicPost        `json:"events"`
	Posts  []PublicPost        `json:"posts"`
	Places []PublicPost        `json:"places"`
}
