package db

func matchIndexDefinitions() []IndexDefinition {
	return []IndexDefinition{
		{
			Name:    "idx_engagement_details_actor_kind_cursor",
			Table:   "engagement_details",
			Using:   "btree",
			Columns: []string{"engager_id", "kind", "created_at DESC", "id DESC"},
		},
		{
			Name:    "idx_engagement_details_pair_kind",
			Table:   "engagement_details",
			Using:   "btree",
			Columns: []string{"engager_id", "engagee_id", "kind"},
		},
	}
}
