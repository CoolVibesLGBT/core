package media

type MediaRole string
type OwnerType string

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

	// Owner Type
	OwnerUser OwnerType = "user"
	OwnerPost OwnerType = "post"
	OwnerBlog OwnerType = "blog"
	OwnerChat OwnerType = "chat"
	OwnerPage OwnerType = "page"
)
