package db

import (
	"strings"
	"testing"
)

func TestChatPaginationIndexesCoverMembershipActivityAndMessages(t *testing.T) {
	definitions := chatPaginationIndexDefinitions()
	if len(definitions) != 3 {
		t.Fatalf("chat pagination index count = %d, want 3", len(definitions))
	}

	byName := make(map[string]IndexDefinition, len(definitions))
	for _, definition := range definitions {
		byName[definition.Name] = definition
	}

	participant := byName["idx_chat_participants_active_user_chat"]
	if strings.Join(participant.Columns, ",") != "user_id,chat_id" || participant.Condition != "left_at IS NULL" {
		t.Fatalf("unexpected participant index: %#v", participant)
	}

	activity := byName["idx_chats_active_activity_page"]
	if strings.Join(activity.Columns, ",") != "COALESCE(last_message_timestamp, created_at) DESC,id DESC" || activity.Condition != "deleted_at IS NULL" {
		t.Fatalf("unexpected chat activity index: %#v", activity)
	}

	messages := byName["idx_posts_active_chat_message_page"]
	if strings.Join(messages.Columns, ",") != "contentable_id,public_id DESC" {
		t.Fatalf("unexpected message page columns: %#v", messages)
	}
	for _, predicate := range []string{"deleted_at IS NULL", "post_kind = 'message'", "contentable_type = 'chat'"} {
		if !strings.Contains(messages.Condition, predicate) {
			t.Fatalf("message index condition missing %q: %#v", predicate, messages)
		}
	}
}
