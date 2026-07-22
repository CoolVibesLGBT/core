package types

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var ErrInvalidOpaqueID = errors.New("invalid opaque id")

// PublicPost is the HTTP/query-side representation of a post. Persistence
// UUIDs and account/storage internals deliberately have no field in this type.
// The legacy `id` and `author_id` keys remain available, but contain public
// Snowflake identifiers so existing clients can migrate without receiving
// database identities.
type PublicPost struct {
	ID              SnowflakeID            `json:"id"`
	PublicID        SnowflakeID            `json:"public_id"`
	ParentID        *SnowflakeID           `json:"parent_id,omitempty"`
	Parent          *PublicPost            `json:"parent,omitempty"`
	Children        []PublicPost           `json:"children,omitempty"`
	PostKind        string                 `json:"post_kind"`
	Domain          string                 `json:"domain"`
	ContentCategory string                 `json:"content_category"`
	ContentableType *string                `json:"contentable_type,omitempty"`
	AuthorID        SnowflakeID            `json:"author_id"`
	Title           map[string]string      `json:"title,omitempty"`
	Slug            *string                `json:"slug,omitempty"`
	Content         map[string]string      `json:"content,omitempty"`
	Summary         map[string]string      `json:"summary,omitempty"`
	Audience        *string                `json:"audience,omitempty"`
	Metadata        json.RawMessage        `json:"metadata,omitempty"`
	Extras          json.RawMessage        `json:"extras,omitempty"`
	Author          PublicPostAuthor       `json:"author"`
	Clusters        []PublicPostCluster    `json:"clusters,omitempty"`
	Attachments     []PublicPostMedia      `json:"attachments,omitempty"`
	Mentions        []PublicPostMention    `json:"mentions,omitempty"`
	Hashtags        []PublicPostHashtag    `json:"hashtags,omitempty"`
	Poll            []PublicPoll           `json:"poll,omitempty"`
	Event           *PublicEvent           `json:"event,omitempty"`
	Location        *PublicPostLocation    `json:"location,omitempty"`
	Engagements     *PublicPostEngagements `json:"engagements,omitempty"`
	Processed       bool                   `json:"processed"`
	Published       bool                   `json:"published"`
	PublishedAt     *time.Time             `json:"published_at,omitempty"`
	ContentHidden   bool                   `json:"content_hidden"`
	ViewedOnce      bool                   `json:"viewed_once"`
	ClientID        string                 `json:"client_id,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       *time.Time             `json:"deleted_at"`
}

type PublicPostPage struct {
	Posts  []PublicPost `json:"posts"`
	Cursor *string      `json:"cursor"`
}

type PublicPostAuthor struct {
	ID              SnowflakeID       `json:"id"`
	PublicID        SnowflakeID       `json:"public_id"`
	UserName        string            `json:"username"`
	DisplayName     string            `json:"displayname"`
	Bio             map[string]string `json:"bio,omitempty"`
	Website         string            `json:"website,omitempty"`
	DateOfBirth     *time.Time        `json:"date_of_birth,omitempty"`
	PrivacyLevel    string            `json:"privacy_level"`
	IsOnline        bool              `json:"is_online"`
	IsPremium       bool              `json:"is_premium"`
	CreatedAt       time.Time         `json:"created_at"`
	DefaultLanguage string            `json:"default_language"`
	Languages       []string          `json:"languages,omitempty"`
	Avatar          *PublicPostMedia  `json:"avatar,omitempty"`
	Cover           *PublicPostMedia  `json:"cover,omitempty"`
}

type PublicPostMedia struct {
	ID               SnowflakeID     `json:"id"`
	PublicID         SnowflakeID     `json:"public_id"`
	OwnerType        string          `json:"owner_type"`
	Role             string          `json:"role"`
	IsPublic         bool            `json:"is_public"`
	ProcessingStatus string          `json:"processing_status"`
	File             PublicMediaFile `json:"file"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type PublicMediaFile struct {
	URL       string          `json:"url"`
	MimeType  string          `json:"mime_type"`
	Size      int64           `json:"size"`
	Width     *int            `json:"width,omitempty"`
	Height    *int            `json:"height,omitempty"`
	Duration  *float64        `json:"duration,omitempty"`
	Variants  json.RawMessage `json:"variants,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type PublicPostMediaWithUser struct {
	PublicPostMedia
	User   PublicPostAuthor `json:"user"`
	Cursor *SnowflakeID     `json:"cursor"`
}

type PublicPostMediaPage struct {
	Items        []PublicPostMediaWithUser
	NextPublicID *int64
}

type PublicPostLocation struct {
	CountryCode *string              `json:"country_code"`
	Address     *string              `json:"address,omitempty"`
	City        *string              `json:"city,omitempty"`
	Country     *string              `json:"country,omitempty"`
	Postal      *string              `json:"postal,omitempty"`
	Region      *string              `json:"region,omitempty"`
	Postcode    *string              `json:"postcode,omitempty"`
	ZipCode     *string              `json:"zip_code,omitempty"`
	Province    *string              `json:"province,omitempty"`
	Town        *string              `json:"town,omitempty"`
	Timezone    *string              `json:"timezone,omitempty"`
	Display     *string              `json:"display"`
	Latitude    *float64             `json:"latitude,omitempty"`
	Longitude   *float64             `json:"longitude,omitempty"`
	Point       *PublicLocationPoint `json:"location_point,omitempty"`
}

type PublicLocationPoint struct {
	Lng float64 `json:"lng"`
	Lat float64 `json:"lat"`
}

type PublicPostMention struct {
	ID     SnowflakeID      `json:"id"`
	UserID SnowflakeID      `json:"user_id"`
	User   PublicPostAuthor `json:"user"`
}

type PublicPostHashtag struct {
	Tag             string              `json:"tag"`
	Slug            string              `json:"slug"`
	Parent          *PublicPostHashtag  `json:"parent,omitempty"`
	RelatedHashtags []PublicPostHashtag `json:"related_hashtags,omitempty"`
}

type PublicPostCluster struct {
	Domain          string                  `json:"domain"`
	Name            map[string]string       `json:"name"`
	Slug            string                  `json:"slug"`
	Description     map[string]string       `json:"description,omitempty"`
	IsActive        bool                    `json:"is_active"`
	MetaTitle       map[string]string       `json:"meta_title,omitempty"`
	MetaDescription map[string]string       `json:"meta_description,omitempty"`
	Children        []PublicPostCluster     `json:"children,omitempty"`
	Intents         []PublicTaxonomyIntent  `json:"intents,omitempty"`
	Entities        []PublicTaxonomyEntity  `json:"entities,omitempty"`
	Synonyms        []PublicTaxonomySynonym `json:"synonyms,omitempty"`
}

type PublicTaxonomyIntent struct {
	Domain   string `json:"domain"`
	Key      string `json:"key"`
	Label    string `json:"label"`
	IsActive bool   `json:"is_active"`
}

type PublicTaxonomyEntity struct {
	Domain      string            `json:"domain"`
	Type        string            `json:"type"`
	Slug        string            `json:"slug"`
	Name        map[string]string `json:"name"`
	Description map[string]string `json:"description,omitempty"`
	ExternalID  *string           `json:"external_id,omitempty"`
	IsActive    bool              `json:"is_active"`
}

type PublicTaxonomySynonym struct {
	Domain       string            `json:"domain"`
	Word         map[string]string `json:"word"`
	Slug         string            `json:"slug"`
	IsPrimary    bool              `json:"is_primary"`
	SearchWeight int               `json:"search_weight"`
}

type PublicPoll struct {
	ID            string             `json:"id"`
	Question      map[string]string  `json:"question"`
	Duration      string             `json:"duration"`
	Kind          string             `json:"kind"`
	MaxSelectable int                `json:"max_selectable"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Choices       []PublicPollChoice `json:"choices,omitempty"`
}

type PublicPollChoice struct {
	ID           string            `json:"id"`
	DisplayOrder int               `json:"display_order"`
	Label        map[string]string `json:"label"`
	VoteCount    int               `json:"vote_count"`
	Votes        []PublicPollVote  `json:"votes,omitempty"`
}

type PublicPollVote struct {
	ChoiceID  string           `json:"choice_id"`
	UserID    SnowflakeID      `json:"user_id"`
	User      PublicPostAuthor `json:"user"`
	Weight    int              `json:"weight"`
	Rank      int              `json:"rank"`
	CreatedAt time.Time        `json:"created_at"`
}

type PublicEvent struct {
	ID            string                `json:"id"`
	PostID        SnowflakeID           `json:"post_id"`
	Title         map[string]string     `json:"title"`
	Description   map[string]string     `json:"description"`
	Kind          string                `json:"kind"`
	StartTime     *time.Time            `json:"start_time,omitempty"`
	EndTime       *time.Time            `json:"end_time,omitempty"`
	Location      *PublicPostLocation   `json:"location,omitempty"`
	Capacity      *int                  `json:"capacity,omitempty"`
	IsPaid        bool                  `json:"is_paid"`
	Price         *float64              `json:"price,omitempty"`
	Currency      *string               `json:"currency,omitempty"`
	IsOnline      bool                  `json:"is_online"`
	OnlineURL     *string               `json:"online_url,omitempty"`
	Status        string                `json:"status"`
	GoingCount    int64                 `json:"going_count"`
	NotGoingCount int64                 `json:"not_going_count"`
	MaybeCount    int64                 `json:"maybe_count"`
	Attendees     []PublicEventAttendee `json:"attendees,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

type PublicEventAttendee struct {
	ID           SnowflakeID `json:"id"`
	EventID      string      `json:"event_id"`
	UserPublicID SnowflakeID `json:"user_public_id"`
	Username     string      `json:"username"`
	DisplayName  string      `json:"displayname"`
	AvatarURL    *string     `json:"avatar_url,omitempty"`
	Status       string      `json:"status"`
	JoinedAt     time.Time   `json:"joined_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type PublicPostEngagements struct {
	Counts            map[string]json.RawMessage   `json:"counts"`
	EngagementDetails []PublicPostEngagementDetail `json:"engagement_details,omitempty"`
	CreatedAt         time.Time                    `json:"created_at"`
	UpdatedAt         time.Time                    `json:"updated_at"`
}

type PublicPostEngagementDetail struct {
	EngagerID SnowflakeID       `json:"engager_id"`
	Engager   PublicPostAuthor  `json:"engager,omitempty"`
	EngageeID SnowflakeID       `json:"engagee_id,omitempty"`
	Engagee   *PublicPostAuthor `json:"engagee,omitempty"`
	Kind      string            `json:"kind"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// EncodeOpaqueID converts an internal 16-byte identifier into a typed API
// token. It is intentionally not a database UUID representation.
func EncodeOpaqueID(prefix string, raw [16]byte) string {
	prefix = strings.TrimSpace(prefix)
	return prefix + "_" + base64.RawURLEncoding.EncodeToString(raw[:])
}

func DecodeOpaqueID(prefix, value string) ([16]byte, error) {
	var result [16]byte
	prefix = strings.TrimSpace(prefix) + "_"
	if !strings.HasPrefix(value, prefix) {
		return result, ErrInvalidOpaqueID
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil || len(raw) != len(result) {
		return result, ErrInvalidOpaqueID
	}
	copy(result[:], raw)
	return result, nil
}
