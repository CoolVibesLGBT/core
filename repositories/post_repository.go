package repositories

import (
	"context"
	"core/constants"
	"core/extensions"
	"core/helpers"
	"core/models"
	"core/models/media"
	"core/models/notifications"
	"core/models/post"
	"core/models/taxonomy"
	"core/models/utils"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	post_payloads "core/models/post/payloads"
	"core/types"
	"mime/multipart"
	"sort"

	"fmt"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PostRepository struct {
	db               *gorm.DB
	snowFlakeNode    *helpers.Node
	mediaRepo        *MediaRepository
	userRepo         *UserRepository
	notificationRepo *NotificationRepository
}

func (r *PostRepository) DB() *gorm.DB {
	return r.db
}

func (r *PostRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewPostRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository, notificationRepo *NotificationRepository) *PostRepository {
	return &PostRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo, notificationRepo: notificationRepo}
}

func (r *PostRepository) CreatePost(post *post.Post) error {
	if post.ID == uuid.Nil {
		post.ID = uuid.New()
	}

	// PublicID için Snowflake tarzı ID veya timestamp tabanlı basit artan ID
	if post.PublicID == 0 {
		post.PublicID = r.snowFlakeNode.Generate().Int64()
	}

	// CreatedAt ve UpdatedAt
	now := time.Now()
	post.CreatedAt = now
	post.UpdatedAt = now

	// GORM ile kaydet
	if err := r.db.Create(post).Error; err != nil {
		return err
	}

	return nil
}

// CreatePoll polls ve seçeneklerini kaydeder
func (r *PostRepository) CreatePoll(poll *post_payloads.Poll) error {
	// Transaction başlat
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Poll kaydet
		if err := tx.Create(poll).Error; err != nil {
			return err
		}
		/*
			// PollChoice'ları kaydet
			for i := range poll.Choices {
				poll.Choices[i].PollID = poll.ID
				fmt.Println("ANKET SECIM", poll.Choices[i].Label, poll.Choices[i].ID, poll.ID)
				if err := tx.Create(&poll.Choices[i]).Error; err != nil {
					return err
				}
			}
		*/

		return nil
	})
}

func (r *PostRepository) CreateEvent(event *post_payloads.Event) error {
	return r.db.Create(event).Error
}

func (r *PostRepository) GetPostByIDEx(id uuid.UUID) (*post.Post, error) {
	var p post.Post

	err := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices").
		Preload("Event").
		Preload("Event.Location").
		Preload("Author").
		Preload("Tags").
		Preload("Attachments").
		Preload("Children").
		Preload("Children.Location").
		Preload("Children.Poll").
		Preload("Children.Poll.Choices").
		Preload("Children.Event").
		Preload("Children.Event.Location").
		Preload("Children.Author").
		Preload("Children.Tags").
		Preload("Children.Attachments").
		First(&p, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("post with id %s not found", id)
		}
		return nil, err
	}

	return &p, nil
}

func (r *PostRepository) GetPostByID(id uuid.UUID) (*post.Post, error) {
	var ids []uuid.UUID
	cte := `
		WITH RECURSIVE post_tree AS (
			SELECT id
			FROM posts
			WHERE id = ?
			UNION ALL
			SELECT p.id
			FROM posts p
			INNER JOIN post_tree pt ON pt.id = p.parent_id
		)
		SELECT id FROM post_tree;
	`
	if err := r.db.Raw(cte, id).Scan(&ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("post with id %s not found", id)
	}

	var posts []post.Post
	if err := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("id IN ?", ids).
		Order("created_at ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("no posts found for %s", id)
	}

	postMap := make(map[uuid.UUID]*post.Post, len(posts))
	for i := range posts {
		posts[i].Children = nil // temizle
		postMap[posts[i].ID] = &posts[i]
	}

	var buildTree func(parent *post.Post)
	buildTree = func(parent *post.Post) {
		for _, p := range posts {
			if p.ParentID != nil && *p.ParentID == parent.ID {
				child := postMap[p.ID]
				buildTree(child)
				parent.Children = append(parent.Children, *child)
			}
		}
	}

	root, ok := postMap[id]
	if !ok {
		return nil, fmt.Errorf("post with id %s not found in postMap", id)
	}

	buildTree(root)

	var sortChildren func(p *post.Post)
	sortChildren = func(p *post.Post) {
		sort.SliceStable(p.Children, func(i, j int) bool {
			if p.Children[i].PublicID != p.Children[j].PublicID {
				return p.Children[i].PublicID < p.Children[j].PublicID
			}
			return p.Children[i].CreatedAt.Before(p.Children[j].CreatedAt)
		})
		for i := range p.Children {
			sortChildren(&p.Children[i])
		}
	}
	sortChildren(root)

	return root, nil
}

func (r *PostRepository) GetPostBySlug(filters types.Filter) (*post.Post, error) {
	var ids []uuid.UUID

	cte := `
	WITH RECURSIVE post_tree AS (
		SELECT id
		FROM posts
		WHERE slug = ?
		UNION ALL
		SELECT p.id
		FROM posts p
		INNER JOIN post_tree pt ON pt.id = p.parent_id
	)
	SELECT id FROM post_tree;
	`

	if err := r.db.Raw(cte, filters.Slug).Scan(&ids).Error; err != nil {

		return nil, err
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("post with slug %s not found", *filters.Slug)
	}

	var posts []post.Post
	if err := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("id IN ?", ids).
		Order("created_at ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("no posts found for slug %s", *filters.Slug)
	}

	// map oluştur
	postMap := make(map[uuid.UUID]*post.Post, len(posts))
	for i := range posts {
		posts[i].Children = nil
		postMap[posts[i].ID] = &posts[i]
	}

	// tree build
	var buildTree func(parent *post.Post)
	buildTree = func(parent *post.Post) {
		for _, p := range posts {
			if p.ParentID != nil && *p.ParentID == parent.ID {
				child := postMap[p.ID]
				buildTree(child)
				parent.Children = append(parent.Children, *child)
			}
		}
	}

	// root bul (slug'a karşılık gelen)
	var root *post.Post
	for _, p := range posts {
		if p.Slug != nil && filters.Slug != nil && *p.Slug == *filters.Slug {
			root = postMap[p.ID]
			break
		}

	}

	if root == nil {
		return nil, fmt.Errorf("root post not found for slug %s", *filters.Slug)
	}

	buildTree(root)

	// children sort
	var sortChildren func(p *post.Post)
	sortChildren = func(p *post.Post) {
		sort.SliceStable(p.Children, func(i, j int) bool {
			if p.Children[i].PublicID != p.Children[j].PublicID {
				return p.Children[i].PublicID < p.Children[j].PublicID
			}
			return p.Children[i].CreatedAt.Before(p.Children[j].CreatedAt)
		})
		for i := range p.Children {
			sortChildren(&p.Children[i])
		}
	}

	sortChildren(root)

	return root, nil
}

