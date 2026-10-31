package repositories

import (
	"core/models/post"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func postTreeVisibility(db *gorm.DB, includeUnpublished bool) *gorm.DB {
	if includeUnpublished {
		return db
	}
	return db.Where("published = ?", true).
		Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
		Where(repositorySubquery(db).Where(map[string]any{"audience": nil}).Or(map[string]any{"audience": ""}).Or(map[string]any{"audience": "public"}))
}

// postTreeIDs walks descendants breadth-first in bounded GORM queries
// instead of recursive SQL. The visited set is standard level-order
// book-keeping, not a guard against bad data: a post's parent is always an
// already-existing post at write time and parent_id is never rewritten
// after creation, so the tree cannot contain a cycle.
func postTreeIDs(db, rootQuery *gorm.DB, includeUnpublished bool) ([]uuid.UUID, error) {
	var roots []uuid.UUID
	if err := postTreeVisibility(rootQuery, includeUnpublished).Pluck("id", &roots).Error; err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(roots))
	visited := make(map[uuid.UUID]bool)
	add := func(candidates []uuid.UUID) []uuid.UUID {
		next := make([]uuid.UUID, 0, len(candidates))
		for _, id := range candidates {
			if !visited[id] {
				visited[id] = true
				ids = append(ids, id)
				next = append(next, id)
			}
		}
		return next
	}
	parents := add(roots)
	for len(parents) > 0 {
		var next []uuid.UUID
		for start := 0; start < len(parents); start += 200 {
			end := min(start+200, len(parents))
			var children []uuid.UUID
			q := postTreeVisibility(db.Model(&post.Post{}), includeUnpublished).Where("parent_id IN ?", parents[start:end])
			if err := q.Pluck("id", &children).Error; err != nil {
				return nil, err
			}
			next = append(next, add(children)...)
		}
		parents = next
	}
	return ids, nil
}
