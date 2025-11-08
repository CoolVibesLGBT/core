package repositories

import (
	"coolvibes/extensions"
	"coolvibes/helpers"
	"coolvibes/models"
	"coolvibes/models/media"
	"coolvibes/models/post"
	"coolvibes/models/post/payloads"
	global_shared "coolvibes/models/shared"
	"strconv"

	userModel "coolvibes/models"
	post_payloads "coolvibes/models/post/payloads"
	"coolvibes/models/post/utils"
	"coolvibes/types"
	"mime/multipart"
	"sort"

	"fmt"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
	mediaRepo     *MediaRepository
	userRepo      *UserRepository
}

func (r *PostRepository) DB() *gorm.DB {
	return r.db
}

func (r *PostRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewPostRepository(db *gorm.DB, snowFlakeNode *helpers.Node, mediaRepo *MediaRepository, userRepo *UserRepository) *PostRepository {
	return &PostRepository{db: db, snowFlakeNode: snowFlakeNode, mediaRepo: mediaRepo, userRepo: userRepo}
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
	fmt.Println("CREATE POLL")
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
	// 1️⃣ Recursive CTE ile root ve tüm children ID'lerini al
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

	// 2️⃣ Tüm post'ları ilişkileriyle al
	var posts []post.Post
	if err := r.db.
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
		Where("id IN ?", ids).
		Order("created_at ASC").
		Find(&posts).Error; err != nil {
		return nil, err
	}

	if len(posts) == 0 {
		return nil, fmt.Errorf("no posts found for %s", id)
	}

	// 3️⃣ ID -> *Post map oluştur
	postMap := make(map[uuid.UUID]*post.Post, len(posts))
	for i := range posts {
		posts[i].Children = nil // temizle
		postMap[posts[i].ID] = &posts[i]
	}

	// 4️⃣ Recursive ağaç oluşturucu fonksiyon
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

	// 5️⃣ Root post'u bul
	root, ok := postMap[id]
	if !ok {
		return nil, fmt.Errorf("post with id %s not found in postMap", id)
	}

	// 6️⃣ Root’un tüm children’larını derinlemesine inşa et
	buildTree(root)

	// 7️⃣ Recursive sort (sabit sıra için)
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

func (r *PostRepository) GetTimeline(limit int, cursor *int64) (types.TimelineResult, error) {
	var posts []post.Post

	fmt.Println("POST_REPO:GetTimeline:TIMELINE:")
	query := r.db.Model(&post.Post{}).
		//Where("published = ?", true).
		Where("contentable_type = ?", post.PostTypePost). // 👈 sadece post olanlar
		Order("public_id DESC").
		Limit(limit).
		Preload("Location").
		Preload("Poll").
		Preload("Poll.Choices").
		Preload("Event").
		Preload("Event.Location").
		Preload("Event.Attendees").
		Preload("Author.GenderIdentities").
		Preload("Author.SexualOrientations").
		Preload("Author.SexualRole").
		Preload("Author.Avatar").
		Preload("Author.Cover").
		Preload("Author.Fantasies").
		Preload("Hashtags").
		Preload("Attachments").
		Preload("Attachments.File")

	if cursor != nil {
		query = query.Where("public_id < ?", *cursor)
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
		Posts:      posts,
		NextCursor: nextCursor,
	}, nil
}

func (r *PostRepository) GetTimelineVibes(limit int, cursor *int64) (types.TimelineResult, error) {
	var posts []post.Post

	query := r.db.Model(&post.Post{}).
		Joins("INNER JOIN medias ON medias.owner_id = posts.id AND medias.owner_type = ?", "post").
		Preload("Author").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Author.Cover").
		Preload("Author.Cover.File").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("published = ?", true).
		Order("posts.public_id DESC").
		Limit(limit).
		Group("posts.id")

	if cursor != nil {
		query = query.Where("posts.public_id < ?", *cursor)
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
		Posts:      posts,
		NextCursor: nextCursor,
	}, nil
}

func (r *PostRepository) GetUserPosts(userId uuid.UUID, cursor *int64, limit int) ([]post.Post, error) {
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
		Where("author_id = ? AND parent_id IS NULL", userId).
		Order("public_id DESC").
		Limit(limit)

	// Eğer cursor verilmişse, sadece daha önceki postları al
	if cursor != nil {
		query = query.Where("public_id < ?", *cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepository) GetUserPostReplies(userID uuid.UUID, cursor *int64, limit int) ([]post.Post, error) {
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
		Where("author_id = ? AND parent_id IS NOT NULL", userID).
		Order("public_id DESC").
		Limit(limit)

	// Cursor varsa sadece daha eski postlar
	if cursor != nil {
		query = query.Where("public_id < ?", *cursor)
	}

	if err := query.Find(&posts).Error; err != nil {
		return nil, err
	}

	return posts, nil
}

func (r *PostRepository) GetUserMedias(userID uuid.UUID, cursor *int64, limit int) ([]types.MediaWithUser, *int64, error) {
	var medias []media.Media

	query := r.db.Unscoped().
		Preload("File").
		Where("user_id = ?", userID).
		Order("public_id DESC").
		Limit(limit)

	if cursor != nil {
		query = query.Where("public_id < ?", *cursor)
	}

	if err := query.Find(&medias).Error; err != nil {
		return nil, nil, err
	}

	// userID'leri topla
	userIDs := make([]uuid.UUID, 0, len(medias))
	for _, m := range medias {
		userIDs = append(userIDs, m.UserID)
	}

	var users []userModel.User
	if len(userIDs) > 0 {
		if err := r.db.Where("id IN ?", userIDs).Find(&users).Error; err != nil {
			return nil, nil, err
		}
	}

	userMap := make(map[uuid.UUID]userModel.User, len(users))
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

func (r *PostRepository) GetRecentHashtags(limit int) ([]types.HashtagStats, error) {
	var results []types.HashtagStats
	cutoff := time.Now().Add(-48 * time.Hour)
	err := r.db.Model(&types.HashtagStats{}).
		Select("tag, COUNT(*) as count").
		Where("created_at >= ?", cutoff).
		Group("tag").
		Order("count DESC").
		Scan(&results).Error

	return results, err
}

func (r *PostRepository) CreateContentablePost(request map[string][]string, files []*multipart.FileHeader, author *userModel.User, contentableType string, contentableID *uuid.UUID) (*post.Post, error) {
	type PollForm struct {
		ID       string   `form:"id"`
		Question string   `form:"question"`
		Duration string   `form:"duration"`
		Options  []string `form:"options"`
	}

	type PostForm struct {
		ParentId string     `form:"parentPostId"`
		Title    string     `form:"title"`
		Summary  string     `form:"summary"`
		Content  string     `form:"content"`
		Audience string     `form:"audience"`
		Hashtags []string   `form:"hashtags[]"`
		Mentions []string   `form:"mentions[]"`
		Polls    []PollForm `form:"polls"`

		EventTitle       string `form:"event[title]"`
		EventDescription string `form:"event[description]"`
		EventDate        string `form:"event[date]"`
		EventTime        string `form:"event[time]"`

		LocationAddress string  `form:"location[address]"`
		LocationLat     float64 `form:"location[lat]"`
		LocationLng     float64 `form:"location[lng]"`
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

	node, err := helpers.NewNode(1)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create snowflake node: %w", err)
	}

	var parentUUID *uuid.UUID
	if len(postForm.ParentId) > 0 {
		parsed, err := uuid.Parse(postForm.ParentId)
		if err == nil {
			parentUUID = &parsed
		}
	}

	defaultLanguage := author.DefaultLanguage

	var postKindType post.PostType
	switch contentableType {
	case "chat":
		postKindType = post.PostTypeChat
	default:
		postKindType = post.PostTypeStatus
	}

	newPost := &post.Post{
		ID:              uuid.New(),
		ParentID:        parentUUID,
		PublicID:        node.Generate().Int64(),
		AuthorID:        author.ID,
		Published:       true,
		PostKind:        postKindType,
		ContentCategory: post.ContentNormal,
		Title:           utils.MakeLocalizedString(defaultLanguage, postForm.Title),
		Content:         utils.MakeLocalizedString(defaultLanguage, postForm.Content),
		Summary:         utils.MakeLocalizedString(defaultLanguage, postForm.Summary),

		// Burada contentable bilgisi de tutuluyor
		ContentableType: &contentableType,
		ContentableID:   contentableID,
	}

	if err := tx.Create(newPost).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// Media ekleme
	for _, f := range files {

		var ownerType media.OwnerType
		var role media.MediaRole

		switch contentableType {
		case "chat":
			ownerType = media.OwnerChat
			role = media.RoleChatMedia // burada istersen MIME type’a göre video da yapabiliriz
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
		poll := &payloads.Poll{
			ID:              uuid.New(),
			ContentableID:   newPost.ID,
			ContentableType: payloads.ContentablePollPost,
			Question:        *utils.MakeLocalizedString(defaultLanguage, pollInfo.Question),
			Duration:        pollInfo.Duration,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		for _, choiceLabel := range pollInfo.Options {
			poll.Choices = append(poll.Choices, payloads.PollChoice{
				ID:        uuid.New(),
				PollID:    poll.ID,
				Label:     *utils.MakeLocalizedString(defaultLanguage, choiceLabel),
				VoteCount: 0,
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
		locationPost := &global_shared.Location{
			ID:              uuid.New(),
			ContentableType: global_shared.LocationOwnerPost,
			ContentableID:   newPost.ID,
			Address:         &postForm.LocationAddress,
			Latitude:        &postForm.LocationLat,
			Longitude:       &postForm.LocationLng,
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

		evt := &payloads.Event{
			ID:          uuid.New(),
			PostID:      newPost.ID,
			Title:       *utils.MakeLocalizedString(defaultLanguage, postForm.EventTitle),
			Description: *utils.MakeLocalizedString(defaultLanguage, postForm.EventDescription),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			StartTime:   &startTime,
		}
		if err := tx.Create(evt).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		locationEvent := &global_shared.Location{
			ID:              uuid.New(),
			ContentableType: global_shared.LocationOwnerEvent,
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

	// Hashtags
	for _, hashtagStr := range postForm.Hashtags {
		hashtagItem := models.Hashtag{
			ID:  uuid.New(),
			Tag: hashtagStr,
		}
		newPost.Hashtags = append(newPost.Hashtags, &hashtagItem)
	}

	if err := tx.Save(newPost).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	return newPost, nil
}
