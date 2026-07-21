package repositories

import (
	"context"
	"core/application/ports"
	"core/models"
	"errors"
	"fmt"
	"strings"

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
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return ports.ErrInvalidReportKind
	}

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
		Reason:          strings.TrimSpace(description),
		Status:          models.ReportStatusPending,
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockKey := fmt.Sprintf("report:%s:%s:%s", reporterID, contentableType, contentableID)
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, 0))", lockKey).Error; err != nil {
			return err
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
