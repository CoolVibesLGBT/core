package handlers

import (
	"context"
	"core/adapters/inbound/http/middleware"
	upload "core/adapters/media"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	chatmodel "core/models/chat"
	"core/utils"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatService ports.ChatUseCase
}

func NewChatHandler(chatService ports.ChatUseCase) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func HandleSendTypingEvent(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		// form parse -> Fiber otomatik yapar
		chatIdStr := c.FormValue("chat_id")
		if chatIdStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id")
		}

		chatId, err := uuid.Parse(chatIdStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id format")
		}

		// service call
		err = s.SendTypingEvent(chatId, authUser, true)
		if err != nil {
			if errors.Is(err, chatmodel.ErrNotParticipant) {
				return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
			}
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to send typing event")
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"success": true,
		})
	}
}

func HandleSendMessage(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		images := form.File["images[]"]
		videos := form.File["videos[]"]
		formParams := form.Value

		if _, err := chatmodel.ParseExpiresInSeconds(formParams["expires_in_seconds"]); err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		if _, err := chatmodel.ParseViewOnce(formParams["view_once"]); err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		_post, err := s.AddMessageToChat(c.Context(), formParams, upload.FromMultipartFiles(images, videos), user)
		if err != nil {
			if errors.Is(err, chatmodel.ErrEmptyMessage) || errors.Is(err, chatmodel.ErrInvalidViewOnce) || errors.Is(err, chatmodel.ErrInvalidEmoticon) || errors.Is(err, chatmodel.ErrInvalidReply) {
				return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
			}
			if errors.Is(err, chatmodel.ErrNotParticipant) {
				return utils.SendErrorWithMessage(c, fiber.StatusForbidden, constants.ErrPermissionDenied, err.Error())
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"success": true,
			"message": _post,
		})
	}
}

func HandleCreateChat(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUnauthorized)
		}

		// form values read directly
		chatType := c.FormValue("type")
		if chatType == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUnsupportedChatType)
		}

		participantIDs, err := parseChatParticipantIdentifiers(c)
		if err != nil {
			return sendChatParticipantError(c, err)
		}

		var chat *types.Chat
		switch chatmodel.ChatType(chatType) {
		case chatmodel.ChatTypePrivate:
			if len(participantIDs) != 1 {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidParticipantsLength)
			}
			chat, err = s.CreateChatFromIdentifier(c.Context(), participantIDs[0], authUser, chatType)
		case chatmodel.ChatTypeGroup:
			header, _ := c.FormFile("avatar")
			if header != nil || c.FormValue("visibility") != "" {
				mediaService, ok := s.(ports.ChatGroupMediaUseCase)
				if !ok {
					return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
				}
				chat, err = mediaService.CreateGroupChatWithAvatar(c.Context(), participantIDs, authUser, c.FormValue("title"), c.FormValue("description"), upload.FromMultipart(header), chatmodel.GroupOptions{Visibility: c.FormValue("visibility"), PublicUsername: c.FormValue("public_username")})
			} else {
				chat, err = s.CreateGroupChat(c.Context(), participantIDs, authUser, c.FormValue("title"), c.FormValue("description"))
			}
		default:
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUnsupportedChatType)
		}
		if errors.Is(err, chatmodel.ErrInvalidGroupAvatar) || errors.Is(err, chatmodel.ErrInvalidGroupSettings) {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidInput)
		}
		if err != nil {
			return sendChatParticipantError(c, err)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"chat": chat,
		})
	}
}

func parseChatParticipantIdentifiers(c fiber.Ctx) ([]string, error) {
	var repeated []string
	if form, err := c.MultipartForm(); err == nil && form != nil {
		repeated = form.Value["participant_ids[]"]
		if len(form.Value["participant_ids"]) > 1 || (len(repeated) > 0 && len(form.Value["participant_ids"]) > 0) {
			return nil, constants.ErrInvalidParticipantsLength
		}
	} else {
		for key, value := range c.Request().PostArgs().All() {
			if string(key) == "participant_ids[]" {
				repeated = append(repeated, string(value))
			}
		}
		if len(c.Request().PostArgs().PeekMulti("participant_ids")) > 1 || (len(repeated) > 0 && c.FormValue("participant_ids") != "") {
			return nil, constants.ErrInvalidParticipantsLength
		}
	}
	if len(repeated) == 0 {
		raw := strings.TrimSpace(c.FormValue("participant_ids"))
		if raw == "" {
			return nil, constants.ErrInvalidParticipantsLength
		}
		if strings.HasPrefix(raw, "[") {
			var identifiers []json.RawMessage
			if err := json.Unmarshal([]byte(raw), &identifiers); err != nil {
				return nil, constants.ErrInvalidParticipantID
			}
			for _, encoded := range identifiers {
				var identifier string
				if len(encoded) > 0 && encoded[0] == '"' {
					if err := json.Unmarshal(encoded, &identifier); err != nil {
						return nil, constants.ErrInvalidParticipantID
					}
				} else {
					identifier = string(encoded)
					if _, err := strconv.ParseInt(identifier, 10, 64); err != nil {
						return nil, constants.ErrInvalidParticipantID
					}
				}
				repeated = append(repeated, identifier)
			}
		} else {
			repeated = []string{raw}
		}
	}
	if len(repeated) == 0 {
		return nil, constants.ErrInvalidParticipantsLength
	}
	for i := range repeated {
		repeated[i] = strings.TrimSpace(repeated[i])
		if repeated[i] == "" {
			return nil, constants.ErrInvalidParticipantID
		}
	}
	return repeated, nil
}

