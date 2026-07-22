package legacyviews

import (
	"core/application/types"
	"core/models"
	"core/models/media"
	"core/models/post"
	postpayloads "core/models/post/payloads"
	"core/models/taxonomy"
	modelutils "core/models/utils"
	"encoding/json"

	"github.com/google/uuid"
)

var publicPostCountKeys = map[string]struct{}{
	"touch_count":            {},
	"banana_count":           {},
	"banana_received_count":  {},
	"banana_given_count":     {},
	"carrot_count":           {},
	"coffee_count":           {},
	"kiss_count":             {},
	"like_given_count":       {},
	"like_received_count":    {},
	"dislike_given_count":    {},
	"dislike_received_count": {},
	"comment_count":          {},
	"view_count":             {},
	"bookmark_count":         {},
	"rating_count":           {},
	"rating_sum":             {},
	"tip_count":              {},
	"tip_amount":             {},
	"gift_count":             {},
	"gift_amount":            {},
}

var publicPostEngagementKinds = map[models.EngagementKind]struct{}{
	models.EngagementKindTouch:           {},
	models.EngagementKindBanana:          {},
	models.EngagementKindCarrot:          {},
	models.EngagementKindCoffee:          {},
	models.EngagementKindKiss:            {},
	models.EngagementKindLikeGiven:       {},
	models.EngagementKindLikeReceived:    {},
	models.EngagementKindDislikeGiven:    {},
	models.EngagementKindDisLikeReceived: {},
	models.EngagementKindComment:         {},
	models.EngagementKindBookmark:        {},
	models.EngagementKindTip:             {},
	models.EngagementKindGift:            {},
}

// ProjectPublicPost is the single compatibility mapper between persistence
// entities and the public application read model. Query visibility remains a
// repository concern; this mapper only constrains what may be serialized.
func ProjectPublicPost(source post.Post) types.PublicPost {
	return projectPublicPost(source, nil, make(map[uuid.UUID]struct{}))
}

func ProjectPublicPosts(posts []post.Post) []types.PublicPost {
	result := make([]types.PublicPost, 0, len(posts))
	for i := range posts {
		result = append(result, ProjectPublicPost(posts[i]))
	}
	return result
}

func ProjectPublicPostPointers(posts []*post.Post) []types.PublicPost {
	result := make([]types.PublicPost, 0, len(posts))
	for _, item := range posts {
		if item == nil {
			continue
		}
		result = append(result, ProjectPublicPost(*item))
	}
	return result
}

func ProjectPublicPostPage(result TimelineResult) types.PublicPostPage {
	return types.PublicPostPage{Posts: ProjectPublicPosts(result.Posts), Cursor: result.Cursor}
}

func ProjectPublicPostsResult(result PostsResult) types.PublicPostPage {
	return types.PublicPostPage{Posts: ProjectPublicPosts(result.Posts), Cursor: result.Cursor}
}

func ProjectPublicMediaItems(items []MediaWithUser) []types.PublicPostMediaWithUser {
	result := make([]types.PublicPostMediaWithUser, 0, len(items))
	for i := range items {
		result = append(result, types.PublicPostMediaWithUser{
			PublicPostMedia: projectPublicMedia(&items[i].Media),
			User:            projectPublicPostAuthor(items[i].User),
		})
	}
	return result
}

