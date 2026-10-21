package user

type FriendshipStatus string

const (
	FriendshipPending   FriendshipStatus = "pending"
	FriendshipAccepted  FriendshipStatus = "accepted"
	FriendshipDenied    FriendshipStatus = "denied"
	FriendshipCancelled FriendshipStatus = "cancelled"
	FriendshipRemoved   FriendshipStatus = "removed"
)

type FriendshipScope string

const (
	FriendshipScopeFriends  FriendshipScope = "friends"
	FriendshipScopeIncoming FriendshipScope = "incoming"
	FriendshipScopeOutgoing FriendshipScope = "outgoing"
)

func DecideFriendship(current, next FriendshipStatus, requester bool) (FriendshipStatus, bool, error) {
	switch next {
	case FriendshipAccepted, FriendshipDenied:
		if requester {
			return current, false, ErrFriendshipForbidden
		}
		if current != FriendshipPending && current != next {
			return current, false, ErrFriendshipTransition
		}
	case FriendshipCancelled:
		if !requester {
			return current, false, ErrFriendshipForbidden
		}
		if current != FriendshipPending && current != next {
			return current, false, ErrFriendshipTransition
		}
	case FriendshipRemoved:
		if current != FriendshipAccepted && current != next {
			return current, false, ErrFriendshipTransition
		}
	default:
		return current, false, ErrFriendshipTransition
	}
	return next, next != current, nil
}
