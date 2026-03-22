package repositories

import (
	"context"
	"core/models/post"
	"core/models/sitemap"
	"core/models/taxonomy"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SitemapRepository struct {
	db *gorm.DB
}

func (r *SitemapRepository) DB() *gorm.DB {
	return r.db
}

func NewSitemapRepository(db *gorm.DB) *SitemapRepository {
	return &SitemapRepository{db: db}
}

func (r *SitemapRepository) GetSitemapPosts(
	ctx context.Context,
	limit int,
	offset int,
) ([]post.Post, error) {

	var posts []post.Post

	err := r.db.WithContext(ctx).
		Where("published = ?", true).
		Where("deleted_at IS NULL").
		Where("post_kind NOT IN ?", []string{"chat", "message"}).
		Order("updated_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error

	return posts, err
}

func BuildSitemapIndex(baseURL string, totalParts int) ([]byte, error) {

	index := sitemap.SitemapIndex{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for i := 1; i <= totalParts; i++ {
		index.Sitemaps = append(index.Sitemaps, sitemap.SitemapItem{
			Loc:     baseURL + fmt.Sprintf("/sitemap-posts-%d.xml", i),
			LastMod: time.Now().Format("2006-01-02T15:04:05Z"),
		})
	}

	output, err := xml.MarshalIndent(index, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

func (r *SitemapRepository) CountPublishedPosts(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&post.Post{}).
		Where("is_published = ?", true).
		Where("deleted_at IS NULL").
		Count(&count).Error

	return count, err
}

func (r *SitemapRepository) GetLatestNewsPosts(ctx context.Context) ([]post.Post, error) {

	var posts []post.Post

	cutoff := time.Now().Add(-48 * time.Hour)

	err := r.db.WithContext(ctx).
		Where("published = ?", true).
		Where("deleted_at IS NULL").
		Where("published_at >= ?", cutoff).
		Order("published_at DESC").
		Limit(1000).
		Find(&posts).Error

	return posts, err
}

func (r *SitemapRepository) BuildNewsSitemap(
	ctx context.Context,
	baseURL string,
	siteName string,
	language string,
) ([]byte, error) {

	posts, err := r.GetLatestNewsPosts(ctx)
	if err != nil {
		return nil, err
	}

	urlSet := sitemap.NewsURLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		News:  "http://www.google.com/schemas/sitemap-news/0.9",
	}

	for _, p := range posts {

		if p.Slug == nil || p.PublishedAt == nil {
			continue
		}

		urlSet.URLs = append(urlSet.URLs, sitemap.NewsURL{
			Loc: fmt.Sprintf("%s/news/%s", baseURL, *p.Slug),
			News: sitemap.NewsMeta{
				Publication: sitemap.Publication{
					Name:     siteName,
					Language: language,
				},
				PublicationDate: p.PublishedAt.UTC().Format(time.RFC3339),
				Title:           p.Title.DefaultValue(),
			},
		})
	}

	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

func (r *SitemapRepository) GenerateSitemapIndex(baseURL string) (string, error) {

	type Sitemap struct {
		Loc     string `xml:"loc"`
		LastMod string `xml:"lastmod,omitempty"`
	}

	type SitemapIndex struct {
		XMLName xml.Name  `xml:"sitemapindex"`
		Xmlns   string    `xml:"xmlns,attr"`
		Maps    []Sitemap `xml:"sitemap"`
	}

	index := SitemapIndex{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
		Maps: []Sitemap{
			{Loc: baseURL + "/sitemap-news.xml"},
			{Loc: baseURL + "/sitemap-posts.xml"},
			{Loc: baseURL + "/sitemap-pillars.xml"},
			{Loc: baseURL + "/sitemap-clusters.xml"},
			{Loc: baseURL + "/sitemap-images.xml"},
			{Loc: baseURL + "/sitemap-videos.xml"},
		},
	}

	output, err := xml.MarshalIndent(index, "", "  ")
	if err != nil {
		return "", err
	}

	return xml.Header + string(output), nil
}

func (r *SitemapRepository) BuildPostSitemap(
	ctx context.Context,
	baseURL string,
	limit int,
	offset int,
) ([]byte, error) {

	posts, err := r.GetSitemapPosts(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	urlSet := sitemap.URLSet{
		Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
	}

	for _, p := range posts {

		if p.Slug == nil || p.PublishedAt == nil {
			continue
		}

		age := time.Since(*p.PublishedAt)
		priority := "0.6"

		switch {
		case age <= 24*time.Hour:
			priority = "1.0"
		case age <= 7*24*time.Hour:
			priority = "0.9"
		case age <= 30*24*time.Hour:
			priority = "0.8"
		}

		urlSet.URLs = append(urlSet.URLs, sitemap.URL{
			Loc:        fmt.Sprintf("%s/news/%s", baseURL, *p.Slug),
			LastMod:    p.UpdatedAt.UTC().Format(time.RFC3339),
			Priority:   priority,
			ChangeFreq: map[bool]string{true: "hourly", false: "daily"}[age <= 24*time.Hour],
		})
	}

	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

func (r *SitemapRepository) GeneratePillarSitemap(ctx context.Context, baseURL string) ([]byte, error) {

	var pillars []taxonomy.Pillar
	r.db.Find(&pillars)

	builder := sitemap.NewSitemapBuilder(baseURL)

	var items []sitemap.SitemapItem

	for _, p := range pillars {
		items = append(items, sitemap.SitemapItem{
			Loc:     builder.BuildLoc("pillar/" + p.Slug),
			LastMod: p.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return builder.BuildURLSitemap(items)
}

func (r *SitemapRepository) GenerateClusterSitemap(ctx context.Context, baseURL string) ([]byte, error) {

	var clusters []taxonomy.Cluster
	r.db.Find(&clusters)

	builder := sitemap.NewSitemapBuilder(baseURL)

	var items []sitemap.SitemapItem

	for _, c := range clusters {
		items = append(items, sitemap.SitemapItem{
			Loc:     builder.BuildLoc("cluster/" + c.Slug),
			LastMod: c.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return builder.BuildURLSitemap(items)
}

func (r *SitemapRepository) GeneratePostSitemap(
	ctx context.Context,
	baseURL string,
) ([]byte, error) {
	return r.BuildPostSitemap(ctx, baseURL, 50000, 0)
}

func (r *SitemapRepository) GenerateNewsSitemap(
	ctx context.Context,
	baseURL string,
) ([]byte, error) {
	return r.BuildNewsSitemap(ctx, baseURL, "Your Site", "tr")
}

func (r *SitemapRepository) GenerateImageSitemap(ctx context.Context, baseURL string) ([]byte, error) {

	var posts []post.Post

	err := r.db.WithContext(ctx).
		Joins("JOIN medias ON medias.owner_id = posts.id AND medias.owner_type = 'post'").
		Joins("JOIN file_metadata ON file_metadata.id = medias.file_id").
		Where("posts.published = ?", true).
		Where("posts.deleted_at IS NULL").
		Where("posts.post_kind NOT IN ?", []string{"chat", "message"}).
		Where("file_metadata.mime_type LIKE ?", "image/%").
		Preload("Author").
		Preload("Attachments").
		Preload("Attachments.File").
		Group("posts.id").
		Find(&posts).Error
	if err != nil {
		return nil, err
	}

	urlSet := sitemap.ImageURLSet{
		Xmlns:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		ImageNS: "http://www.google.com/schemas/sitemap-image/1.1",
	}

	for _, p := range posts {

		for _, attachment := range p.Attachments {
			if attachment == nil || attachment.File.ID == uuid.Nil {
				continue
			}

			if strings.HasPrefix(attachment.File.MimeType, "image/") && attachment.File.StoragePath != "" {
				urlSet.URLs = append(urlSet.URLs, sitemap.ImageURLItem{
					Loc: fmt.Sprintf("%s/%s/%s/%d", baseURL, p.Author.UserName, p.PostKind, p.PublicID),
					Images: []sitemap.ImageEntry{
						{
							Loc:   baseURL + "/" + strings.TrimPrefix(attachment.File.StoragePath, "./"),
							Title: p.SafeTitle(),
						},
					},
				})
				break
			}
		}
	}

	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}

func (r *SitemapRepository) GenerateVideoSitemap(ctx context.Context, baseURL string) ([]byte, error) {

	var posts []post.Post

	err := r.db.WithContext(ctx).
		Joins("JOIN medias ON medias.owner_id = posts.id AND medias.owner_type = 'post'").
		Joins("JOIN file_metadata ON file_metadata.id = medias.file_id").
		Where("posts.published = ?", true).
		Where("posts.deleted_at IS NULL").
		Where("posts.post_kind NOT IN ?", []string{"chat", "message"}).
		Where("file_metadata.mime_type LIKE ?", "video/%").
		Preload("Attachments.File").
		Preload("Author").
		Group("posts.id").
		Find(&posts).Error

	if err != nil {
		return nil, err
	}

	urlSet := sitemap.VideoURLSet{
		Xmlns:   "http://www.sitemaps.org/schemas/sitemap/0.9",
		VideoNS: "http://www.google.com/schemas/sitemap-video/1.1",
	}

	for _, p := range posts {

		if p.Slug == nil || p.Title == nil || p.Summary == nil {
			continue
		}

		for _, attachment := range p.Attachments {
			if attachment == nil || attachment.File.ID == uuid.Nil {
				continue
			}

			if strings.HasPrefix(attachment.File.MimeType, "video/") && attachment.File.URL != "" {

				videoMeta := sitemap.VideoMeta{
					Title:       p.Title.DefaultValue(),
					Description: p.Summary.DefaultValue(),
					ContentLoc:  baseURL + "/" + strings.TrimPrefix(attachment.File.URL, "./"),
				}

				if attachment.File.Variants != nil && attachment.File.Variants.Video != nil && attachment.File.Variants.Video.Poster != nil && attachment.File.Variants.Video.Poster.URL != "" {
					videoMeta.ThumbnailLoc = baseURL + "/" + strings.TrimPrefix(attachment.File.Variants.Video.Poster.URL, "./")
				}

				urlSet.URLs = append(urlSet.URLs, sitemap.VideoURLItem{
					Loc:    fmt.Sprintf("%s/%s/%s/%d", baseURL, p.Author.UserName, p.PostKind, p.PublicID),
					Videos: []sitemap.VideoMeta{videoMeta},
				})

				break
			}
		}
	}

	output, err := xml.MarshalIndent(urlSet, "", "  ")
	if err != nil {
		return nil, err
	}

	return append([]byte(xml.Header), output...), nil
}