func projectPublicPost(source post.Post, parentID *types.SnowflakeID, visited map[uuid.UUID]struct{}) types.PublicPost {
	if source.ID != uuid.Nil {
		if _, exists := visited[source.ID]; exists {
			return types.PublicPost{
				ID:       types.SnowflakeID(source.PublicID),
				PublicID: types.SnowflakeID(source.PublicID),
				ParentID: parentID,
			}
		}
		visited[source.ID] = struct{}{}
		defer delete(visited, source.ID)
	}

	publicID := types.SnowflakeID(source.PublicID)
	result := types.PublicPost{
		ID:              publicID,
		PublicID:        publicID,
		ParentID:        parentID,
		PostKind:        string(source.PostKind),
		Domain:          string(source.Domain),
		ContentCategory: string(source.ContentCategory),
		ContentableType: source.ContentableType,
		AuthorID:        types.SnowflakeID(source.Author.PublicID),
		Title:           copyLocalized(source.Title),
		Slug:            source.Slug,
		Content:         copyLocalized(source.Content),
		Summary:         copyLocalized(source.Summary),
		Audience:        source.Audience,
		Metadata:        cloneRawJSON(source.Metadata),
		Extras:          cloneRawJSON(source.Extras),
		Author:          projectPublicPostAuthor(source.Author),
		Processed:       source.Processed,
		Published:       source.Published,
		PublishedAt:     source.PublishedAt,
		ContentHidden:   source.ContentHidden,
		ViewedOnce:      source.ViewedOnce,
		ClientID:        source.ClientID,
		CreatedAt:       source.CreatedAt,
		UpdatedAt:       source.UpdatedAt,
	}
	if source.DeletedAt.Valid {
		deletedAt := source.DeletedAt.Time
		result.DeletedAt = &deletedAt
	}

	if source.Parent != nil {
		parentPublicID := types.SnowflakeID(source.Parent.PublicID)
		result.ParentID = &parentPublicID
		projected := projectPublicPost(*source.Parent, nil, visited)
		result.Parent = &projected
	}
	for i := range source.Children {
		result.Children = append(result.Children, projectPublicPost(source.Children[i], &publicID, visited))
	}
	for i := range source.Clusters {
		result.Clusters = append(result.Clusters, projectPublicCluster(source.Clusters[i]))
	}
	for _, attachment := range source.Attachments {
		if attachment != nil && attachment.IsPublic {
			result.Attachments = append(result.Attachments, projectPublicMedia(attachment))
		}
	}
	for _, mention := range source.Mentions {
		if mention == nil || mention.User.PublicID <= 0 {
			continue
		}
		userID := types.SnowflakeID(mention.User.PublicID)
		result.Mentions = append(result.Mentions, types.PublicPostMention{
			ID:     userID,
			UserID: userID,
			User:   projectPublicPostAuthor(mention.User),
		})
	}
	for _, hashtag := range source.Hashtags {
		if hashtag != nil {
			result.Hashtags = append(result.Hashtags, projectPublicHashtag(hashtag, 0))
		}
	}
	for _, poll := range source.Poll {
		if poll != nil {
			result.Poll = append(result.Poll, projectPublicPoll(poll))
		}
	}
	if source.Event != nil {
		result.Event = projectPublicEvent(source.Event, publicID)
	}
	result.Location = projectPublicLocation(source.Location)
	result.Engagements = projectPublicEngagements(source.Engagements)
	return result
}

func projectPublicPostAuthor(source models.User) types.PublicPostAuthor {
	publicID := types.SnowflakeID(source.PublicID)
	return types.PublicPostAuthor{
		ID:              publicID,
		PublicID:        publicID,
		UserName:        source.UserName,
		DisplayName:     source.DisplayName,
		Bio:             copyLocalized(source.Bio),
		Website:         source.Website,
		DateOfBirth:     source.DateOfBirth,
		PrivacyLevel:    string(source.PrivacyLevel),
		IsOnline:        source.IsOnline,
		IsPremium:       source.IsPremium,
		CreatedAt:       source.CreatedAt,
		DefaultLanguage: source.DefaultLanguage,
		Languages:       append([]string(nil), source.Languages...),
		Avatar:          projectOptionalPublicMedia(source.Avatar),
		Cover:           projectOptionalPublicMedia(source.Cover),
	}
}

func projectOptionalPublicMedia(source *media.Media) *types.PublicPostMedia {
	if source == nil || !source.IsPublic {
		return nil
	}
	result := projectPublicMedia(source)
	return &result
}

func projectPublicMedia(source *media.Media) types.PublicPostMedia {
	if source == nil {
		return types.PublicPostMedia{}
	}
	publicID := types.SnowflakeID(source.PublicID)
	return types.PublicPostMedia{
		ID:               publicID,
		PublicID:         publicID,
		OwnerType:        string(source.OwnerType),
		Role:             string(source.Role),
		IsPublic:         source.IsPublic,
		ProcessingStatus: string(source.ProcessingStatus),
		File: types.PublicMediaFile{
			URL:       source.File.URL,
			MimeType:  source.File.MimeType,
			Size:      source.File.Size,
			Width:     source.File.Width,
			Height:    source.File.Height,
			Duration:  source.File.Duration,
			Variants:  marshalRaw(source.File.Variants),
			CreatedAt: source.File.CreatedAt,
		},
		CreatedAt: source.CreatedAt,
		UpdatedAt: source.UpdatedAt,
	}
}