func (r *PostRepository) GetPostByPublicID(id int64) (*post.Post, error) {
	var p post.Post

	err := r.db.
		First(&p, "public_id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("post with id %d not found", id)
		}
		return nil, err
	}

	return r.GetPostByID(p.ID)
}

func (r *PostRepository) GetPostByIDWithoutRelations(id uuid.UUID) (*post.Post, error) {
	var p post.Post

	err := r.db.
		Preload("Author").
		Where("id = ?", id).
		First(&p).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("post with id %s not found", id)
		}
		return nil, err
	}

	return &p, nil
}

func (r *PostRepository) GetTimeline(filters types.Filter) (types.TimelineResult, error) {
	var posts []post.Post

	query := r.db.Model(&post.Post{}).
		//Where("published = ?", true).
		Where("contentable_type IN ?", []string{string(post.PostKindPost), string(post.PostKindNews), string(post.PostKindStatus), string(post.PostKindVideo)}).
		Where("parent_id IS NULL").
		Order("public_id DESC").
		Limit(filters.Limit).
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Hashtags").
		Preload("Mentions").
		Preload("Attachments").
		Preload("Attachments.File")

	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return types.TimelineResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		s := strconv.FormatInt(int64(posts[len(posts)-1].PublicID), 10)
		nextCursor = &s
	}

	return types.TimelineResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func (r *PostRepository) GetTimelineVibes(filters types.Filter) (types.TimelineResult, error) {
	var posts []post.Post

	query := r.db.Model(&post.Post{}).
		//	Joins("INNER JOIN medias ON medias.owner_id = posts.id AND medias.owner_type = ?", "post").
		Preload("Author").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("contentable_type IN ?", []string{string(post.PostKindPost), string(post.PostKindStatus)}).
		//Where("published = ?", true).
		Order("posts.public_id DESC").
		Limit(filters.Limit).
		Group("posts.id")

	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil {
		query = query.Where("posts.public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return types.TimelineResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		s := strconv.FormatInt(int64(posts[len(posts)-1].PublicID), 10)
		nextCursor = &s
	}

	return types.TimelineResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func (r *PostRepository) GetPostsByKind(filters types.Filter) (types.PostsResult, error) {
	var posts []post.Post

	query := r.db.Model(&post.Post{}).
		//Where("published = ?", true).
		Where("contentable_type IN ?", []string{string(filters.PostKind)}).
		Where("parent_id IS NULL").
		Order("public_id DESC").
		Limit(filters.Limit).
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Hashtags").
		Preload("Mentions").
		Preload("Attachments").
		Preload("Attachments.File")

	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return types.PostsResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		s := strconv.FormatInt(int64(posts[len(posts)-1].PublicID), 10)
		nextCursor = &s
	}

	return types.PostsResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func (r *PostRepository) FindPostsByKind(filters types.Filter) (types.PostsResult, error) {
	var posts []post.Post

	fmt.Println("DOMAIN", *filters.Domain)
	query := r.db.Model(&post.Post{}).
		Where("published = ?", true).
		Where("domain = ?", *filters.Domain).
		Where("contentable_type IN ?", []string{string(filters.PostKind), string(post.PostKindVideo), string(post.PostKindStatus), string(post.PostKindPost), string(post.PostKindNews), string(post.PostKindClassified), string(post.PostKindPlace)}).
		Where("parent_id IS NULL").
		Order("public_id DESC").
		Limit(filters.Limit).
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices", func(db *gorm.DB) *gorm.DB {
			return db.Order("display_order ASC")
		}).
		Preload("Poll.Choices.Votes").
		Preload("Poll.Choices.Votes.User").
		Preload("Poll.Choices.Votes.User.Avatar").
		Preload("Poll.Choices.Votes.User.Avatar.File").
		Preload("Engagements").
		Preload("Engagements.EngagementDetails").
		Preload("Engagements.EngagementDetails.Engager").
		Preload("Engagements.EngagementDetails.Engagee").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Hashtags").
		Preload("Mentions").
		Preload("Attachments").
		Preload("Attachments.File")

	if filters.Search != nil && *filters.Search != "" {
		search := "%" + *filters.Search + "%"
		query = query.Where(
			"title::text ILIKE ? OR content::text ILIKE ?",
			search, search,
		)
	}
	query = applyTaxonomyCategoryFilter(query, filters.Category)

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return types.PostsResult{}, err
	}

	var nextCursor *string
	if len(posts) > 0 {
		s := strconv.FormatInt(int64(posts[len(posts)-1].PublicID), 10)
		nextCursor = &s
	}

	return types.PostsResult{
		Posts:  posts,
		Cursor: nextCursor,
	}, nil
}

