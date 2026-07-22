package db

// chatPaginationIndexDefinitions cover the three ordered/filtering legs of
// chat list and message-history keyset pagination. They are kept together so
// the migration registry and its contract test cannot drift independently.
func chatPaginationIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:      "idx_chat_participants_active_user_chat",
			Table:     "chat_participants",
			Using:     "btree",
			Columns:   []string{"user_id", "chat_id"},
			Condition: "left_at IS NULL",
		},
		{
			Name:      "idx_chats_active_activity_page",
			Table:     "chats",
			Using:     "btree",
			Columns:   []string{"COALESCE(last_message_timestamp, created_at) DESC", "id DESC"},
			Condition: "deleted_at IS NULL",
		},
		{
			Name:      "idx_posts_active_chat_message_page",
			Table:     "posts",
			Using:     "btree",
			Columns:   []string{"contentable_id", "public_id DESC"},
			Condition: "deleted_at IS NULL AND post_kind = 'message' AND contentable_type = 'chat'",
		},
	}
}
