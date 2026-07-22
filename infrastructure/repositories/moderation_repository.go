package repositories

import (
	"bytes"
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	domainmoderation "core/domain/moderation"
	"core/models"
	"core/models/post"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ModerationRepository struct {
	db *gorm.DB
}

var fallbackModerationTargetLock sync.Mutex

func moderationTargetLockKey(contentableType models.EngagementContentableType, contentableID uuid.UUID) string {
	return fmt.Sprintf("moderation-target:%s:%s", contentableType, contentableID)
}

func lockModerationTarget(tx *gorm.DB, contentableType models.EngagementContentableType, contentableID uuid.UUID) *gorm.DB {
	return tx.Exec(
		"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
		moderationTargetLockKey(contentableType, contentableID),
	)
}

func NewModerationRepository(db *gorm.DB) *ModerationRepository {
	return &ModerationRepository{db: db}
}

func (r *ModerationRepository) FetchReports(ctx context.Context, filter ports.ModerationReportFilter) (ports.ModerationReportPage, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = constants.DEFAULT_LIMIT
	}
	if limit > constants.MAXIMUM_LIMIT {
		limit = constants.MAXIMUM_LIMIT
	}

	query := r.reportQueueQuery(ctx, filter)

	var reports []models.Report
	if err := query.Order("reports.created_at DESC, reports.id DESC").Limit(limit + 1).Find(&reports).Error; err != nil {
		return ports.ModerationReportPage{}, err
	}

	hasMore := len(reports) > limit
	if hasMore {
		reports = reports[:limit]
	}

	posts, err := r.postsForReports(ctx, reports)
	if err != nil {
		return ports.ModerationReportPage{}, err
	}
	users, err := r.usersForReports(ctx, reports)
	if err != nil {
		return ports.ModerationReportPage{}, err
	}

	items := make([]ports.ModerationReportItem, 0, len(reports))
	for _, report := range reports {
		reportView, err := mapModerationReportView(report)
		if err != nil {
			return ports.ModerationReportPage{}, err
		}

		var postView ports.ModerationPostView
		if postModel := posts[report.ContentableID]; postModel != nil {
			postView, err = mapModerationView[ports.ModerationPostView](postModel)
			if err != nil {
				return ports.ModerationReportPage{}, err
			}
		}

		var userView ports.ModerationUserView
		if userModel := users[report.ContentableID]; userModel != nil {
			userView, err = mapModerationView[ports.ModerationUserView](userModel)
			if err != nil {
				return ports.ModerationReportPage{}, err
			}
		}

		items = append(items, ports.ModerationReportItem{
			Report: reportView,
			Post:   postView,
			User:   userView,
		})
	}

	var cursor *string
	if hasMore && len(reports) > 0 {
		cursor, err = types.NewTimeUUIDCursor(reports[len(reports)-1].CreatedAt, reports[len(reports)-1].ID)
		if err != nil {
			return ports.ModerationReportPage{}, err
		}
	}

	return ports.ModerationReportPage{
		Items:  items,
		Cursor: cursor,
		Count:  len(items),
		Limit:  limit,
	}, nil
}

func (r *ModerationRepository) reportQueueQuery(ctx context.Context, filter ports.ModerationReportFilter) *gorm.DB {
	query := r.db.WithContext(ctx).
		Model(&models.Report{}).
		Preload("Reporter").
		Preload("Reporter.Avatar").
		Preload("Reporter.Avatar.File").
		Preload("ReportKind").
		Preload("ReviewedBy").
		Preload("ReviewedBy.Avatar").
		Preload("ReviewedBy.Avatar.File")

	if filter.ContentableType != "" {
		query = query.Where("reports.contentable_type = ?", filter.ContentableType)
	}
	if filter.Status != "" {
		query = query.Where("reports.status = ?", filter.Status)
	}
	if filter.Cursor != nil {
		if filter.CursorID != nil {
			query = query.Where(
				"(reports.created_at < ? OR (reports.created_at = ? AND reports.id < ?))",
				*filter.Cursor,
				*filter.Cursor,
				*filter.CursorID,
			)
		} else {
			query = query.Where("reports.created_at < ?", *filter.Cursor)
		}
	}
	if filter.PostPublicID != 0 {
		query = query.Joins(
			"JOIN posts moderation_posts ON moderation_posts.id = reports.contentable_id AND reports.contentable_type = ?",
			models.EngagementContentableTypePost,
		).Where("moderation_posts.public_id = ?", filter.PostPublicID)
	}
	if filter.UserPublicID != 0 {
		query = query.Joins(
			"JOIN users moderation_targets ON moderation_targets.id = reports.contentable_id AND reports.contentable_type = ?",
			models.EngagementContentableTypeUser,
		).Where("moderation_targets.public_id = ?", filter.UserPublicID)
	}
	if filter.ReporterPublicID != 0 {
		query = query.Joins(
			"JOIN users moderation_reporters ON moderation_reporters.id = reports.reporter_id",
		).Where("moderation_reporters.public_id = ?", filter.ReporterPublicID)
	}
	return query
}