func (r *PostRepository) GetUserPosts(userId uuid.UUID, filters types.Filter) ([]post.Post, error) {
	var posts []post.Post

	query := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Avatar").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("author_id = ? AND parent_id IS NULL and contentable_type = ?", userId, filters.PostKind).
		Order("public_id DESC").
		Limit(filters.Limit)

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepository) GetUserPostReplies(filters types.Filter) ([]post.Post, error) {
	var posts []post.Post

	query := r.db.
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices").
		Preload("Event").
		Preload("Parent").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Author").
		Preload("Author.Cover").
		Preload("Author.Avatar").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("author_id = ? AND parent_id IS NOT NULL and contentable_type = ? ", filters.UserUUID, filters.PostKind).
		Order("public_id DESC").
		Limit(filters.Limit)

	// Cursor varsa sadece daha eski postlar
	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepository) GetUserMedias(filters types.Filter) ([]types.MediaWithUser, *int64, error) {
	var medias []media.Media

	query := r.db.Unscoped().
		Preload("File").
		Where("user_id = ?", filters.UserUUID).
		Order("public_id DESC").
		Limit(filters.Limit)

	if filters.Cursor != nil {
		query = query.Where("public_id < ?", *filters.Cursor)
	}

	if err := query.Find(&medias).Error; err != nil {
		return nil, nil, err
	}

	// userID'leri topla
	userIDs := make([]uuid.UUID, 0, len(medias))
	for _, m := range medias {
		userIDs = append(userIDs, m.UserID)
	}

	var users []models.User
	if len(userIDs) > 0 {
		if err := r.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, nil, err
		}
	}

	userMap := make(map[uuid.UUID]models.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}

	// Sonuçları MediaWithUser tipine dönüştür
	results := make([]types.MediaWithUser, 0, len(medias))
	for _, m := range medias {
		results = append(results, types.MediaWithUser{
			Media: m,
			User:  userMap[m.UserID],
		})
	}

	var lastCursor *int64 = nil
	if len(medias) > 0 {
		lastCursor = &medias[len(medias)-1].PublicID
	}

	return results, lastCursor, nil
}

func (r *PostRepository) GetRecentHashtags(filters types.Filter) ([]types.HashtagStats, error) {
	var results []types.HashtagStats
	cutoff := time.Now().Add(-48 * time.Hour)

	err := r.db.Model(&models.Hashtag{}).
		Preload("RelatedHashtags").
		Select("tag, COUNT(*) as count").
		Where("created_at >= ?", cutoff).
		Group("tag").
		Order("count DESC").
		Limit(filters.Limit).
		Scan(&results).Error

	return results, err
}

