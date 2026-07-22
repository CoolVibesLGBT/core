package types

type PageResult[T any] struct {
	Data  []T   `json:"data"`
	Meta  Meta  `json:"meta"`
	Links Links `json:"links"`
}

type Meta struct {
	CurrentPage int   `json:"current_page,omitempty"`
	PerPage     int   `json:"per_page,omitempty"`
	Total       int   `json:"total,omitempty"`
	LastPage    int   `json:"last_page,omitempty"`
	PrevPage    int   `json:"prev_page,omitempty"`
	NextPage    int   `json:"next_page,omitempty"`
	PageNumbers []int `json:"page_numbers,omitempty"`

	// cursor mode
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

type Links struct {
	Next string `json:"next,omitempty"`
	Prev string `json:"prev,omitempty"`
}

type PaginationQuery struct {
	Page    int
	PerPage int
	Cursor  string
	Mode    string // "offset" | "cursor"
}

/*
func ParsePagination(c *fiber.Ctx) PaginationQuery {
	c.
	page, _ := strconv.Atoi(c("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	if perPage > 100 {
		perPage = 100
	}

	return PaginationQuery{
		Page:    max(page, 1),
		PerPage: perPage,
		Cursor:  c.Query("cursor"),
		Mode:    c.Query("mode", "offset"),
	}
}
*/
