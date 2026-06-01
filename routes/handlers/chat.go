package handlers

import (
	usecases "core/application/usecases"
	"core/constants"
	"core/middleware"
	"core/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatService *usecases.ChatService
}

func NewChatHandler(chatService *usecases.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func HandleSendTypingEvent(s *usecases.ChatService) fiber.Handler {
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
		err = s.SendTypingEvent(chatId, authUser.ID, true)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to send typing event")
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"success": true,
		})
	}
}

func HandleSendMessage(s *usecases.ChatService) fiber.Handler {
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

		_post, err := s.AddMessageToChat(c.Context(), uploadedFormData(formParams, images, videos), user)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrInternalServer)
		}
		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"success": true,
			"message": _post,
		})
	}
}

func HandleCreateChat(s *usecases.ChatService) fiber.Handler {
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

		// array form field: participant_ids[]
		participantIds := c.FormValue("participant_ids[]")
		if participantIds == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidParticipantsLength)
		}

		parsedParticipantId, err := uuid.Parse(participantIds)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidParticipantID)
		}

		chat, err := s.CreateChat(c.Context(), parsedParticipantId, authUser.ID, chatType)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrUnknown)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"chat": chat,
		})
	}
}

func HandleChatMessageRead(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrUnknown)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Messages marked as read")

	}
}

func HandleGetChatsByUserID(s *usecases.ChatService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		chats, err := s.GetChatsByUserID(authUser.ID)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrUnknown)
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"chats": chats,
		})
	}
}

func HandleGetMessagesByChatID(s *usecases.ChatService) fiber.Handler {
	return func(c fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return utils.SendError(c, fiber.StatusUnauthorized, constants.ErrUserUnauthorized)
		}

		form, err := c.MultipartForm()
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidForm)
		}

		// read chat_id
		values := form.Value["chat_id"]
		if len(values) == 0 || values[0] == "" {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		chatIdStr := values[0]

		chatId, err := uuid.Parse(chatIdStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusBadRequest, constants.ErrInvalidChatID)
		}

		// service call
		messages, err := s.GetMessagesByChatID(authUser.ID, chatId)
		if err != nil {
			return utils.SendErrorWithMessage(c, fiber.StatusInternalServerError, constants.ErrFailedToLoadMessages, err.Error())
		}

		return utils.SendSuccess(c, fiber.StatusOK, fiber.Map{
			"messages": messages,
		})
	}
}

func HandlePinMessage(s *usecases.ChatService) fiber.Handler {
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
		err = s.PinMessage(c.Context(), authUser, authUser.ID, chatID, messageID)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToPinMessage)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message pinned successfully")
	}
}

func HandleUnpinMessage(s *usecases.ChatService) fiber.Handler {
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
		err = s.UnpinMessage(c.Context(), authUser, authUser.ID, chatID, messageID)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToUnpinMessage)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message unpinned successfully")
	}
}

func HandleDeleteMessageForUser(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteMessageForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message deleted successfully")
	}
}

func HandleDeleteMessageForAll(s *usecases.ChatService) fiber.Handler {
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
		err = s.DeleteMessageForAll(c.Context(), authUser, authUser.ID, chatID, messageID)
		if err != nil {
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteChatForAll)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message deleted successfully")
	}
}

func HandleDeleteChatForUser(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteChatForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat deleted successfully")
	}
}

func HandleDeleteChatForAll(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteChatForAll)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat deleted successfully")
	}
}

func HandleDeleteChat(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteChatForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat deleted successfully")
	}
}

func HandleDeleteMessage(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteMessageForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Message deleted successfully")
	}
}

func HandleClearChatHistoryForUser(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteChatHistoryForUser)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat history cleared successfully")
	}
}

func HandleClearChatHistoryForAll(s *usecases.ChatService) fiber.Handler {
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
			return utils.SendError(c, fiber.StatusInternalServerError, constants.ErrFailedToDeleteChatHistoryForAll)
		}

		return utils.SendSuccessWithMessage(c, fiber.StatusOK, nil, "Chat history cleared successfully")
	}
}