func (r *PostRepository) CreateContentablePost(ctx context.Context, request map[string][]string, files []*multipart.FileHeader, author *models.User, contentableType string, contentableID *uuid.UUID) (*post.Post, error) {
	type PollForm struct {
		ID            string   `form:"id"`
		Question      string   `form:"question"`
		Duration      string   `form:"duration"`
		Kind          string   `form:"kind"`           // single, multiple, ranked, weighted
		MaxSelectable string   `form:"max_selectable"` // only for multiple, weighted
		Options       []string `form:"options"`        // seçenekler
		InitialRanks  []string `form:"ranks"`          // ranked için: her option'ın başlangıç sırası
		Weights       []string `form:"weights"`        // only for weighted, optional
	}

	type PostForm struct {
		ParentId string     `form:"parentPostId"`
		Title    string     `form:"title"`
		Slug     string     `form:"slug"`
		Summary  string     `form:"summary"`
		Language string     `form:"language"`
		Content  string     `form:"content"`
		Audience string     `form:"audience"`
		Hashtags []string   `form:"hashtags[]"`
		Mentions []string   `form:"mentions[]"`
		Polls    []PollForm `form:"polls"`

		EventTitle       string `form:"event[title]"`
		EventDescription string `form:"event[description]"`
		EventDate        string `form:"event[date]"`
		EventTime        string `form:"event[time]"`
		EventKind        string `form:"event[kind]"`
		EventCapacity    string `form:"event[capacity]"`
		EventIsPaid      string `form:"event[is_paid]"`
		EventPrice       string `form:"event[price]"`
		EventCurrency    string `form:"event[currency]"`
		EventIsOnline    string `form:"event[is_online]"`
		EventIsOnlineURL string `form:"event[online_url]"`

		LocationAddress string            `form:"location[address]"`
		LocationLat     float64           `form:"location[lat]"`
		LocationLng     float64           `form:"location[lng]"`
		CountryCode     string            `form:"location[country_code]"`
		Region          string            `form:"location[region]"`
		City            string            `form:"location[city]"`
		ZipCode         string            `form:"location[zip_code]"`
		Province        string            `form:"location[province]"`
		Town            string            `form:"location[town]"`
		Postcode        string            `form:"location[postcode]"`
		Country         string            `form:"location[country]"`
		Extras          map[string]string `form:"extras"`
	}

	decoder := form.NewDecoder()
	postForm := PostForm{}

	if err := decoder.Decode(&postForm, request); err != nil {
		return nil, err
	}

	tx := r.DB().Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var parentUUID *uuid.UUID
	var parentPost *post.Post
	if len(postForm.ParentId) > 0 {
		parentIDInt, err := strconv.ParseInt(postForm.ParentId, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid parentId %s: %w", postForm.ParentId, err)
		}
		parentPost, err = r.FindPostByPublicID(parentIDInt)
		if err == nil {
			parentUUID = &parentPost.ID
		}
	}

	defaultLanguage := helpers.DefaultIfEmpty(postForm.Language, author.DefaultLanguage)

	var postKindType post.PostKind
	var isPublished = false
	switch contentableType {
	case "chat":
		postKindType = post.PostKindMessage
		isPublished = true
	case "post":
		postKindType = post.PostKindPost
		isPublished = true
	case "event":
		postKindType = post.PostKindEvent
		isPublished = true
	case "status":
		postKindType = post.PostKindStatus
		isPublished = true
	case "classified":
		postKindType = post.PostKindClassified
		isPublished = true
	case "job_offer":
		postKindType = post.PostKindJobOffer
		isPublished = true
	case "job_search":
		postKindType = post.PostKindJobSearch
		isPublished = true
	case "news":
		postKindType = post.PostKindNews
		isPublished = false
	case "place":
		postKindType = post.PostKindPlace
		isPublished = true
	case "checkin":
		postKindType = post.PostKindCheckIn
		isPublished = true
	case "video":
		postKindType = post.PostKindVideo
		isPublished = true
	default:
		postKindType = post.PostKindStatus
	}

	postForm.Slug = helpers.GenerateSlug(
		func() string {
			if postForm.Slug != "" {
				return postForm.Slug
			}
			return postForm.Title
		}(),
	)
	newPost := &post.Post{
		ID:              uuid.New(),
		ParentID:        parentUUID,
		PublicID:        r.Node().Generate().Int64(),
		AuthorID:        author.ID,
		Published:       isPublished,
		Domain:          author.Domain,
		PostKind:        postKindType,
		ContentCategory: post.ContentNormal,
		Title:           utils.MakeLocalizedString(defaultLanguage, postForm.Title),
		Content:         utils.MakeLocalizedString(defaultLanguage, postForm.Content),
		Summary:         utils.MakeLocalizedString(defaultLanguage, postForm.Summary),
		Slug:            &postForm.Slug,
		Audience:        &postForm.Audience,
		ContentableType: &contentableType,
		ContentableID:   contentableID,
	}

	if err := tx.Create(newPost).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for _, f := range files {
		var ownerType media.OwnerType
		var role media.MediaRole

		switch contentableType {
		case "chat":
			ownerType = media.OwnerChat
			role = media.RoleChatMedia
		case "post":
			ownerType = media.OwnerPost
			role = media.RolePost
		case "news":
			ownerType = media.OwnerNews
			role = media.RolePost
		case "video":
			ownerType = media.OwnerVideo
			role = media.RoleVideo
		default:
			ownerType = media.OwnerPost
			role = media.RolePost
		}

		mediaModel, err := r.mediaRepo.AddMedia(newPost.ID, ownerType, author.ID, role, f)
		if err != nil {
			tx.Rollback()
			return nil, err
		}
		newPost.Attachments = append(newPost.Attachments, mediaModel)
	}

	// Polls ekleme
	for _, pollInfo := range postForm.Polls {

		maxSelectable := 1
		if len(pollInfo.MaxSelectable) > 0 {
			if v, err := strconv.Atoi(pollInfo.MaxSelectable); err == nil {
				maxSelectable = v
			}
		}
		pollKind := post_payloads.PollKindSingle
		if len(pollInfo.Kind) > 0 {
			pollKind = post_payloads.PollKind(pollInfo.Kind)
		}

		if len(pollInfo.Question) == 0 {
			tx.Rollback()
			return nil, errors.New(constants.ErrPollTitleEmpty.String())
		}

		poll := &post_payloads.Poll{
			ID:              uuid.New(),
			ContentableID:   newPost.ID,
			ContentableType: post_payloads.ContentablePollPost,
			Question:        *utils.MakeLocalizedString(defaultLanguage, pollInfo.Question),
			Duration:        pollInfo.Duration,
			Kind:            pollKind,
			MaxSelectable:   maxSelectable,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}

		for index, choiceLabel := range pollInfo.Options {
			if len(choiceLabel) == 0 {
				tx.Rollback()
				return nil, errors.New(constants.ErrPollOptionsEmpty.String())
			}

			poll.Choices = append(poll.Choices, post_payloads.PollChoice{
				ID:           uuid.New(),
				DisplayOrder: index,
				PollID:       poll.ID,
				Label:        *utils.MakeLocalizedString(defaultLanguage, choiceLabel),
				VoteCount:    0,
			})
		}
		if err := r.CreatePoll(poll); err != nil {
			tx.Rollback()
			return nil, err
		}
		newPost.Poll = append(newPost.Poll, poll)
	}

	// Location
	var locationPoint *extensions.PostGISPoint = nil
	if postForm.LocationLat != 0 && postForm.LocationLng != 0 {
		locationPoint = &extensions.PostGISPoint{
			Lat: postForm.LocationLat,
			Lng: postForm.LocationLng,
		}
		locationPost := &utils.Location{
			ID:              uuid.New(),
			ContentableType: utils.LocationOwnerPost,
			ContentableID:   newPost.ID,
			Address:         &postForm.LocationAddress,
			Latitude:        &postForm.LocationLat,
			Longitude:       &postForm.LocationLng,
			CountryCode:     &postForm.CountryCode,
			Region:          &postForm.Region,
			City:            &postForm.City,
			ZipCode:         &postForm.ZipCode,
			Province:        &postForm.Province,
			Town:            &postForm.Town,
			Postcode:        &postForm.Postcode,
			Country:         &postForm.Country,
			LocationPoint:   locationPoint,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := r.userRepo.UpsertLocation(locationPost); err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Event
	if len(postForm.EventTitle) > 0 {
		startTime := time.Time{}
		if len(postForm.EventDate) > 0 && len(postForm.EventTime) > 0 {
			if parsedTime, err := time.Parse("2006-01-02 15:04", postForm.EventDate+" "+postForm.EventTime); err == nil {
				startTime = parsedTime
			}
		}

		isPaid, _ := strconv.ParseBool(postForm.EventIsPaid)
		isOnline, _ := strconv.ParseBool(postForm.EventIsOnline)
		var pricePtr *float64
		if postForm.EventPrice != "" {
			price, _ := strconv.ParseFloat(postForm.EventPrice, 64)
			pricePtr = &price
		}

		evt := &post_payloads.Event{
			ID:          uuid.New(),
			PostID:      newPost.ID,
			Title:       *utils.MakeLocalizedString(defaultLanguage, postForm.EventTitle),
			Description: *utils.MakeLocalizedString(defaultLanguage, postForm.EventDescription),
			Kind:        postForm.EventKind,
			IsPaid:      isPaid,
			Price:       pricePtr,
			Currency:    &postForm.EventCurrency,
			IsOnline:    isOnline,
			OnlineURL:   &postForm.EventIsOnlineURL,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			StartTime:   &startTime,
		}

		if err := tx.Create(evt).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		locationEvent := &utils.Location{
			ID:              uuid.New(),
			ContentableType: utils.LocationOwnerEvent,
			ContentableID:   evt.ID,
			Address:         &postForm.LocationAddress,
			Latitude:        &postForm.LocationLat,
			Longitude:       &postForm.LocationLng,
			LocationPoint:   locationPoint,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := tx.Create(locationEvent).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		evt.Location = locationEvent
		newPost.Event = evt
	}

	// Mentions
	for _, mentionText := range postForm.Mentions {
		mentionUser, err := r.userRepo.GetUserByNameOrEmailOrNickname(mentionText)
		if err == nil {
			mentionItem := models.Mention{
				ID:     uuid.New(),
				UserID: mentionUser.ID,
			}
			newPost.Mentions = append(newPost.Mentions, &mentionItem)
		}
	}

	var hashtagItems []*models.Hashtag
	seen := make(map[string]bool)

	for _, raw := range postForm.Hashtags {
		normalized := helpers.SlugifyStrict(strings.TrimPrefix(raw, "#"))

		if normalized == "" {
			continue
		}

		if seen[normalized] {
			continue
		}
		seen[normalized] = true

		hashtagItem := &models.Hashtag{
			Domain: author.Domain,
			ID:     uuid.New(),
			Tag:    normalized,
			Slug:   helpers.GenerateSlug(normalized),
		}

		hashtagItems = append(hashtagItems, hashtagItem)
	}
	for i := range hashtagItems {
		for j := range hashtagItems {
			if i != j {
				hashtagItems[i].RelatedHashtags =
					append(hashtagItems[i].RelatedHashtags, hashtagItems[j])
			}
		}
	}
	newPost.Hashtags = hashtagItems

	for _, hashtagStr := range postForm.Hashtags {
		hashtagStr := helpers.SlugifyStrict(hashtagStr)
		hashtagItem := models.Hashtag{
			Domain: author.Domain,
			ID:     uuid.New(),
			Tag:    hashtagStr,
			Slug:   helpers.GenerateSlug(hashtagStr),
		}
		newPost.Hashtags = append(newPost.Hashtags, &hashtagItem)
	}

	if len(postForm.Extras) > 0 {
		extras := make(map[string]any)

		for key, val := range postForm.Extras {
			var parsed any
			if err := json.Unmarshal([]byte(val), &parsed); err == nil {
				extras[key] = parsed
			}
		}

		extrasBytes, err := json.Marshal(extras)
		if err == nil {
			newPost.Extras = datatypes.JSON(extrasBytes)
		}
	}

	if err := tx.Save(newPost).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if parentPost != nil {
		err := r.userRepo.engagementRepo.AddEngagement(context.Background(), author.ID, parentPost.AuthorID, models.EngagementKindComment, parentPost.ID, models.EngagementContentableTypePost)
		if err != nil {
			return nil, err
		}
	}

	return newPost, nil
}

func (r *PostRepository) Vote(ctx context.Context, choiceId uuid.UUID, weight int, rank int, userId uuid.UUID) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 1) Choice mevcut mu kontrol et
	var choice post_payloads.PollChoice
	if err := tx.WithContext(ctx).
		First(&choice, "id = ?", choiceId).Error; err != nil {

		tx.Rollback()
		return fmt.Errorf("choice not found: %w", err)
	}

	// 2) Kullanıcının mevcut oyu var mı?
	var existingVote post_payloads.PollVote
	err := tx.WithContext(ctx).
		Where("choice_id = ? AND user_id = ?", choiceId, userId).
		First(&existingVote).Error

	// Vote zaten varsa -> sil (toggle off)
	if err == nil {
		if err := tx.WithContext(ctx).Delete(&existingVote).Error; err != nil {
			tx.Rollback()
			return err
		}

		// VoteCount azalt
		if err := tx.WithContext(ctx).
			Model(&post_payloads.PollChoice{}).
			Where("id = ?", choiceId).
			UpdateColumn("vote_count", gorm.Expr("vote_count - 1")).Error; err != nil {

			tx.Rollback()
			return err
		}

		return tx.Commit().Error
	}

	// Eğer hata "record not found" değilse gerçek hata
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return err
	}

	// 3) Vote yoksa -> yeni oy ekle (toggle on)
	newVote := post_payloads.PollVote{
		ID:       uuid.New(),
		ChoiceID: choiceId,
		UserID:   userId,
		Weight:   weight,
		Rank:     rank,
	}

	if err := tx.WithContext(ctx).Create(&newVote).Error; err != nil {
		tx.Rollback()
		return err
	}

	// VoteCount artır
	if err := tx.WithContext(ctx).
		Model(&post_payloads.PollChoice{}).
		Where("id = ?", choiceId).
		UpdateColumn("vote_count", gorm.Expr("vote_count + 1")).Error; err != nil {

		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (r *PostRepository) FindPostByPublicID(id int64) (*post.Post, error) {
	var p post.Post
	err := r.db.
		First(&p, "public_id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("post with id %d not found", id)
		}
		return nil, err
	}
	return &p, nil
}

func (r *PostRepository) ExistsBySlug(filters types.Filter) (bool, error) {
	var count int64
	err := r.db.Model(&post.Post{}).
		Where("slug = ?", filters.Search).
		Where("post_kind = ?", filters.PostKind).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *PostRepository) SendNotification(ctx context.Context, senderUserUUID uuid.UUID, receiverUserUUID uuid.UUID, notificationPayload notifications.NotificationPayload) error {
	canSendNotification := true
	senderUser, err := r.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: senderUserUUID})
	if err != nil {
		canSendNotification = false
	}
	receiverUser, err := r.userRepo.GetUserByUUIDdWithoutRelations(types.Filter{Context: ctx, UserUUID: receiverUserUUID})
	if err != nil {
		canSendNotification = false
	}

	if canSendNotification {
		err = r.notificationRepo.SendNotificationToUser(*senderUser, *receiverUser, notifications.NotificationTypeReferral, notificationPayload.Title, notificationPayload.Body, notificationPayload)
		if err != nil {
			fmt.Printf("Bildirim gönderilemedi user %s -> %s: %v\n", senderUserUUID, receiverUserUUID, err)
			return err
		}
	}
	return nil
}
func (r *PostRepository) Like(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindLikeReceived, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}

		if isOk {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Like",
					Body:  "Someone liked your post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Liked",
					Body:  "You liked this post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Like Removed",
					Body:  "Someone removed their like",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Like Removed",
					Body:  "You removed your like",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		}
	}
	return nil
}