func HandleChatMessageRead(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		chatValues := form.Value["chat_id"]
		if len(chatValues) == 0 || chatValues[0] == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatValues[0])
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		messageValues := form.Value["message_ids"]

		if len(messageValues) == 0 {
			messageValues = form.Value["message_ids[]"]
		}

		if len(messageValues) == 0 {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}

		var messageIDs []uuid.UUID

		for _, id := range messageValues {
			parsed, err := uuid.Parse(id)
			if err != nil {
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
			}
			messageIDs = append(messageIDs, parsed)
		}

		err = s.MarkChatMessageRead(c.Context(), authUser, chatID, messageIDs)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrUnknown)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Messages marked as read")

	}
}

func HandleChatMessageOpen(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatID, err := uuid.Parse(c.FormValue("chat_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		messageID, err := uuid.Parse(c.FormValue("message_id"))
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}

		result, err := s.OpenMessage(c.Context(), authUser, chatID, messageID, time.Now().UTC())
		if err != nil {
			switch {
			case errors.Is(err, chatmodel.ErrNotParticipant), errors.Is(err, chatmodel.ErrAuthorCannotOpen):
				return utils.SendErrorWithMessage(c, fiber.StatusForbidden, constants.ErrPermissionDenied, err.Error())
			case errors.Is(err, chatmodel.ErrMessageNotFound):
				return utils.SendErrorWithMessage(c, fiber.StatusNotFound, constants.ErrInvalidMessageID, err.Error())
			case errors.Is(err, chatmodel.ErrMessageExpired):
				return utils.SendErrorWithMessage(c, fiber.StatusGone, constants.ErrInvalidMessageID, err.Error())
			case errors.Is(err, chatmodel.ErrMessageAlreadySeen):
				return utils.SendErrorWithMessage(c, fiber.StatusConflict, constants.ErrInvalidAction, err.Error())
			case errors.Is(err, chatmodel.ErrInvalidViewOnce), errors.Is(err, chatmodel.ErrNotDisappearing):
				return utils.SendErrorWithMessage(c, fiber.StatusUnprocessableEntity, constants.ErrInvalidInput, err.Error())
			default:
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		c.Set(fiber.HeaderCacheControl, "private, no-store")
		c.Set("Pragma", "no-cache")
		return utils.SendSuccess(c, fiber.StatusOK, result)
	}
}

func HandleGetChatsByUserID(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		limit, err := parseChatPageLimit(c)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		cursor, err := parseChatListCursor(requestField(c, "cursor"))
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		chats, nextCursor, err := s.FetchChats(c.Context(), authUser.ID, cursor, limit)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrUnknown)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"chats":  chats,
			"cursor": nextCursor,
		})
	}
}

func HandleGetMessagesByChatID(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDRaw := requestField(c, "chat_id")
		if chatIDRaw == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDRaw)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}
		limit, err := parseChatPageLimit(c)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}
		cursor, err := parseChatMessageListCursor(requestField(c, "cursor"))
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		messages, nextCursor, err := s.FetchChatMessages(c.Context(), authUser.ID, chatID, cursor, limit)
		if err != nil {
			if errors.Is(err, chatmodel.ErrNotParticipant) {
				return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
			}
			if errors.Is(err, context.DeadlineExceeded) {
				return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
			}
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToLoadMessages)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"messages": messages,
			"cursor":   nextCursor,
		})
	}
}

func parseChatPageLimit(c fiber.Ctx) (int, error) {
	raw := strings.TrimSpace(requestField(c, "limit"))
	if raw == "" {
		return constants.DEFAULT_LIMIT, nil
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 0, errors.New("limit must be a positive integer")
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}
	return limit, nil
}

