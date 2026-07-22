package types

import (
	domainuser "core/domain/user"
	"time"

	"github.com/google/uuid"
)

// ReactionType remains as an API compatibility alias while the matching rule
// itself lives in the user domain.
type ReactionType = domainuser.MatchReaction

const (
	ReactionLike      ReactionType = domainuser.MatchReactionLike
	ReactionDislike   ReactionType = domainuser.MatchReactionDislike
	ReactionFavorite  ReactionType = "favorite"
	ReactionBookmark  ReactionType = "bookmark"
	ReactionMatched   ReactionType = "matched"
	ReactionSuperLike ReactionType = "superlike"
)

// MatchListCursor uses the engagement detail ordering key, not the target
// user's account creation date. The UUID is encoded in the opaque HTTP cursor
// only to make rows with identical timestamps paginate deterministically.
type MatchListCursor struct {
	OccurredAt time.Time
	DetailID   uuid.UUID
}

type MatchListPage struct {
	Users []PublicUserSummary
	Next  *MatchListCursor
}