func (r *PostRepository) Dislike(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindDisLikeReceived, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}

		if isOk {
			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Dislike",
					Body:  "Someone disliked your post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Disliked",
					Body:  "You disliked this post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {
			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Dislike Removed",
					Body:  "Someone removed their dislike",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Dislike Removed",
					Body:  "You removed your dislike",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		}
	}
	return nil
}

func (r *PostRepository) Banana(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindBanana, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}
		if isOk {
			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Banana 🍌",
					Body:  "Someone sent you a banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Banana Sent 🍌",
					Body:  "You sent a banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Banana Removed",
					Body:  "Someone removed their banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Banana Removed",
					Body:  "You removed your banana",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}
		}
	}
	return nil
}

func (r *PostRepository) Report(ctx context.Context, postId int64, kind string, description string, authUser *models.User) error {
	post, err := r.FindPostByPublicID(postId)
	if err != nil {
		return err
	}

	if post != nil {
		report := models.Report{
			ContentableID:   post.ID,
			ContentableType: models.EngagementContentableTypePost,
			ReporterID:      authUser.ID,
			ReportKindKey:   kind,
			Reason:          description,
			Status:          "pending",
		}

		err = r.userRepo.engagementRepo.AddEngagement(context.Background(), authUser.ID, post.AuthorID, models.EngagementKindReport, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}

		if err := r.db.WithContext(ctx).Create(&report).Error; err != nil {
			return err
		}
	}
	return nil
}

