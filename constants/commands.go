package constants

import "encoding/json"

type CommandEnvelope struct {
	Version string          `json:"version"` // "v1", "v2" gibi
	Code    string          `json:"code"`    // "chat.send_gif" gibi
	Payload json.RawMessage `json:"payload"` // tip bilinmiyor, sonra parse edilir
}

type TCommandTypes int

const (
	//SYSTEM
	CMD_INITIAL_SYNC         = "system.initial_sync"
	CMD_GET_VAPID_PUBLIC_KEY = "system_vapid_get_key"
	CMD_SET_VAPID_SUBSCRIBE  = "system_vapid_subscribe"
	CMD_GET_NOTIFICATIONS    = "system_notifications"

	// AUTH
	CMD_AUTH_LOGIN     = "auth.login"
	CMD_AUTH_REGISTER  = "auth.register"
	CMD_AUTH_LOGOUT    = "auth.logout"
	CMD_AUTH_TEST      = "auth.test"
	CMD_AUTH_USER_INFO = "auth.user_info"

	// CHAT
	CMD_CHAT_SEND_TEXT    = "chat.send_text"
	CMD_CHAT_SEND_GIF     = "chat.send_gif"
	CMD_CHAT_SEND_CALL    = "chat.send_call"
	CMD_CHAT_SEND_STICKER = "chat.send_sticker"

	// USER
	CMD_USER_UPDATE_PREFERENCES = "user.update_preferences"
	CMD_USER_UPDATE_IDENTIFY    = "user.update_identify"

	CMD_USER_UPLOAD_AVATAR  = "user.upload_avatar"
	CMD_USER_UPLOAD_COVER   = "user.upload_cover"
	CMD_USER_UPLOAD_STORY   = "user.upload_story"
	CMD_UPDATE_USER_PROFILE = "user.update_profile"

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

	CMD_USER_BLOCK        = "user.block"
	CMD_USER_UNBLOCK      = "user.unblock"
	CMD_USER_TOGGLE_BLOCK = "user.block.toggle"
	CMD_USER_REPORT       = "user.report"

	CMD_USER_FETCH_NEARBY_USERS      = "user.fetch.nearby.users"
	CMD_USER_GET_NOTIFICATIONS       = "user.fetch.notifications"
	CMD_USER_MARK_NOTIFICATIONS_SEEN = "user.notifications.mark.seen"

	CMD_USER_POSTS          = "user.fetch.posts"
	CMD_USER_POST_REPLIES   = "user.fetch.posts.replies"
	CMD_USER_POST_MEDIA     = "user.fetch.posts.media"
	CMD_USER_POST_LIKES     = "user.fetch.posts.likes"
	CMD_USER_POST_BOOKMARKS = "user.fetch.posts.bookmarks"

	CMD_POST_CREATE   = "post.create"
	CMD_POST_VOTE     = "post.vote"
	CMD_POST_UPDATE   = "post.update"
	CMD_POST_DELETE   = "post.delete"
	CMD_POST_FETCH    = "post.fetch"
	CMD_POST_TIMELINE = "post.timeline"
	CMD_POST_VIBES    = "post.vibes"
	CMD_POST_LIKE     = "post.like"
	CMD_POST_DISLIKE  = "post.dislike"
	CMD_POST_BOOKMARK = "post.bookmark"
	CMD_POST_REPORT   = "post.report"
	CMD_POST_VIEW     = "post.view"
	CMD_POST_BANANA   = "post.banana"

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

	CMD_CHAT_CREATE    = "chat.create" // Chat olustur
	CMD_TYPING         = "chat.typing"
	CMD_SEND_MESSAGE   = "chat.send_message"   // Mesaj gönder
	CMD_DELETE_CHAT    = "chat.delete_chat"    // Sohbeti sil
	CMD_FETCH_CHATS    = "chat.fetch_chats"    // Sohbetleri getir
	CMD_DELETE_MESSAGE = "chat.delete_message" // Mesajı sil
	CMD_FETCH_MESSAGES = "chat.fetch_messages" // Mesajları getir

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
