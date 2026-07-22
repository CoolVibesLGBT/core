package types

import (
	"encoding/json"
	"time"
)

// PublicUserSummary is the read model shared by user lookup and global search.
// ID intentionally mirrors PublicID for clients that still use the legacy
// `id` key. Neither field contains the user's database UUID.
type PublicUserSummary struct {
	ID          SnowflakeID         `json:"id"`
	PublicID    SnowflakeID         `json:"public_id"`
	UserName    string              `json:"username"`
	DisplayName string              `json:"displayname"`
	Bio         map[string]string   `json:"bio,omitempty"`
	IsOnline    bool                `json:"is_online"`
	Location    *PublicUserLocation `json:"location,omitempty"`
	Avatar      *PublicUserMedia    `json:"avatar,omitempty"`
}

// PublicUserProfile is the public profile boundary. Account credentials,
// authorization data, wallet state, preference bitsets, broadcast metadata,
// database UUIDs and storage paths are deliberately absent.
type PublicUserProfile struct {
	ID              SnowflakeID           `json:"id"`
	PublicID        SnowflakeID           `json:"public_id"`
	UserName        string                `json:"username"`
	DisplayName     string                `json:"displayname"`
	Bio             map[string]string     `json:"bio,omitempty"`
	Website         string                `json:"website,omitempty"`
	DateOfBirth     *time.Time            `json:"date_of_birth,omitempty"`
	PrivacyLevel    string                `json:"privacy_level"`
	IsOnline        bool                  `json:"is_online"`
	IsPremium       bool                  `json:"is_premium"`
	CreatedAt       time.Time             `json:"created_at"`
	DefaultLanguage string                `json:"default_language"`
	Languages       []string              `json:"languages,omitempty"`
	Hobbies         []string              `json:"hobbies,omitempty"`
	MoviesGenres    []string              `json:"movies_genres,omitempty"`
	TVShowsGenres   []string              `json:"tv_shows_genres,omitempty"`
	TheaterGenres   []string              `json:"theater_genres,omitempty"`
	CinemaGenres    []string              `json:"cinema_genres,omitempty"`
	ArtInterests    []string              `json:"art_interests,omitempty"`
	Entertainment   []string              `json:"entertainment,omitempty"`
	Location        *PublicUserLocation   `json:"location,omitempty"`
	Avatar          *PublicUserMedia      `json:"avatar,omitempty"`
	Cover           *PublicUserMedia      `json:"cover,omitempty"`
	Engagements     PublicUserEngagements `json:"engagements"`
}

type PublicUserLocation struct {
	Display *string `json:"display,omitempty"`
	City    *string `json:"city,omitempty"`
	Region  *string `json:"region,omitempty"`
	Country *string `json:"country,omitempty"`
}

// PublicUserMedia keeps the client-compatible avatar/cover shape without
// exposing media UUIDs, ownership data or file-system metadata.
type PublicUserMedia struct {
	File PublicUserMediaFile `json:"file"`
}

type PublicUserMediaFile struct {
	URL      string          `json:"url,omitempty"`
	Variants json.RawMessage `json:"variants,omitempty"`
}

type PublicUserEngagements struct {
	Counts PublicUserEngagementCounts `json:"counts"`
}

// These are the aggregate counters rendered by the current profile clients.
// Private engagement rows and financial/report counters are not exposed.
type PublicUserEngagementCounts struct {
	FollowerCount        int64 `json:"follower_count"`
	FollowingCount       int64 `json:"following_count"`
	PostCount            int64 `json:"post_count"`
	BlockingCount        int64 `json:"blocking_count"`
	BlockedByCount       int64 `json:"blocked_by_count"`
	LikeGivenCount       int64 `json:"like_given_count"`
	LikeReceivedCount    int64 `json:"like_received_count"`
	DislikeGivenCount    int64 `json:"dislike_given_count"`
	DislikeReceivedCount int64 `json:"dislike_received_count"`
	MatchCount           int64 `json:"match_count"`
	ViewReceivedCount    int64 `json:"view_received_count"`
}
