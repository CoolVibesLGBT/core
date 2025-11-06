package chat

type MessageType string
type ParticipantRole string
type MessageStatus string
type ChatType string

const (
	Text      MessageType = "text"
	Image     MessageType = "image"
	Video     MessageType = "video"
	Audio     MessageType = "audio"
	GIF       MessageType = "gif"
	Sticker   MessageType = "sticker"
	File      MessageType = "file"
	Location  MessageType = "location"
	System    MessageType = "system"
	Gift      MessageType = "gift"
	Poll      MessageType = "poll"
	CallAudio MessageType = "call_audio"
	CallVideo MessageType = "call_video"
)

const (
	Pending   MessageStatus = "pending"
	Delivered MessageStatus = "delivered"
	Seen      MessageStatus = "seen"
	Deleted   MessageStatus = "deleted"
)

const (
	ChatTypePrivate ChatType = "private"
	ChatTypeGroup   ChatType = "group"
	ChatTypeChannel ChatType = "channel"
)
