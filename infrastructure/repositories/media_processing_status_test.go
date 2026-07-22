package repositories

import (
	"strings"
	"testing"

	modelmedia "core/models/media"
	modelutils "core/models/utils"

	"github.com/google/uuid"
)

func TestProcessClaimedMediaMutatesTerminalStatusOnlyAfterPersistence(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		tx := &recordingTransaction{}
		repository := NewMediaRepository(newRecordingPostCreationDB(t, &recordingTransactionPool{tx: tx}), nil)
		item := &modelmedia.Media{
			ID:               uuid.New(),
			FileID:           uuid.New(),
			ProcessingStatus: modelmedia.ProcessingStatusProcessing,
			File: modelutils.FileMetadata{
				MimeType: "application/octet-stream",
			},
		}

		if err := repository.ProcessClaimedMedia(item); err != nil {
			t.Fatalf("ProcessClaimedMedia() error = %v", err)
		}
		if item.ProcessingStatus != modelmedia.ProcessingStatusReady {
			t.Fatalf("item status = %q, want %q", item.ProcessingStatus, modelmedia.ProcessingStatusReady)
		}
		if tx.commitCount != 1 {
			t.Fatalf("ready transaction commits = %d, want 1", tx.commitCount)
		}
	})

	t.Run("failed", func(t *testing.T) {
		t.Chdir(t.TempDir())
		tx := &recordingTransaction{}
		repository := NewMediaRepository(newRecordingPostCreationDB(t, &recordingTransactionPool{tx: tx}), nil)
		item := &modelmedia.Media{
			ID:               uuid.New(),
			FileID:           uuid.New(),
			ProcessingStatus: modelmedia.ProcessingStatusProcessing,
			File: modelutils.FileMetadata{
				MimeType:    "image/jpeg",
				StoragePath: "missing-private-photo.jpg",
			},
		}

		err := repository.ProcessClaimedMedia(item)
		if err == nil {
			t.Fatal("ProcessClaimedMedia(missing source) error = nil")
		}
		if item.ProcessingStatus != modelmedia.ProcessingStatusFailed {
			t.Fatalf("item status = %q, want %q", item.ProcessingStatus, modelmedia.ProcessingStatusFailed)
		}
		foundFailedUpdate := false
		for _, query := range tx.executedSQL {
			if strings.Contains(strings.ToLower(query), `update "medias"`) {
				foundFailedUpdate = true
				break
			}
		}
		if !foundFailedUpdate {
			t.Fatalf("failed status was not persisted: %v", tx.executedSQL)
		}
	})
}
