package repositories

import (
	"bifrost/helpers"
	"bifrost/models/media"
	"bifrost/models/post"
	post_payloads "bifrost/models/post/payloads"
	userModel "bifrost/models/user"
	"bifrost/types"
	"sort"

	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostRepository struct {
	db            *gorm.DB
	snowFlakeNode *helpers.Node
}

func (r *PostRepository) DB() *gorm.DB {
	return r.db
}

func (r *PostRepository) Node() *helpers.Node {
	return r.snowFlakeNode
}

func NewPostRepository(db *gorm.DB, snowFlakeNode *helpers.Node) *PostRepository {
	return &PostRepository{db: db, snowFlakeNode: snowFlakeNode}
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

	var nextCursor *int64
	if len(posts) > 0 {
		nextCursor = &posts[len(posts)-1].PublicID
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
