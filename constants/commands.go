package constants

import "encoding/json"

const APPLICATION_NAME = "COOLVIBES"

type CommandEnvelope struct {
	Version string          `json:"version"`
	Code    string          `json:"code"`
	Payload json.RawMessage `json:"payload"`
}

type TCommandTypes int

const (
	//SYSTEM
	CMD_INITIAL_SYNC    = "system.initial_sync"
	CMD_PAYMENT_METHODS = "system.payment_methods"
	CMD_LINK_METADATA   = "system.link_metadata"

	CMD_GET_VAPID_PUBLIC_KEY = "system_vapid_get_key"
	CMD_SET_VAPID_SUBSCRIBE  = "system_vapid_subscribe"
	CMD_GET_NOTIFICATIONS    = "system_notifications"

	// AUTH
	CMD_AUTH_LOGIN     = "auth.login"
	CMD_AUTH_REGISTER  = "auth.register"
	CMD_AUTH_LOGOUT    = "auth.logout"
	CMD_AUTH_TEST      = "auth.test"
	CMD_AUTH_USER_INFO = "auth.user_info"
	CMD_AUTH_CHECK     = "auth.check"

	// CHAT
	CMD_CHAT_SEND_TEXT              = "chat.send_text"
	CMD_CHAT_SEND_GIF               = "chat.send_gif"
	CMD_CHAT_SEND_CALL              = "chat.send_call"
	CMD_CHAT_SEND_STICKER           = "chat.send_sticker"
	CMD_CHAT_MESSAGE_READ           = "chat.message_read"
	CMD_CHAT_CREATE                 = "chat.create"
	CMD_TYPING                      = "chat.typing"
	CMD_SEND_MESSAGE                = "chat.send_message"
	CMD_EDIT_MESSAGE                = "chat.edit_message"
	CMD_DELETE_CHAT                 = "chat.delete_chat"
	CMD_FETCH_CHATS                 = "chat.fetch_chats"
	CMD_DELETE_MESSAGE              = "chat.delete_message"
	CMD_FETCH_MESSAGES              = "chat.fetch_messages"
	CMD_DELETE_MESSAGE_FOR_USER     = "chat.delete_message_for_user"
	CMD_DELETE_MESSAGE_FOR_ALL      = "chat.delete_message_for_all"
	CMD_DELETE_CHAT_FOR_USER        = "chat.delete_chat_for_user"
	CMD_DELETE_CHAT_FOR_ALL         = "chat.delete_chat_for_all"
	CMD_CLEAR_CHAT_HISTORY_FOR_USER = "chat.clear_chat_history_for_user"
	CMD_CLEAR_CHAT_HISTORY_FOR_ALL  = "chat.clear_chat_history_for_all"

	CMD_PIN_MESSAGE   = "chat.pin_message"
	CMD_UNPIN_MESSAGE = "chat.unpin_message"

	// LISTINGS
	CMD_FETCH_JOB_OFFERS   = "classifieds.offers"
	CMD_FETCH_JOB_SEARCH   = "classifieds.search"
	CMD_CLASSIFIEDS_CREATE = "classifieds.create"
	CMD_CLASSIFIEDS_FETCH  = "classifieds.get"

	// USER
	CMD_USER_UPDATE_PREFERENCES = "user.update_preferences"
	CMD_USER_UPDATE_IDENTIFY    = "user.update_identify"
	CMD_USER_CHECK_IN           = "user.check_in"
	CMD_USER_CHECK_IN_FETCH     = "user.check_in_fetch"
	CMD_USER_UPLOAD_AVATAR      = "user.upload_avatar"
	CMD_USER_UPLOAD_COVER       = "user.upload_cover"
	CMD_USER_UPLOAD_STORY       = "user.upload_story"
	CMD_UPDATE_USER_PROFILE     = "user.update_profile"
	CMD_USER_DELETE_PROFILE     = "user.delete_profile"

	CMD_USER_FETCH_STORIES     = "user.fetch.stories"
	CMD_USER_FETCH_PROFILE     = "user.fetch_profile"
	CMD_USER_FETCH_ENGAGEMENTS = "user.fetch_engagements"

	CMD_USER_FETCH_FOLLOWINGS = "user.fetch.followings"
	CMD_USER_FETCH_FOLLOWERS  = "user.fetch.followers"

	CMD_USER_FOLLOW        = "user.follow"
	CMD_USER_UNFOLLOW      = "user.unfollow"
	CMD_USER_TOGGLE_FOLLOW = "user.follow.toggle"

	CMD_USER_LIKE           = "user.like"
	CMD_USER_DISLIKE        = "user.dislike"
	CMD_USER_TOGGLE_LIKE    = "user.like.toggle"
	CMD_USER_TOGGLE_DISLIKE = "user.dislike.toggle"

	CMD_USER_BLOCK            = "user.block"
	CMD_USER_UNBLOCK          = "user.unblock"
	CMD_USER_TOGGLE_BLOCK     = "user.block.toggle"
	CMD_USER_REPORT           = "user.report"
	CMD_USER_TOGGLE_SUBSCRIBE = "user.subscribe.toggle"

	CMD_USER_FETCH_NEARBY_USERS      = "user.fetch.nearby.users"
	CMD_USER_FETCH_BROADCASTERS      = "user.fetch.broadcasters"
	CMD_USER_GET_NOTIFICATIONS       = "user.fetch.notifications"
	CMD_USER_MARK_NOTIFICATIONS_SEEN = "user.notifications.mark.seen"

	CMD_USER_POSTS          = "user.fetch.posts"
	CMD_USER_POST_REPLIES   = "user.fetch.posts.replies"
	CMD_USER_POST_MEDIA     = "user.fetch.posts.media"
	CMD_USER_POST_LIKES     = "user.fetch.posts.likes"
	CMD_USER_POST_BOOKMARKS = "user.fetch.posts.bookmarks"

	CMD_USER_DEPOSIT      = "user.deposit"
	CMD_USER_WITHDRAW     = "user.withdraw"
	CMD_USER_TRANSACTIONS = "user.transactions"

	CMD_POST_CATEGORIES = "post.categories"
	CMD_POST_CREATE     = "post.create"
	CMD_POST_VOTE       = "post.vote"
	CMD_POST_UPDATE     = "post.update"
	CMD_POST_DELETE     = "post.delete"
	CMD_POST_GET        = "post.get"
	CMD_POST_FETCH      = "post.fetch"
	CMD_POST_SEARCH     = "post.search"
	CMD_POST_TIMELINE   = "post.timeline"
	CMD_POST_VIBES      = "post.vibes"
	CMD_POST_LIKE       = "post.like"
	CMD_POST_DISLIKE    = "post.dislike"
	CMD_POST_BOOKMARK   = "post.bookmark"
	CMD_POST_SUBSCRIBE  = "post.subscribe"

	CMD_POST_REPORT = "post.report"
	CMD_POST_VIEW   = "post.view"
	CMD_POST_BANANA = "post.banana"
	CMD_POST_TIP    = "post.tip"

	// MODERATION
	CMD_MODERATION_REPORTS_FETCH  = "moderation.reports.fetch"
	CMD_MODERATION_REPORT_RESOLVE = "moderation.report.resolve"
	CMD_MODERATION_POST_HIDE      = "moderation.post.hide"
	CMD_MODERATION_POST_UNHIDE    = "moderation.post.unhide"

	//MATCH EKRANI
	CMD_MATCH_CREATE = "match.create" // Yeni eşleşme oluşturma (örneğin karşılıklı like)
	CMD_MATCH_DELETE = "match.delete" // Eşleşmeyi kaldırma
	CMD_MATCH_FETCH  = "match.fetch"  // Tüm eşleşmeleri listeleme

	CMD_MATCH_FETCH_LIKED   = "match.fetch.liked"   // Beğenilen kullanıcıları getirme
	CMD_MATCH_FETCH_PASSED  = "match.fetch.passed"  // Geçilen kullanıcıları getirme
	CMD_MATCH_FETCH_MATCHED = "match.fetch.matched" // Karşılıklı eşleşmeleri getirme (gerçek matchler)

	CMD_MATCH_GET_UNSEEN = "match.fetch.unseen" // Görülmemiş eşleşmeler
	CMD_MATCH_UPDATE     = "match.update"       // Eşleşme durumunu güncelleme

	CMD_SEARCH_LOOKUP_USER = "search.user.lookup"
	CMD_SEARCH_TRENDS      = "search.trends"

	//PLACE EKRANI
	CMD_PLACE_CATEGORIES     = "place.categories"
	CMD_PLACE_CREATE         = "place.create"
	CMD_PLACE_FETCH          = "place.fetch"
	CMD_PLACE_VOTE           = "place.vote"
	CMD_PLACE_DELETE         = "place.delete"
	CMD_PLACE_UPDATE         = "place.update"
	CMD_PLACE_COMMENT        = "place.comment"
	CMD_PLACE_FETCH_COMMENTS = "place.fetch.comments"

	//NEWS EKRANI
	CMD_NEWS_CREATE         = "news.create"
	CMD_NEWS_FETCH          = "news.fetch"
	CMD_NEWS_GET            = "news.get"
	CMD_NEWS_CATEGORIES     = "news.categories"
	CMD_NEWS_VOTE           = "news.vote"
	CMD_NEWS_DELETE         = "news.delete"
	CMD_NEWS_UPDATE         = "news.update"
	CMD_NEWS_COMMENT        = "news.comment"
	CMD_NEWS_FETCH_COMMENTS = "news.fetch.comments"

	//	BROADCASTS
	CMD_BROADCASTS_FETCH  = "broadcast.fetch"
	CMD_BROADCASTS_JOIN   = "broadcast.join"
	CMD_BROADCASTS_VIEW   = "broadcast.view"
	CMD_BROADCASTS_CREATE = "broadcast.create"
	CMD_BROADCASTS_LIKE   = "broadcast.like"
)

/*
func main() {
	// Example usage
	command := ACT_ACT_LOGIN
	switch command {
	case ACT_ACT_PROMPT:
		// Handle prompt action
	case ACT_ACT_REGISTER:
		// Handle register action
	case ACT_ACT_LOGIN:
		// Handle login action
	case ACT_ACT_PROFILE:
		// Handle profile action
	case ACT_ACT_REQUEST:
		// Handle request action
	case ACT_ACT_CHECK_AUTH:
		// Handle check auth action
	default:
		// Handle unknown action
	}
}
*/
