package handlers

import (
	"coolvibes/middleware"
	services "coolvibes/services/user"
	"coolvibes/utils"
	"mime/multipart"
	"net/http"

	"github.com/google/uuid"
)

type ChatHandler struct {
	chatService *services.ChatService
}

func NewChatHandler(chatService *services.ChatService) *ChatHandler {
	return &ChatHandler{chatService: chatService}
}

func HandleSendTypingEvent(s *services.ChatService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		chatIdStr := r.FormValue("chat_id")
		if chatIdStr == "" {
			http.Error(w, "Invalid chat type", http.StatusUnauthorized)
			return
		}

		chatId, err := uuid.Parse(chatIdStr) // Burada artık string var
		if err != nil {
			http.Error(w, "Failed to send typng event users", http.StatusInternalServerError)

			return
		}

		err = s.SendTypingEvent(chatId, auth_user.ID, true)
		if err != nil {
			http.Error(w, "Failed to send typng event users", http.StatusInternalServerError)
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
		})
	}
}

func HandleSendMessage(s *services.ChatService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := r.ParseMultipartForm(5 * 1024 * 1024 * 1024)
		if err != nil {
			http.Error(w, "Could not parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		images := r.MultipartForm.File["images[]"] // images array
		videos := r.MultipartForm.File["videos[]"] // images array

		formParams := r.MultipartForm.Value // text fields
		files := append([]*multipart.FileHeader{}, images...)
		files = append(files, videos...)

		_post, err := s.AddMessageToChat(formParams, files, user)
		if err != nil {
			http.Error(w, "Send message failed", http.StatusInternalServerError)
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"message": _post,
		})
	}
}

func HandleCreateChat(s *services.ChatService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}

		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		chatType := r.FormValue("type")
		if chatType == "" {
			http.Error(w, "Invalid chat type", http.StatusUnauthorized)
			return
		}

		participantIds := r.Form["participant_ids[]"] // Burada dizi alıyoruz
		if len(participantIds) == 0 {
			http.Error(w, "Invalid participants length", http.StatusUnauthorized)
			return
		}

		parsedParticipantId, err := uuid.Parse(participantIds[0]) // Burada artık string var
		if err != nil {
			http.Error(w, "Invalid participant id", http.StatusUnauthorized)
			return
		}

		chat, err := s.CreateChat(parsedParticipantId, auth_user.ID, chatType)
		if err != nil {
			http.Error(w, "Failed to create chat", http.StatusInternalServerError)
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"chat":    chat,
		})
	}
}

func HandleGetChatsByUserID(s *services.ChatService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		chats, err := s.GetChatsByUserID(auth_user.ID)
		if err != nil {
			http.Error(w, "Failed to fetch GetChatsByUserID", http.StatusInternalServerError)
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"chats":   chats,
		})
	}
}

func HandleGetMessagesByChatID(s *services.ChatService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth_user, ok := middleware.GetAuthenticatedUser(r)
		if !ok {
			http.Error(w, "User not authenticated", http.StatusUnauthorized)
			return
		}

		err := r.ParseMultipartForm(5 * 1024 * 1024 * 1024)
		if err != nil {
			http.Error(w, "Could not parse multipart form: "+err.Error(), http.StatusBadRequest)
			return
		}

		chatIdStr := r.FormValue("chat_id")
		if chatIdStr == "" {
			http.Error(w, "Invalid chat", http.StatusUnauthorized)
			return
		}

		chatId, err := uuid.Parse(chatIdStr) // Burada artık string var
		if err != nil {
			http.Error(w, "Invalid chat", http.StatusUnauthorized)
			return
		}

		messages, err := s.GetMessagesByChatID(auth_user.ID, chatId)
		if err != nil {
			http.Error(w, "Failed to load messages", http.StatusInternalServerError)
			return
		}
		utils.SendJSON(w, http.StatusOK, map[string]interface{}{
			"success":  true,
			"messages": messages,
		})
	}
}
