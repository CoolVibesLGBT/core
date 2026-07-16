package user

import "errors"

var ErrUnsupportedInteraction = errors.New("unsupported interaction")

type EngagementKind string

const (
	EngagementFollowing       EngagementKind = "following"
	EngagementFollower        EngagementKind = "follower"
	EngagementLikeGiven       EngagementKind = "like_given"
	EngagementLikeReceived    EngagementKind = "like_received"
	EngagementDislikeGiven    EngagementKind = "dislike_given"
	EngagementDislikeReceived EngagementKind = "dislike_received"
	EngagementBlocking        EngagementKind = "blocking"
	EngagementBlockedBy       EngagementKind = "blocked_by"
	EngagementSubscribing     EngagementKind = "subscribing"
	EngagementSubscribedBy    EngagementKind = "subscribed_by"
)

type EngagementPair struct {
	Given    EngagementKind
	Received EngagementKind
}

func InteractionEngagementPair(kind InteractionKind, positive bool) (EngagementPair, error) {
	switch kind {
	case InteractionFollow:
		return EngagementPair{Given: EngagementFollowing, Received: EngagementFollower}, nil
	case InteractionLike:
		if positive {
			return EngagementPair{Given: EngagementLikeGiven, Received: EngagementLikeReceived}, nil
		}
		return EngagementPair{Given: EngagementDislikeGiven, Received: EngagementDislikeReceived}, nil
	case InteractionBlock:
		return EngagementPair{Given: EngagementBlocking, Received: EngagementBlockedBy}, nil
	case InteractionSubscribe:
		return EngagementPair{Given: EngagementSubscribing, Received: EngagementSubscribedBy}, nil
	default:
		return EngagementPair{}, ErrUnsupportedInteraction
	}
}