func projectPublicLocation(source *modelutils.Location) *types.PublicPostLocation {
	if source == nil {
		return nil
	}
	result := &types.PublicPostLocation{
		CountryCode: source.CountryCode,
		Address:     source.Address,
		City:        source.City,
		Country:     source.Country,
		Postal:      source.Postal,
		Region:      source.Region,
		Postcode:    source.Postcode,
		ZipCode:     source.ZipCode,
		Province:    source.Province,
		Town:        source.Town,
		Timezone:    source.Timezone,
		Display:     source.Display,
		Latitude:    source.Latitude,
		Longitude:   source.Longitude,
	}
	if source.LocationPoint != nil {
		result.Point = &types.PublicLocationPoint{Lng: source.LocationPoint.Lng, Lat: source.LocationPoint.Lat}
	}
	return result
}

func projectPublicPoll(source *postpayloads.Poll) types.PublicPoll {
	result := types.PublicPoll{
		ID:            types.EncodeOpaqueID("poll", [16]byte(source.ID)),
		Question:      copyLocalizedValue(source.Question),
		Duration:      source.Duration,
		Kind:          string(source.Kind),
		MaxSelectable: source.MaxSelectable,
		CreatedAt:     source.CreatedAt,
		UpdatedAt:     source.UpdatedAt,
	}
	for i := range source.Choices {
		choice := source.Choices[i]
		choiceToken := types.EncodeOpaqueID("pc", [16]byte(choice.ID))
		projected := types.PublicPollChoice{
			ID:           choiceToken,
			DisplayOrder: choice.DisplayOrder,
			Label:        copyLocalizedValue(choice.Label),
			VoteCount:    choice.VoteCount,
		}
		for j := range choice.Votes {
			vote := choice.Votes[j]
			if vote.User.PublicID <= 0 {
				continue
			}
			projected.Votes = append(projected.Votes, types.PublicPollVote{
				ChoiceID:  choiceToken,
				UserID:    types.SnowflakeID(vote.User.PublicID),
				User:      projectPublicPostAuthor(vote.User),
				Weight:    vote.Weight,
				Rank:      vote.Rank,
				CreatedAt: vote.CreatedAt,
			})
		}
		result.Choices = append(result.Choices, projected)
	}
	return result
}

func projectPublicEvent(source *postpayloads.Event, postID types.SnowflakeID) *types.PublicEvent {
	if source == nil {
		return nil
	}
	eventID := types.EncodeOpaqueID("event", [16]byte(source.ID))
	result := &types.PublicEvent{
		ID:            eventID,
		PostID:        postID,
		Title:         copyLocalizedValue(source.Title),
		Description:   copyLocalizedValue(source.Description),
		Kind:          source.Kind,
		StartTime:     source.StartTime,
		EndTime:       source.EndTime,
		Location:      projectPublicLocation(source.Location),
		Capacity:      source.Capacity,
		IsPaid:        source.IsPaid,
		Price:         source.Price,
		Currency:      source.Currency,
		IsOnline:      source.IsOnline,
		OnlineURL:     source.OnlineURL,
		Status:        source.Status,
		GoingCount:    source.GoingCount,
		NotGoingCount: source.NotGoingCount,
		MaybeCount:    source.MaybeCount,
		CreatedAt:     source.CreatedAt,
		UpdatedAt:     source.UpdatedAt,
	}
	for i := range source.Attendees {
		attendee := source.Attendees[i]
		if attendee.UserPublicID <= 0 {
			continue
		}
		userID := types.SnowflakeID(attendee.UserPublicID)
		result.Attendees = append(result.Attendees, types.PublicEventAttendee{
			ID:           userID,
			EventID:      eventID,
			UserPublicID: userID,
			Username:     attendee.Username,
			DisplayName:  attendee.DisplayName,
			AvatarURL:    attendee.AvatarURL,
			Status:       string(attendee.Status),
			JoinedAt:     attendee.JoinedAt,
			UpdatedAt:    attendee.UpdatedAt,
		})
	}
	return result
}

