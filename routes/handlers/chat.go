package handlers

import (
	"coolvibes/constants"
	"coolvibes/middleware"
	services "coolvibes/services/user"
	"coolvibes/utils"
	"mime/multipart"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ChatHandler struct {
	chatService *services.ChatService
}

func NewChatHandler(chatService *services.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func HandleSendTypingEvent(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

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

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleSendMessage(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		user, ok := middleware.GetAuthenticatedUser(c)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Could not parse multipart form: " + err.Error())
		}

		images := form.File["images[]"]
		videos := form.File["videos[]"]
		formParams := form.Value
		files := append([]*multipart.FileHeader{}, images...)
		files = append(files, videos...)

		_post, err := s.AddMessageToChat(formParams, files, user)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Send message failed")
		}
		return utils.SendJSON(c, fiber.StatusOK, map[string]interface{}{
			"success": true,
			"message": _post,
		})
	}
}

func HandleCreateChat(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

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

		chat, err := s.CreateChat(parsedParticipantId, authUser.ID, chatType)
		if err != nil {
			return c.
				Status(fiber.StatusBadRequest).
				JSON(fiber.Map{
					"success": false,
					"code":    constants.ErrUnknown,
					"error":   err.Error(),
				})
		}

		return c.JSON(fiber.Map{
			"success": true,
			"chat":    chat,
		})
	}
}

func HandleGetChatsByUserID(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		// servis çağrısı
		chats, err := s.GetChatsByUserID(authUser.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to fetch chats")
		}

		// json response
		return c.JSON(fiber.Map{
			"success": true,
			"chats":   chats,
		})
	}
}

func HandleGetMessagesByChatID(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		// multipart parse — Fiber kendisi yapar
		form, err := c.MultipartForm()
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Could not parse multipart form: " + err.Error())
		}

		// read chat_id
		values := form.Value["chat_id"]
		if len(values) == 0 || values[0] == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat")
		}

		chatIdStr := values[0]

		chatId, err := uuid.Parse(chatIdStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id")
		}

		// service call
		messages, err := s.GetMessagesByChatID(authUser.ID, chatId)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to load messages")
		}

		// JSON response
		return c.JSON(fiber.Map{
			"success":  true,
			"messages": messages,
		})
	}
}

func HandlePinMessage(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat or message id")
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id format")
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid message id format")
		}

		// service call
		err = s.PinMessage(c.Context(), authUser, authUser.ID, chatID, messageID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to pin message")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleUnpinMessage(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat or message id")
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id format")
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid message id format")
		}

		// service call
		err = s.UnpinMessage(c.Context(), authUser, authUser.ID, chatID, messageID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to unpin message")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleDeleteMessageForUser(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat or message id")
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id format")
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid message id format")
		}

		// service call
		err = s.DeleteMessageForUser(c.Context(), authUser, chatID, authUser.ID, messageID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete message for user")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleDeleteMessageForAll(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
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
		err = s.DeleteChatForAll(c.Context(), authUser, chatID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete chat for all")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleDeleteChatForUser(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
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
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete chat for user")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleDeleteChatForAll(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
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
		err = s.DeleteChatForAll(c.Context(), authUser, chatID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete chat for all")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleDeleteChat(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
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
		err = s.DeleteChat(c.Context(), authUser, chatID, authUser.ID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete chat")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}

func HandleDeleteMessage(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		chatIDStr := c.FormValue("chat_id")
		messageIDStr := c.FormValue("message_id")

		if chatIDStr == "" || messageIDStr == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat or message id")
		}

		chatID, err := uuid.Parse(chatIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat id format")
		}

		messageID, err := uuid.Parse(messageIDStr)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid message id format")
		}

		err = s.DeleteMessage(c.Context(), authUser, chatID, authUser.ID, messageID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to delete message")
		}

		return c.JSON(fiber.Map{
			"success": true,
		})
	}
}
