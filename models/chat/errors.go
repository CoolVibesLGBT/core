package chat

import "errors"

var (
	ErrEmptyMessage       = errors.New("message must contain text or an attachment")
	ErrNotParticipant     = errors.New("user is not an active chat participant")
	ErrPermissionDenied   = errors.New("chat mutation is not permitted")
	ErrChatNotFound       = errors.New("chat not found")
	ErrInvalidViewOnce    = errors.New("view-once messages must contain exactly one image and no video")
	ErrMessageNotFound    = errors.New("chat message not found")
	ErrMessageExpired     = errors.New("chat message has expired")
	ErrMessageAlreadySeen = errors.New("view-once message has already been viewed")
	ErrAuthorCannotOpen   = errors.New("message authors cannot open their own disappearing message")
	ErrNotDisappearing    = errors.New("message is not a disappearing message")
)
