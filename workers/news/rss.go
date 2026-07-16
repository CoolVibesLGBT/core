package news

import (
	"context"
	"time"

	"github.com/mmcdole/gofeed"
)

type FeedItem struct {
	Title       string
	Link        string
	Description string
	Published   *time.Time
}

func FetchRSS(ctx context.Context, url string) ([]FeedItem, error) {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, err
	}

	var items []FeedItem
	for _, item := range feed.Items {
		items = append(items, FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			Published:   item.PublishedParsed,
		})
	}
	return items, nil
}
