package types

type ReactionType string

const (
	ReactionLike      ReactionType = "like"
	ReactionDislike   ReactionType = "dislike"
	ReactionFavorite  ReactionType = "favorite"
	ReactionBookmark  ReactionType = "bookmark"
	ReactionMatched   ReactionType = "matched"
	ReactionSuperLike ReactionType = "superlike"
)