func (r *ModerationRepository) ResolveReport(ctx context.Context, input ports.ModerationResolveInput) (*ports.ModerationReportView, error) {
	now := time.Now().UTC()
	reviewerID := input.ReviewedByID
	requestedStatus := models.ReportStatus(input.Status)
	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackModerationTargetLock.Lock()
		defer fallbackModerationTargetLock.Unlock()
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target struct {
			ContentableID   uuid.UUID
			ContentableType models.EngagementContentableType
		}
		if err := tx.Model(&models.Report{}).
			Select("contentable_id", "contentable_type").
			Where("id = ?", input.ReportID).
			Take(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrReportNotFound
			}
			return err
		}
		if input.PublishPost != nil && target.ContentableType != models.EngagementContentableTypePost {
			return ports.ErrInvalidModerationAction
		}
		if isPostgres {
			if result := lockModerationTarget(tx, target.ContentableType, target.ContentableID); result.Error != nil {
				return result.Error
			}
		}

		var report models.Report
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&report, "id = ?", input.ReportID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrReportNotFound
			}
			return err
		}
		if report.ContentableID != target.ContentableID || report.ContentableType != target.ContentableType {
			return errors.New("report target changed during moderation")
		}
		nextStatus, err := report.Status.TransitionTo(requestedStatus)
		if err != nil {
			return ports.ErrInvalidReportTransition
		}

		if input.PublishPost != nil {
			updates := postPublishUpdates(*input.PublishPost, now)
			result := tx.Model(&post.Post{}).Where("id = ?", report.ContentableID).Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ports.ErrReportTargetNotFound
			}
		}

		updates := map[string]any{
			"status":         nextStatus,
			"reviewed_by_id": &reviewerID,
			"reviewed_at":    &now,
		}
		if input.Resolution != "" || report.Resolution == "" {
			updates["resolution"] = input.Resolution
		}
		return tx.Model(&models.Report{}).Where("id = ?", input.ReportID).Updates(updates).Error
	}); err != nil {
		return nil, err
	}

	report, err := r.getReport(ctx, input.ReportID)
	if err != nil {
		return nil, err
	}
	view, err := mapModerationReportView(*report)
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func (r *ModerationRepository) SetPostPublished(ctx context.Context, postPublicID int64, published bool, moderatorID uuid.UUID, resolution string) (ports.ModerationPostView, error) {
	now := time.Now().UTC()
	reviewerID := moderatorID
	if resolution == "" {
		if published {
			resolution = "Post restored by moderator"
		} else {
			resolution = "Post hidden by moderator"
		}
	}
	isPostgres := r.db.Name() == "postgres"
	if !isPostgres {
		fallbackModerationTargetLock.Lock()
		defer fallbackModerationTargetLock.Unlock()
	}

	var postModel post.Post
	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target struct {
			ID uuid.UUID
		}
		if err := tx.Model(&post.Post{}).Select("id").Where("public_id = ?", postPublicID).Take(&target).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrReportTargetNotFound
			}
			return err
		}
		if isPostgres {
			if result := lockModerationTarget(tx, models.EngagementContentableTypePost, target.ID); result.Error != nil {
				return result.Error
			}
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&postModel, "id = ?", target.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ports.ErrReportTargetNotFound
			}
			return err
		}

		if err := tx.Model(&post.Post{}).
			Where("id = ?", postModel.ID).
			Updates(postPublishUpdates(published, now)).Error; err != nil {
			return err
		}

		reportStatus := models.ReportStatusActioned
		if published {
			reportStatus = models.ReportStatusReviewed
		}

		return tx.Model(&models.Report{}).
			Where("contentable_id = ? AND contentable_type = ? AND status = ?", postModel.ID, models.EngagementContentableTypePost, models.ReportStatusPending).
			Updates(map[string]any{
				"status":         reportStatus,
				"reviewed_by_id": &reviewerID,
				"reviewed_at":    &now,
				"resolution":     resolution,
			}).Error
	}); err != nil {
		return nil, err
	}

	updated, err := r.getPostByID(ctx, postModel.ID)
	if err != nil {
		return nil, err
	}
	return mapModerationView[ports.ModerationPostView](updated)
}

