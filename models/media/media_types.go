package media

type MediaRole string
type OwnerType string
type ProcessingStatus string

const (
	// Role
	RoleProfile MediaRole = "profile"
	RoleAvatar  MediaRole = "avatar"
	RoleCover   MediaRole = "cover"
	RoleStory   MediaRole = "story"

	RolePost      MediaRole = "post"
	RoleBlog      MediaRole = "blog"
	RoleChatImage MediaRole = "chat_image"
	RoleChatMedia MediaRole = "chat_media"
	RoleChatVideo MediaRole = "chat_video"
	RoleOther     MediaRole = "other"
	RoleVideo     MediaRole = "video"

	// Owner Type
	OwnerUser  OwnerType = "user"
	OwnerPost  OwnerType = "post"
	OwnerNews  OwnerType = "news"
	OwnerBlog  OwnerType = "blog"
	OwnerChat  OwnerType = "chat"
	OwnerPage  OwnerType = "page"
	OwnerVideo OwnerType = "video"

	ProcessingStatusPending    ProcessingStatus = "pending"
	ProcessingStatusProcessing ProcessingStatus = "processing"
	ProcessingStatusReady      ProcessingStatus = "ready"
	ProcessingStatusFailed     ProcessingStatus = "failed"
)
