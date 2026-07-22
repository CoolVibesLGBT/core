package handlers

import (
	"context"
	"core/adapters/inbound/http/middleware"
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

		expiresIn, err := chatmodel.ParseExpiresInSeconds(formParams["expires_in_seconds"])
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, err.Error())
		}

		viewOnce := false
		if values := formParams["view_once"]; len(values) > 0 && values[0] != "" {
			switch strings.ToLower(strings.TrimSpace(values[0])) {
			case "on", "yes":
				viewOnce = true
			case "off", "no":
				viewOnce = false
			default:
				viewOnce, err = strconv.ParseBool(values[0])
				if err != nil {
					return utils.SendErrorWithMessage(c, fiber.StatusBadRequest, constants.ErrInvalidInput, "view_once must be true or false")
				}
			}
		}

		messageForm := uploadedFormData(formParams, images, videos)
		messageForm.ExpiresInSeconds = expiresIn
		messageForm.ViewOnce = viewOnce
		messageForm.ImageCount = len(images)
		messageForm.VideoCount = len(videos)
		if values := formParams["client_id"]; len(values) > 0 {
			messageForm.ClientID = values[0]
		}

		_post, err := s.AddMessageToChat(c.Context(), messageForm, user)
		if err != nil {
			if errors.Is(err, chatmodel.ErrEmptyMessage) || errors.Is(err, chatmodel.ErrInvalidViewOnce) {
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

		participantID, err := parseSingleChatParticipantIdentifier(c)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidParticipantsLength)
		}

		chat, err := s.CreateChatFromIdentifier(c.Context(), participantID, authUser, chatType)
		if err != nil {
			switch err.Error() {
			case constants.ErrUserNotFound.String():
				return utils.SendError(c, fiber.StatusNotFound, constants.ErrUserNotFound)
			case constants.ErrSelfChatNotAllowed.String():
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrSelfChatNotAllowed)
			case constants.ErrUnsupportedChatType.String():
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUnsupportedChatType)
			case constants.ErrInvalidParticipantID.String():
				return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidParticipantID)
			default:
				return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
			}
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"chat": chat,
		})
	}
}

func parseSingleChatParticipantIdentifier(c fiber.Ctx) (string, error) {
	if form, err := c.MultipartForm(); err == nil && form != nil {
		legacyValues := form.Value["participant_ids[]"]
		jsonValues := form.Value["participant_ids"]
		if len(legacyValues) > 0 && len(jsonValues) > 0 {
			return "", errors.New("participant identifier fields are ambiguous")
		}
		if len(legacyValues) > 0 {
			if len(legacyValues) != 1 || strings.TrimSpace(legacyValues[0]) == "" {
				return "", errors.New("exactly one participant is required")
			}
			return strings.TrimSpace(legacyValues[0]), nil
		}
		if len(jsonValues) > 1 {
			return "", errors.New("exactly one participant field is required")
		}
	}

	raw := strings.TrimSpace(c.FormValue("participant_ids"))
	if raw == "" {
		raw = strings.TrimSpace(c.FormValue("participant_ids[]"))
	}
	if raw == "" {
		return "", errors.New("one participant is required")
	}
	if !strings.HasPrefix(raw, "[") {
		return raw, nil
	}

	var identifiers []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &identifiers); err != nil || len(identifiers) != 1 {
		return "", errors.New("exactly one participant is required")
	}
	encoded := strings.TrimSpace(string(identifiers[0]))
	if strings.HasPrefix(encoded, `"`) {
		var identifier string
		if err := json.Unmarshal(identifiers[0], &identifier); err != nil || strings.TrimSpace(identifier) == "" {
			return "", errors.New("participant identifier is invalid")
		}
		return strings.TrimSpace(identifier), nil
	}
	if _, err := strconv.ParseInt(encoded, 10, 64); err != nil {
		return "", errors.New("participant identifier is invalid")
	}
	return encoded, nil
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