func projectPublicEngagements(source *models.Engagement) *types.PublicPostEngagements {
	if source == nil {
		return nil
	}
	result := &types.PublicPostEngagements{
		Counts:    publicCounts(json.RawMessage(source.Counts)),
		CreatedAt: source.CreatedAt,
		UpdatedAt: source.UpdatedAt,
	}
	for i := range source.EngagementDetails {
		detail := source.EngagementDetails[i]
		if _, ok := publicPostEngagementKinds[detail.Kind]; !ok || detail.Engager.PublicID <= 0 {
			continue
		}
		projected := types.PublicPostEngagementDetail{
			EngagerID: types.SnowflakeID(detail.Engager.PublicID),
			Engager:   projectPublicPostAuthor(detail.Engager),
			Kind:      string(detail.Kind),
			CreatedAt: detail.CreatedAt,
			UpdatedAt: detail.UpdatedAt,
		}
		if detail.Engagee.PublicID > 0 {
			projected.EngageeID = types.SnowflakeID(detail.Engagee.PublicID)
			engagee := projectPublicPostAuthor(detail.Engagee)
			projected.Engagee = &engagee
		}
		result.EngagementDetails = append(result.EngagementDetails, projected)
	}
	return result
}

func publicCounts(source json.RawMessage) map[string]json.RawMessage {
	result := make(map[string]json.RawMessage)
	var values map[string]json.RawMessage
	if len(source) == 0 || json.Unmarshal(source, &values) != nil {
		return result
	}
	for key, value := range values {
		if _, ok := publicPostCountKeys[key]; ok {
			result[key] = cloneRawJSON(value)
		}
	}
	return result
}

func projectPublicCluster(source taxonomy.Cluster) types.PublicPostCluster {
	result := types.PublicPostCluster{
		Domain:          string(source.Domain),
		Name:            copyLocalizedValue(source.Name),
		Slug:            source.Slug,
		Description:     copyLocalized(source.Description),
		IsActive:        source.IsActive,
		MetaTitle:       copyLocalized(source.MetaTitle),
		MetaDescription: copyLocalized(source.MetaDescription),
	}
	for i := range source.Children {
		result.Children = append(result.Children, projectPublicCluster(source.Children[i]))
	}
	for i := range source.Intents {
		intent := source.Intents[i]
		result.Intents = append(result.Intents, types.PublicTaxonomyIntent{
			Domain: string(intent.Domain), Key: string(intent.Key), Label: intent.Label, IsActive: intent.IsActive,
		})
	}
	for i := range source.Entities {
		entity := source.Entities[i]
		result.Entities = append(result.Entities, types.PublicTaxonomyEntity{
			Domain:      string(entity.Domain),
			Type:        string(entity.Type),
			Slug:        entity.Slug,
			Name:        copyLocalizedValue(entity.Name),
			Description: copyLocalized(entity.Description),
			ExternalID:  entity.ExternalID,
			IsActive:    entity.IsActive,
		})
	}
	for i := range source.Synonyms {
		synonym := source.Synonyms[i]
		result.Synonyms = append(result.Synonyms, types.PublicTaxonomySynonym{
			Domain:       string(synonym.Domain),
			Word:         copyLocalizedValue(synonym.Word),
			Slug:         synonym.Slug,
			IsPrimary:    synonym.IsPrimary,
			SearchWeight: synonym.SearchWeight,
		})
	}
	return result
}

func projectPublicHashtag(source *models.Hashtag, depth int) types.PublicPostHashtag {
	if source == nil || depth > 4 {
		return types.PublicPostHashtag{}
	}
	result := types.PublicPostHashtag{Tag: source.Tag, Slug: source.Slug}
	if source.Parent != nil {
		parent := projectPublicHashtag(source.Parent, depth+1)
		result.Parent = &parent
	}
	for _, related := range source.RelatedHashtags {
		result.RelatedHashtags = append(result.RelatedHashtags, projectPublicHashtag(related, depth+1))
	}
	return result
}

func copyLocalized(source *modelutils.LocalizedString) map[string]string {
	if source == nil {
		return nil
	}
	return copyLocalizedValue(*source)
}

func copyLocalizedValue(source modelutils.LocalizedString) map[string]string {
	if len(source) == 0 {
		return nil
	}
	result := make(map[string]string, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneRawJSON(source []byte) json.RawMessage {
	if len(source) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), source...)
}

func marshalRaw(value any) json.RawMessage {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) == "null" {
		return nil
	}
	return encoded
}
