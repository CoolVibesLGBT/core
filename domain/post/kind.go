package post

type Kind string

const (
	KindStatus     Kind = "status"
	KindTimeline   Kind = "timeline"
	KindPlace      Kind = "place"
	KindClassified Kind = "classified"
	KindJobOffer   Kind = "job_offer"
	KindJobSearch  Kind = "job_search"
	KindGeneric    Kind = "generic"
	KindNews       Kind = "news"
	KindStory      Kind = "story"
	KindChat       Kind = "chat"
	KindMessage    Kind = "message"
	KindPost       Kind = "post"
	KindEvent      Kind = "event"
	KindCheckIn    Kind = "checkin"
	KindVideo      Kind = "video"
)

func ParseKind(input string) (Kind, bool) {
	kind := Kind(input)
	if !kind.IsValid() {
		return "", false
	}
	return kind, true
}

func (k Kind) IsValid() bool {
	switch k {
	case KindStatus,
		KindTimeline,
		KindPlace,
		KindClassified,
		KindGeneric,
		KindNews,
		KindStory,
		KindChat,
		KindMessage,
		KindPost,
		KindVideo,
		KindEvent,
		KindCheckIn,
		KindJobOffer,
		KindJobSearch:
		return true
	default:
		return false
	}
}
