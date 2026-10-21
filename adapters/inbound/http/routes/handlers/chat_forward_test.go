package handlers

import (
	"context"
	"core/application/types"
	"core/models"
	"core/models/chat"
	"github.com/google/uuid"
	"net/http"
	"testing"
)

type forwardHandlerService struct {
	source, target uuid.UUID
	ids            []uuid.UUID
	comment        string
	err            error
}

func (s *forwardHandlerService) ForwardMessages(_ context.Context, _ *models.User, source, target uuid.UUID, ids []uuid.UUID, comment string) ([]types.ChatMessage, error) {
	s.source, s.target, s.ids, s.comment = source, target, ids, comment
	if s.err != nil {
		return nil, s.err
	}
	return []types.ChatMessage{{ID: uuid.New()}}, nil
}
func TestForwardMessagesHTTP(t *testing.T) {
	actor := &models.User{ID: uuid.New(), PublicID: 10}
	source, target, id := uuid.New(), uuid.New(), uuid.New()
	for _, test := range []struct {
		name   string
		fields map[string]string
		actor  *models.User
		err    error
		status int
	}{
		{"valid", map[string]string{"source_chat_id": source.String(), "chat_id": target.String(), "message_ids[]": id.String(), "content": "comment"}, actor, nil, http.StatusOK},
		{"unauthenticated", map[string]string{}, nil, nil, http.StatusUnauthorized},
		{"invalid source", map[string]string{"source_chat_id": "bad", "chat_id": target.String()}, actor, nil, http.StatusBadRequest},
		{"missing IDs", map[string]string{"source_chat_id": source.String(), "chat_id": target.String()}, actor, nil, http.StatusBadRequest},
		{"bad ID", map[string]string{"source_chat_id": source.String(), "chat_id": target.String(), "message_ids[]": "bad"}, actor, nil, http.StatusBadRequest},
		{"unauthorized destination", map[string]string{"source_chat_id": source.String(), "chat_id": target.String(), "message_ids[]": id.String()}, actor, chat.ErrNotParticipant, http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &forwardHandlerService{err: test.err}
			response := performMultipartHandlerRequest(t, HandleForwardMessages(service), test.actor, test.fields, nil)
			defer func() {
				if err := response.Body.Close(); err != nil {
					t.Errorf("close response body: %v", err)
				}
			}()
			if response.StatusCode != test.status {
				t.Fatalf("status=%d want=%d", response.StatusCode, test.status)
			}
			if test.status == http.StatusOK && (service.source != source || service.target != target || len(service.ids) != 1 || service.ids[0] != id || service.comment != "comment") {
				t.Fatal("forward parameters lost")
			}
		})
	}
}