func (r *ModerationRepository) getReport(ctx context.Context, id uuid.UUID) (*models.Report, error) {
	var report models.Report
	if err := r.db.WithContext(ctx).
		Preload("Reporter").
		Preload("Reporter.Avatar").
		Preload("Reporter.Avatar.File").
		Preload("ReportKind").
		Preload("ReviewedBy").
		Preload("ReviewedBy.Avatar").
		Preload("ReviewedBy.Avatar.File").
		First(&report, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *ModerationRepository) postsForReports(ctx context.Context, reports []models.Report) (map[uuid.UUID]*post.Post, error) {
	ids := make([]uuid.UUID, 0, len(reports))
	seen := map[uuid.UUID]struct{}{}
	for _, report := range reports {
		if report.ContentableType != models.EngagementContentableTypePost {
			continue
		}
		if _, ok := seen[report.ContentableID]; ok {
			continue
		}
		seen[report.ContentableID] = struct{}{}
		ids = append(ids, report.ContentableID)
	}

	result := make(map[uuid.UUID]*post.Post, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var posts []post.Post
	if err := r.db.WithContext(ctx).
		Preload("Author").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Attachments").
		Preload("Attachments.File").
		Where("id IN ?", ids).
		Find(&posts).Error; err != nil {
		return nil, err
	}

	for i := range posts {
		result[posts[i].ID] = &posts[i]
	}
	return result, nil
}

func (r *ModerationRepository) usersForReports(ctx context.Context, reports []models.Report) (map[uuid.UUID]*models.User, error) {
	ids := make([]uuid.UUID, 0, len(reports))
	seen := make(map[uuid.UUID]struct{}, len(reports))
	for _, report := range reports {
		if report.ContentableType != models.EngagementContentableTypeUser {
			continue
		}
		if _, ok := seen[report.ContentableID]; ok {
			continue
		}
		seen[report.ContentableID] = struct{}{}
		ids = append(ids, report.ContentableID)
	}

	result := make(map[uuid.UUID]*models.User, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	var users []models.User
	if err := r.db.WithContext(ctx).
		Preload("Avatar").
		Preload("Avatar.File").
		Preload("Cover").
		Preload("Cover.File").
		Where("id IN ?", ids).
		Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		result[users[i].ID] = &users[i]
	}
	return result, nil
}

func (r *ModerationRepository) getPostByID(ctx context.Context, id uuid.UUID) (*post.Post, error) {
	var postModel post.Post
	if err := r.db.WithContext(ctx).
		Preload("Author").
		Preload("Author.Avatar").
		Preload("Author.Avatar.File").
		Preload("Attachments").
		Preload("Attachments.File").
		First(&postModel, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &postModel, nil
}

func postPublishUpdates(published bool, now time.Time) map[string]any {
	updates := map[string]any{
		"published": published,
	}
	if published {
		updates["published_at"] = &now
	} else {
		updates["published_at"] = nil
	}
	return updates
}

func mapModerationView[T ~map[string]any](value any) (T, error) {
	var view T
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	if err := decoder.Decode(&view); err != nil {
		return nil, err
	}
	return view, nil
}

func mapModerationReportView(report models.Report) (ports.ModerationReportView, error) {
	var reporter ports.ModerationUserView
	var err error
	if report.Reporter != nil {
		reporter, err = mapModerationView[ports.ModerationUserView](report.Reporter)
		if err != nil {
			return ports.ModerationReportView{}, err
		}
	}

	var reviewedBy ports.ModerationUserView
	if report.ReviewedBy != nil {
		reviewedBy, err = mapModerationView[ports.ModerationUserView](report.ReviewedBy)
		if err != nil {
			return ports.ModerationReportView{}, err
		}
	}

	var reportKind *ports.ModerationReportKindView
	if report.ReportKind != nil {
		reportKind = &ports.ModerationReportKindView{
			Key:          report.ReportKind.Key,
			DisplayOrder: report.ReportKind.DisplayOrder,
			Name:         copyModerationLocalizedText(report.ReportKind.Name),
			Description:  copyModerationLocalizedText(report.ReportKind.Description),
			CreatedAt:    report.ReportKind.CreatedAt,
			UpdatedAt:    report.ReportKind.UpdatedAt,
		}
	}

	return ports.ModerationReportView{
		ID:              report.ID,
		ContentableID:   report.ContentableID,
		ContentableType: domainmoderation.TargetType(report.ContentableType),
		ReporterID:      report.ReporterID,
		Reporter:        reporter,
		ReportKindKey:   report.ReportKindKey,
		ReportKind:      reportKind,
		Reason:          report.Reason,
		Status:          domainmoderation.Status(report.Status),
		ReviewedByID:    report.ReviewedByID,
		ReviewedBy:      reviewedBy,
		ReviewedAt:      report.ReviewedAt,
		Resolution:      report.Resolution,
		CreatedAt:       report.CreatedAt,
		UpdatedAt:       report.UpdatedAt,
	}, nil
}

func copyModerationLocalizedText(source map[string]string) ports.ModerationLocalizedText {
	if source == nil {
		return nil
	}
	result := make(ports.ModerationLocalizedText, len(source))
	for language, value := range source {
		result[language] = value
	}
	return result
}
