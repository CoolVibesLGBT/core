package handlers

import (
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

		// authenticated user
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

		// authenticated user
		authUser, ok := middleware.GetAuthenticatedUser(c)
		if !ok || authUser == nil {
			return c.Status(fiber.StatusUnauthorized).SendString("User not authenticated")
		}

		// form values read directly
		chatType := c.FormValue("type")
		if chatType == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid chat type")
		}

		// array form field: participant_ids[]
		participantIds := c.FormValue("participant_ids[]")
		if participantIds == "" {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid participants length")
		}

		parsedParticipantId, err := uuid.Parse(participantIds)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid participant id")
		}

		chat, err := s.CreateChat(parsedParticipantId, authUser.ID, chatType)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString("Failed to create chat")
		}

		return c.JSON(fiber.Map{
			"success": true,
			"chat":    chat,
		})
	}
}

func HandleGetChatsByUserID(s *services.ChatService) fiber.Handler {
	return func(c *fiber.Ctx) error {

		// authenticated user alma
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

		// authenticated user
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
