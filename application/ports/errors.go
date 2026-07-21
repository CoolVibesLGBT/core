package ports

import "errors"

var ErrNotFound = errors.New("resource not found")

var (
	ErrInvalidReportKind       = errors.New("invalid report kind")
	ErrReportTargetNotFound    = errors.New("report target not found")
	ErrReportNotFound          = errors.New("report not found")
	ErrInvalidReportTransition = errors.New("invalid report status transition")
	ErrInvalidModerationAction = errors.New("invalid moderation action")
)