func parseChatListCursor(raw string) (*ports.ChatListCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if values, ok := types.DecodePaginationCursor(raw); ok {
		activityAt, timeOK := types.CursorCreatedAt(values)
		chatID, idOK := types.CursorUUID(values)
		if timeOK && idOK {
			return &ports.ChatListCursor{ActivityAt: activityAt, ChatID: chatID}, nil
		}
		return nil, errors.New("invalid chat cursor")
	}

	// Compatibility for clients that previously derived their cursor from
	// last_message_timestamp. New responses always return the tie-safe token.
	activityAt, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		activityAt, err = time.Parse(time.RFC3339, raw)
	}
	if err != nil {
		return nil, errors.New("invalid chat cursor")
	}
	return &ports.ChatListCursor{ActivityAt: activityAt, ChatID: uuid.Nil}, nil
}

func parseChatMessageListCursor(raw string) (*ports.ChatMessageListCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if values, ok := types.DecodePaginationCursor(raw); ok {
		if publicID, ok := types.CursorPublicID(values); ok && publicID > 0 {
			return &ports.ChatMessageListCursor{PublicID: publicID}, nil
		}
		return nil, errors.New("invalid message cursor")
	}
	publicID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || publicID <= 0 {
		return nil, errors.New("invalid message cursor")
	}
	return &ports.ChatMessageListCursor{PublicID: publicID}, nil
}

func sendChatMutationError(c fiber.Ctx, err error, fallback constants.ErrorCode) error {
	switch {
	case errors.Is(err, chatmodel.ErrNotParticipant), errors.Is(err, chatmodel.ErrPermissionDenied):
		return utils.SendError(c, fiber.StatusForbidden, constants.ErrPermissionDenied)
	case errors.Is(err, chatmodel.ErrChatNotFound):
		return utils.SendError(c, fiber.StatusNotFound, constants.ErrInvalidChatID)
	case errors.Is(err, chatmodel.ErrMessageNotFound):
		return utils.SendError(c, fiber.StatusNotFound, constants.ErrInvalidMessageID)
	case errors.Is(err, context.DeadlineExceeded):
		return utils.SendError(c, fiber.StatusGatewayTimeout, constants.ErrRequestTimeout)
	default:
		return utils.SendError(c, fiber.StatusInternalServerError, fallback)
	}
}

func HandlePinMessage(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}

		// service call
		err = s.PinMessage(c.Context(), authUser, chatID, authUser.ID, messageID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToPinMessage)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message pinned successfully")
	}
}

func HandleUnpinMessage(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}

		// service call
		err = s.UnpinMessage(c.Context(), authUser, chatID, authUser.ID, messageID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToUnpinMessage)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message unpinned successfully")
	}
}

func HandleDeleteMessageForUser(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}

		// service call
		err = s.DeleteMessageForUser(c.Context(), authUser, chatID, authUser.ID, messageID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteMessageForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message deleted successfully")
	}
}

func HandleDeleteMessageForAll(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")

		if chatIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		messageIDStr := c.FormValue("message_id")
		if messageIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}
		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}

		// service call
		err = s.DeleteMessageForAll(c.Context(), authUser, chatID, authUser.ID, messageID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteMessageForAll)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message deleted successfully")
	}
}

func HandleDeleteChatForUser(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")

		if chatIDStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id")
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id format")
		}

		// service call
		err = s.DeleteChatForUser(c.Context(), authUser, chatID, authUser.ID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteChatForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat deleted successfully")
	}
}

func HandleDeleteChatForAll(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")
		if chatIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		err = s.DeleteChatForAll(c.Context(), authUser, chatID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteChatForAll)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat deleted successfully")
	}
}

func HandleDeleteChat(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")
		if chatIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		// service call
		err = s.DeleteChat(c.Context(), authUser, chatID, authUser.ID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteChatForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat deleted successfully")
	}
}

func HandleDeleteMessage(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidMessageID)
		}

		err = s.DeleteMessage(c.Context(), authUser, chatID, authUser.ID, messageID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteMessage)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message deleted successfully")
	}
}

func HandleClearChatHistoryForUser(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)

		}

		chatIDStr := c.FormValue("chat_id")
		if chatIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		err = s.DeleteChatHistoryForUser(c.Context(), authUser, chatID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteChatHistoryForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat history cleared successfully")
	}
}

func HandleClearChatHistoryForAll(s ports.ChatUseCase) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chatIDStr := c.FormValue("chat_id")
		if chatIDStr == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		err = s.DeleteChatHistoryForAll(c.Context(), authUser, chatID)
		if err != nil {
			return sendChatMutationError(c, err, constants.ErrFailedToDeleteChatHistoryForAll)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat history cleared successfully")
	}
}
