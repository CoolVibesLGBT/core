package repositories

import (
	"context"
	"core/application/ports"
	domainmoderation "core/domain/moderation"
	"core/models"
	"core/models/post"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func createReport(
	ctx context.Context,
	db *gorm.DB,
	contentableID uuid.UUID,
	contentableType models.EngagementContentableType,
	reporterID uuid.UUID,
	kind string,
	description string,
) error {
	if _, err := domainmoderation.ParseTargetType(string(contentableType)); err != nil {
		return err
	}
	reportKind, err := domainmoderation.NewKind(kind)
	if err != nil {
		return err
	}
	reportDescription, err := domainmoderation.NewDescription(description)
	if err != nil {
		return err
	}
	kind = reportKind.String()

	var kindCount int64
	if err := db.WithContext(ctx).
		Model(&models.ReportKind{}).
		Where("key = ?", kind).
		Count(&kindCount).Error; err != nil {
		return err
	}
	if kindCount == 0 {
		return ports.ErrInvalidReportKind
	}

	report := models.Report{
		ID:              uuid.New(),
		ContentableID:   contentableID,
		ContentableType: contentableType,
		ReporterID:      reporterID,
		ReportKindKey:   kind,
		Reason:          reportDescription.String(),
		Status:          models.ReportStatusPending,
	}
	isPostgres := db.Name() == "postgres"
	if !isPostgres {
		fallbackModerationTargetLock.Lock()
		defer fallbackModerationTargetLock.Unlock()
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if isPostgres {
			if result := lockModerationTarget(tx, contentableType, contentableID); result.Error != nil {
				return result.Error
			}
		}
		// Target resolution performed by the public-ID repository method is only
		// a fast pre-check. Revalidate after taking the same target lock used by
		// moderation writes so a concurrent hide/delete cannot leave a new
		// pending report attached to an ineligible target.
		if err := validateReportTarget(tx, contentableType, contentableID); err != nil {
			return err
		}
		lockKey := fmt.Sprintf("report:%s:%s:%s", reporterID, contentableType, contentableID)
		if isPostgres {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", lockKey).Error; err != nil {
				return err
			}
		}

		var existing models.Report
		err := tx.
			Where("reporter_id = ? AND contentable_id = ? AND contentable_type = ? AND status = ?", reporterID, contentableID, contentableType, models.ReportStatusPending).
			First(&existing).Error
		if err == nil {
			return tx.Model(&existing).Updates(map[string]any{
				"report_kind_key": report.ReportKindKey,
				"reason":          report.Reason,
			}).Error
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return tx.Create(&report).Error
	})
}

func validateReportTarget(tx *gorm.DB, contentableType models.EngagementContentableType, contentableID uuid.UUID) error {
	var count int64
	query := tx
	switch contentableType {
	case models.EngagementContentableTypePost:
		query = query.Model(&post.Post{}).
			Where("id = ? AND published = TRUE", contentableID).
			Where("post_kind NOT IN ?", []post.PostKind{post.PostKindChat, post.PostKindMessage}).
			Where("COALESCE(NULLIF(audience, ''), 'public') = 'public'")
	case models.EngagementContentableTypeUser:
		query = query.Model(&models.User{}).Where("id = ?", contentableID)
	default:
		return domainmoderation.ErrInvalidTargetType
	}
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ports.ErrReportTargetNotFound
	}
	return nil
}