func (r *PostRepository) Bookmark(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		isOk, err := r.userRepo.engagementRepo.ToggleEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindBookmark, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}

		if isOk {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "New Bookmark",
					Body:  "Someone bookmarked your post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Post Bookmarked",
					Body:  "You bookmarked this post",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		} else {

			if err := r.SendNotification(
				filters.Context,
				filters.AuthUser.ID,
				post.AuthorID,
				notifications.NotificationPayload{
					Title: "Bookmark Removed",
					Body:  "Someone removed their bookmark",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

			if err := r.SendNotification(
				filters.Context,
				post.AuthorID,
				filters.AuthUser.ID,
				notifications.NotificationPayload{
					Title: "Bookmark Removed",
					Body:  "You removed your bookmark",
				},
			); err != nil {
				helpers.Println("notification error:", err)
			}

		}

	}
	return nil
}

func (r *PostRepository) View(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}
	if post != nil {
		err = r.userRepo.engagementRepo.AddEngagement(filters.Context, filters.AuthUser.ID, post.AuthorID, models.EngagementKindView, post.ID, models.EngagementContentableTypePost)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostRepository) Delete(filters types.Filter) error {
	post, err := r.FindPostByPublicID(filters.PostID)
	if err != nil {
		return err
	}

	if post == nil {
		return errors.New(constants.ErrPostNotFound.String())
	}

	if post.AuthorID != filters.AuthUser.ID {
		return errors.New(constants.ErrPostDeleteDenied.String())
	}

	allowed := post.AuthorID == filters.AuthUser.ID ||
		filters.AuthUser.UserRole == constants.UserRoleModerator ||
		filters.AuthUser.UserRole == constants.UserRoleAdmin ||
		filters.AuthUser.UserRole == constants.UserRoleSuperAdmin

	if !allowed {
		return errors.New(constants.ErrPostDeleteDenied.String())
	}

	// If the post is a comment (has a parent), decrement the parent's comment count.
	if post.ParentID != nil {
		var parentEngagement models.Engagement
		// Find the engagement record of the parent post
		err := r.userRepo.engagementRepo.DB().Where("contentable_id = ? AND contentable_type = ?", *post.ParentID, models.EngagementContentableTypePost).First(&parentEngagement).Error

		if err == nil {
			// Found the parent's engagement, now find a specific detail to remove.
			var detailToRemove models.EngagementDetail
			err := r.userRepo.engagementRepo.DB().Where("engagement_id = ? AND engager_id = ? AND kind = ?", parentEngagement.ID, post.AuthorID, models.EngagementKindComment).First(&detailToRemove).Error

			if err == nil {
				// We found an engagement detail to remove. Let's remove it.
				// This will also trigger the counter decrement.
				errRemove := r.userRepo.engagementRepo.RemoveEngagementDetail(filters.Context, detailToRemove.ID)
				if errRemove != nil {
					helpers.Println("Failed to remove engagement detail during comment deletion:", errRemove)
				}
			} else {
				helpers.Println("Could not find comment engagement detail to remove for comment:", post.ID)
			}
		} else {
			helpers.Println("Could not find parent engagement for comment:", post.ID)
		}
	}

	// 3. Soft delete
	if err := r.db.WithContext(filters.Context).Delete(&post).Error; err != nil {
		return errors.New(constants.ErrPostDeleteFailed.String())
	}
	return nil
}

func (r *PostRepository) Tip(ctx context.Context, postId int64, authUser *models.User, amount decimal.Decimal) (*decimal.Decimal, error) {
	// Postu bul
	post, err := r.FindPostByPublicID(postId)
	if err != nil {
		return &authUser.Balance, err
	}
	if post == nil {
		return &authUser.Balance, errors.New(constants.ErrPostNotFound.String())
	}

	if post.AuthorID == authUser.ID {
		return &authUser.Balance, errors.New(constants.ErrCannotTipOwnPost.String()) // veya özel hata: "Cannot tip own post"
	}

	if amount.Cmp(decimal.Zero) <= 0 {
		return &authUser.Balance, errors.New(constants.ErrInvalidAmount.String()) // veya uygun başka hata
	}

	minAmount := decimal.NewFromFloat(0.01)
	if amount.Cmp(minAmount) < 0 {
		return &authUser.Balance, errors.New(constants.ErrInvalidAmount.String()) // veya uygun başka hata
	}

	if !authUser.Balance.GreaterThanOrEqual(amount) {
		return &authUser.Balance, errors.New(constants.ErrInsufficientBalance.String())
	}

	tx := r.db.Begin()
	if tx.Error != nil {
		return &authUser.Balance, tx.Error
	}

	authUser.Balance = authUser.Balance.Sub(amount)
	if err := tx.Model(&models.User{}).Where("id = ?", authUser.ID).Update("balance", authUser.Balance).Error; err != nil {
		tx.Rollback()
		return &authUser.Balance, err
	}

	var postAuthor models.User
	if err := tx.Where("id = ?", post.AuthorID).First(&postAuthor).Error; err != nil {
		tx.Rollback()
		return &authUser.Balance, err
	}
	postAuthor.Balance = postAuthor.Balance.Add(amount)
	if err := tx.Model(&models.User{}).Where("id = ?", post.AuthorID).Update("balance", postAuthor.Balance).Error; err != nil {
		tx.Rollback()
		return &authUser.Balance, err
	}

	err = r.userRepo.engagementRepo.AddTip(ctx, authUser.ID, post.AuthorID, amount, post.ID, models.EngagementContentableTypePost, models.EngagementKindTip)
	if err != nil {
		tx.Rollback()
		return &authUser.Balance, err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return &authUser.Balance, err
	}

	return &authUser.Balance, nil
}

func (r *PostRepository) CreatePillar(ctx context.Context, pillar *taxonomy.Pillar) error {
	var existing taxonomy.Pillar

	pillar.Slug = helpers.GenerateSlug(pillar.Slug)

	err := r.db.WithContext(ctx).
		Where("slug = ?", pillar.Slug).
		First(&existing).Error

	if err == nil {
		return fmt.Errorf("pillar already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if pillar.ID == uuid.Nil {
		pillar.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(pillar).Error
}

func normalizeTaxonomySlug(raw string) string {
	return helpers.GenerateSlug(strings.TrimSpace(raw))
}

func normalizeTaxonomySearch(raw string) (string, string) {
	slug := normalizeTaxonomySlug(raw)
	return slug, strings.ToLower(helpers.SlugifyStrict(slug))
}

func activeTaxonomyPreload(db *gorm.DB) *gorm.DB {
	return db.Where("is_active = ?", true).
		Where("deleted_at IS NULL")
}

func orderedSynonymPreload(db *gorm.DB) *gorm.DB {
	return db.Order("is_primary DESC").
		Order("search_weight DESC").
		Order("slug ASC")
}

func rootClusterPreload(db *gorm.DB) *gorm.DB {
	return activeTaxonomyPreload(db).
		Where("parent_id IS NULL").
		Order("slug ASC")
}

func childClusterPreload(db *gorm.DB) *gorm.DB {
	return activeTaxonomyPreload(db).
		Order("slug ASC")
}

func taxonomyPillarTreeQuery(tx *gorm.DB) *gorm.DB {
	return tx.Preload("Clusters", rootClusterPreload).
		Preload("Clusters.Synonyms", orderedSynonymPreload).
		Preload("Clusters.Children", childClusterPreload).
		Preload("Clusters.Children.Synonyms", orderedSynonymPreload)
}

func exactClusterLookupQuery(tx *gorm.DB, pillarID uuid.UUID, parentID *uuid.UUID, slug string) *gorm.DB {
	slug = normalizeTaxonomySlug(slug)

	query := tx.Where("pillar_id = ?", pillarID).
		Where("slug = ?", slug).
		Where("deleted_at IS NULL")

	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id IS NULL")
	}

	return query
}

func taxonomyPillarMatchQuery(tx *gorm.DB, slug string) *gorm.DB {
	normalizedSlug, strictSlug := normalizeTaxonomySearch(slug)
	if normalizedSlug == "" {
		return activeTaxonomyPreload(tx)
	}

	likePattern := "%" + strictSlug + "%"

	return activeTaxonomyPreload(tx).
		Where(`
			pillars.slug = ?
			OR EXISTS (
				SELECT 1
				FROM clusters
				WHERE clusters.pillar_id = pillars.id
				  AND clusters.is_active = true
				  AND clusters.deleted_at IS NULL
				  AND (
					clusters.slug = ?
					OR clusters.search_vector ILIKE ?
					OR EXISTS (
						SELECT 1
						FROM synonyms
						WHERE synonyms.cluster_id = clusters.id
						  AND synonyms.slug = ?
					)
				  )
			)
		`, normalizedSlug, normalizedSlug, likePattern, normalizedSlug)
}

func applyTaxonomyCategoryFilter(query *gorm.DB, category *string) *gorm.DB {
	if category == nil {
		return query
	}

	normalizedSlug, strictSlug := normalizeTaxonomySearch(*category)
	if normalizedSlug == "" {
		return query
	}

	likePattern := "%" + strictSlug + "%"

	return query.Where(`
		EXISTS (
			SELECT 1
			FROM post_clusters
			JOIN clusters ON clusters.id = post_clusters.cluster_id
			JOIN pillars ON pillars.id = clusters.pillar_id
			LEFT JOIN synonyms ON synonyms.cluster_id = clusters.id
			WHERE post_clusters.post_id = posts.id
			  AND clusters.is_active = true
			  AND clusters.deleted_at IS NULL
			  AND pillars.is_active = true
			  AND pillars.deleted_at IS NULL
			  AND (
				pillars.slug = ?
				OR clusters.slug = ?
				OR clusters.search_vector ILIKE ?
				OR synonyms.slug = ?
			  )
		)
	`, normalizedSlug, normalizedSlug, likePattern, normalizedSlug)
}

func (r *PostRepository) CreateCluster(ctx context.Context, cluster *taxonomy.Cluster) error {
	var existing taxonomy.Cluster

	cluster.Slug = normalizeTaxonomySlug(cluster.Slug)

	err := exactClusterLookupQuery(r.db.WithContext(ctx), cluster.PillarID, cluster.ParentID, cluster.Slug).
		First(&existing).Error
	if err == nil {
		return fmt.Errorf("cluster already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if cluster.ID == uuid.Nil {
		cluster.ID = uuid.New()
	}

	return r.db.WithContext(ctx).Create(cluster).Error
}

func (r *PostRepository) CreateSynonym(ctx context.Context, synonym *taxonomy.Synonym) error {
	var existing taxonomy.Synonym
	synonym.Slug = normalizeTaxonomySlug(synonym.Slug)
	err := r.db.WithContext(ctx).
		Where("cluster_id = ?", synonym.ClusterID).
		Where("slug = ?", synonym.Slug).
		First(&existing).Error

	if err == nil {
		return fmt.Errorf("synonym already exists")
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if synonym.ID == uuid.Nil {
		synonym.ID = uuid.New()
	}

	if synonym.SearchWeight <= 0 {
		synonym.SearchWeight = 1
	}

	if synonym.IsPrimary {
		var primaryCount int64
		if err := r.db.WithContext(ctx).
			Model(&taxonomy.Synonym{}).
			Where("cluster_id = ? AND is_primary = ?", synonym.ClusterID, true).
			Count(&primaryCount).Error; err != nil {
			return err
		}

		if primaryCount > 0 {
			return fmt.Errorf("primary synonym already exists for this cluster")
		}
	}

	synonym.CreatedAt = time.Now()

	return r.db.WithContext(ctx).Create(synonym).Error
}

func (r *PostRepository) PillarExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var count int64
	slug = helpers.GenerateSlug(slug)

	err := r.db.WithContext(ctx).
		Model(&taxonomy.Pillar{}).
		Where("slug = ?", slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostRepository) GetPillarBySlug(ctx context.Context, slug string) (*taxonomy.Pillar, error) {
	var pillar taxonomy.Pillar
	slug = helpers.GenerateSlug(slug)

	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&pillar).Error

	if err != nil {
		return nil, err
	}

	return &pillar, nil
}

func (r *PostRepository) GetOrCreatePillar(ctx context.Context, slug string, name utils.LocalizedString) (*taxonomy.Pillar, error) {
	var pillar taxonomy.Pillar
	slug = helpers.GenerateSlug(slug)

	err := r.db.WithContext(ctx).
		Where("slug = ?", slug).
		First(&pillar).Error

	if err == nil {
		return &pillar, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	pillar = taxonomy.Pillar{
		ID:   uuid.New(),
		Slug: slug,
		Name: name,
	}

	if err := r.db.WithContext(ctx).Create(&pillar).Error; err != nil {
		return nil, err
	}

	return &pillar, nil
}

func (r *PostRepository) ClusterExists(ctx context.Context, pillarID uuid.UUID, parentID *uuid.UUID, slug string) (bool, error) {
	_, err := r.GetCluster(ctx, pillarID, parentID, slug)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *PostRepository) FindClusterBySlug(ctx context.Context, pillarID uuid.UUID, slug string) (*taxonomy.Cluster, error) {
	var clusters []taxonomy.Cluster

	err := r.db.WithContext(ctx).
		Where("pillar_id = ?", pillarID).
		Where("slug = ?", normalizeTaxonomySlug(slug)).
		Where("deleted_at IS NULL").
		Limit(2).
		Find(&clusters).Error
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	if len(clusters) > 1 {
		return nil, fmt.Errorf("cluster slug %q is ambiguous within pillar %s", slug, pillarID)
	}

	return &clusters[0], nil
}

func (r *PostRepository) GetCluster(ctx context.Context, pillarID uuid.UUID, parentID *uuid.UUID, slug string) (*taxonomy.Cluster, error) {
	var cluster taxonomy.Cluster
	if err := exactClusterLookupQuery(r.db.WithContext(ctx), pillarID, parentID, slug).
		First(&cluster).Error; err != nil {
		return nil, err
	}

	return &cluster, nil
}

func (r *PostRepository) GetOrCreateCluster(ctx context.Context, pillarID uuid.UUID, parentID *uuid.UUID, slug string, name utils.LocalizedString) (*taxonomy.Cluster, error) {

	var cluster taxonomy.Cluster
	slug = normalizeTaxonomySlug(slug)

	err := exactClusterLookupQuery(r.db.WithContext(ctx), pillarID, parentID, slug).
		First(&cluster).Error
	if err == nil {
		return &cluster, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	cluster = taxonomy.Cluster{
		ID:       uuid.New(),
		PillarID: pillarID,
		ParentID: parentID,
		Slug:     slug,
		Name:     name,
		IsActive: true,
	}

	if err := r.db.WithContext(ctx).Create(&cluster).Error; err != nil {
		return nil, err
	}

	return &cluster, nil
}

func (r *PostRepository) SynonymExists(ctx context.Context, clusterID uuid.UUID, slug string) (bool, error) {
	var count int64
	slug = normalizeTaxonomySlug(slug)

	err := r.db.WithContext(ctx).
		Model(&taxonomy.Synonym{}).
		Where("cluster_id = ? AND slug = ?", clusterID, slug).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *PostRepository) GetSynonym(ctx context.Context, clusterID uuid.UUID, slug string) (*taxonomy.Synonym, error) {
	var synonym taxonomy.Synonym
	slug = normalizeTaxonomySlug(slug)

	if err := r.db.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		Where("slug = ?", slug).
		First(&synonym).Error; err != nil {
		return nil, err
	}

	return &synonym, nil
}

func (r *PostRepository) GetOrCreateSynonym(ctx context.Context, clusterID uuid.UUID, slug string, word utils.LocalizedString, isPrimary bool, weight int) (*taxonomy.Synonym, error) {

	var synonym taxonomy.Synonym
	slug = normalizeTaxonomySlug(slug)

	err := r.db.WithContext(ctx).
		Where("cluster_id = ?", clusterID).
		Where("slug = ?", slug).
		First(&synonym).Error

	if err == nil {
		return &synonym, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	synonym = taxonomy.Synonym{
		ID:           uuid.New(),
		ClusterID:    clusterID,
		Slug:         slug,
		Word:         word,
		IsPrimary:    isPrimary,
		SearchWeight: weight,
	}

	if err := r.db.WithContext(ctx).Create(&synonym).Error; err != nil {
		return nil, err
	}

	return &synonym, nil
}

func (r *PostRepository) GetPillars(ctx context.Context) ([]taxonomy.Pillar, error) {
	var pillars []taxonomy.Pillar

	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("updated_at DESC").
		Find(&pillars).Error

	return pillars, err
}

func (r *PostRepository) GetClusters(ctx context.Context) ([]taxonomy.Cluster, error) {
	var clusters []taxonomy.Cluster

	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("updated_at DESC").
		Find(&clusters).Error

	return clusters, err
}

func (r *PostRepository) GetPillarsWithClusters(filters types.Filter) ([]taxonomy.Pillar, error) {
	var pillars []taxonomy.Pillar

	err := taxonomyPillarTreeQuery(
		activeTaxonomyPreload(
			r.db.WithContext(filters.Context).Model(&taxonomy.Pillar{}),
		),
	).
		Where("domain = ?", *filters.Domain).
		Order("pillars.slug ASC").
		Find(&pillars).Error

	if err != nil {
		return nil, err
	}

	return pillars, nil
}

func (r *PostRepository) GetPillarsWithClustersWithSlug(ctx context.Context, slug string) ([]taxonomy.Pillar, error) {
	var pillars []taxonomy.Pillar

	err := taxonomyPillarTreeQuery(
		taxonomyPillarMatchQuery(
			r.db.WithContext(ctx).Model(&taxonomy.Pillar{}),
			slug,
		),
	).
		Order("pillars.slug ASC").
		Find(&pillars).Error

	if err != nil {
		return nil, err
	}

	return pillars, nil
}
