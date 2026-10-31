package repositories

import (
	"bytes"
	"context"
	upload "core/adapters/media"
	"core/application/ports"
	"core/application/usecases"
	"core/constants"
	domainmedia "core/domain/media"
	domainuser "core/domain/user"
	"core/helpers"
	"core/models"
	modelmedia "core/models/media"
	"core/models/notifications"
	modelutils "core/models/utils"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// Use a separate schema and real connections so row locks and concurrent
// commits are exercised without touching the application's tables.
func privatePhotoIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not configured")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	schema := "photo_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		t.Fatal(err)
	}
	if err := admin.Exec("CREATE SCHEMA " + schema).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = admin.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		pool, _ := admin.DB()
		_ = pool.Close()
	})
	if strings.Contains(dsn, "://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		query := parsed.Query()
		query.Set("search_path", schema+",public")
		parsed.RawQuery = query.Encode()
		dsn = parsed.String()
	} else {
		dsn += " search_path=" + schema + ",public"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Silent),
		DisableForeignKeyConstraintWhenMigrating: true,
		IgnoreRelationshipsWhenMigrating:         true,
	})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := db.DB()
	pool.SetMaxOpenConns(20)
	t.Cleanup(func() { _ = pool.Close() })
	if err := db.AutoMigrate(&models.User{}, &modelutils.FileMetadata{}, &modelmedia.Media{},
		&models.ReferralReward{}, &models.BroadcastSession{}, &models.BroadcastGift{}, &models.VapidKey{}, &models.Engagement{}, &models.TransactionLock{}, &models.EngagementDetail{}, &notifications.Notification{}, &notifications.Delivery{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE posts (id uuid PRIMARY KEY, author_id uuid, post_kind text,
		contentable_type text, contentable_id uuid, published boolean, audience text, deleted_at timestamptz)`).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func privatePhotoTestRepository(t *testing.T, db *gorm.DB) *PrivatePhotoRepository {
	t.Helper()
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatal(err)
	}
	return NewPrivatePhotoRepository(db, node, NewMediaRepository(db, node))
}

func privatePhotoTestUsers(t *testing.T, db *gorm.DB) (models.User, models.User) {
	t.Helper()
	owner := models.User{ID: uuid.New(), PublicID: 10, UserRole: constants.UserRoleUser, UserName: "owner"}
	viewer := models.User{ID: uuid.New(), PublicID: 20, UserRole: constants.UserRoleUser, UserName: "viewer"}
	if err := db.Omit(clause.Associations).Create(&[]models.User{owner, viewer}).Error; err != nil {
		t.Fatal(err)
	}
	return owner, viewer
}

func privatePhotoPNG(t *testing.T) *domainmedia.UploadedFile {
	t.Helper()
	var body bytes.Buffer
	if err := png.Encode(&body, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}
	return upload.FromBytes("photo.png", "image/png", body.Bytes())
}

func TestPrivatePhotoFailedProcessingDoesNotConsumeUploadQuotaIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	owner, _ := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	first, err := repo.AddPrivatePhoto(context.Background(), owner.ID, privatePhotoPNG(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Model(first).Update("processing_status", modelmedia.ProcessingStatusFailed).Error; err != nil {
		t.Fatal(err)
	}
	count, err := repo.CountPrivatePhotos(context.Background(), owner.ID)
	if err != nil || count != 0 {
		t.Fatalf("quota count = %d, %v; want 0", count, err)
	}
	if _, err := repo.AddPrivatePhoto(context.Background(), owner.ID, privatePhotoPNG(t), 1); err != nil {
		t.Fatalf("replacement for failed upload: %v", err)
	}
}

func TestPrivatePhotoMoveIsIdempotentAtQuotaIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	photo, err := repo.AddPrivatePhoto(context.Background(), owner.ID, privatePhotoPNG(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.MoveProfilePhoto(context.Background(), owner.ID, photo.PublicID, modelmedia.RolePrivatePhoto, 1); err != nil {
		t.Fatalf("repeat private move at quota: %v", err)
	}
	if _, err := repo.MoveProfilePhoto(context.Background(), viewer.ID, photo.PublicID, modelmedia.RoleProfile, 1); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("another user's photo move = %v", err)
	}
}

func TestPrivatePhotoAccessLifecycleIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	service := usecases.NewPrivatePhotoService(repo, nil, nil)
	principal := ports.PrivatePhotoPrincipal{ID: viewer.ID, PublicID: viewer.PublicID}
	ownerPrincipal := ports.PrivatePhotoPrincipal{ID: owner.ID, PublicID: owner.PublicID}
	ctx := context.Background()
	photo, err := repo.AddPrivatePhoto(ctx, owner.ID, privatePhotoPNG(t), 10)
	if err != nil {
		t.Fatal(err)
	}
	assertVisible := func(want int) {
		t.Helper()
		result, err := service.Fetch(ctx, principal, owner.PublicID)
		if err != nil || len(result.Photos) != want {
			t.Fatalf("visible photos = %+v, %v; want %d", result, err, want)
		}
	}
	assertVisible(0)
	request, err := service.RequestAccess(ctx, principal, owner.PublicID)
	if err != nil || request.Status != "pending" {
		t.Fatalf("request = %+v, %v", request, err)
	}
	assertVisible(0)
	record, err := repo.GetPrivatePhotoAccess(ctx, owner.ID, viewer.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RespondAccess(ctx, principal, record.PublicID, "approved"); !errors.Is(err, usecases.ErrPrivatePhotoForbidden) {
		t.Fatalf("viewer approved own request: %v", err)
	}
	if _, err := service.RespondAccess(ctx, ownerPrincipal, record.PublicID, "approved"); err != nil {
		t.Fatal(err)
	}
	assertVisible(1)
	if _, err := service.RevokeAccess(ctx, ownerPrincipal, viewer.PublicID); err != nil {
		t.Fatal(err)
	}
	assertVisible(0)
	if _, err := service.RequestAccess(ctx, principal, owner.PublicID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.RespondAccess(ctx, ownerPrincipal, record.PublicID, "denied"); err != nil {
		t.Fatal(err)
	}
	assertVisible(0)
	if _, err := service.MoveProfilePhoto(ctx, ownerPrincipal, photo.PublicID, "public"); err != nil {
		t.Fatal(err)
	}
	public, err := service.FetchProfilePhotos(ctx, principal, owner.PublicID)
	if err != nil || len(public.Photos) != 1 {
		t.Fatalf("public photos = %+v, %v", public, err)
	}
	assertVisible(0)
	if _, err := service.MoveProfilePhoto(ctx, ownerPrincipal, photo.PublicID, "private"); err != nil {
		t.Fatal(err)
	}
	public, err = service.FetchProfilePhotos(ctx, principal, owner.PublicID)
	if err != nil || len(public.Photos) != 0 {
		t.Fatalf("private photo leaked into public album = %+v, %v", public, err)
	}
}

func TestPrivatePhotoConcurrentRequestsAreIdempotentIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	changed := make(chan bool, 12)
	for i := 0; i < 12; i++ {
		wg.Go(func() {
			record, didChange, err := repo.RequestPrivatePhotoAccess(context.Background(), owner.ID, viewer.ID, time.Now().UTC())
			if err == nil && record.Status != domainmedia.PrivatePhotoAccessPending {
				err = errors.New("concurrent request did not remain pending")
			}
			errs <- err
			changed <- didChange
		})
	}
	wg.Wait()
	close(errs)
	close(changed)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	changes := 0
	for didChange := range changed {
		if didChange {
			changes++
		}
	}
	if changes != 1 {
		t.Fatalf("request mutations = %d; want 1", changes)
	}
}

func TestPrivatePhotoDeletedOwnerCannotBeViewedByStoredGrantIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	ctx := context.Background()
	request, _, err := repo.RequestPrivatePhotoAccess(ctx, owner.ID, viewer.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.RespondPrivatePhotoAccess(ctx, request.PublicID, owner.ID, domainmedia.PrivatePhotoAccessApproved, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&owner).Error; err != nil {
		t.Fatal(err)
	}
	allowed, err := repo.HasApprovedPrivatePhotoAccess(ctx, owner.ID, viewer.ID)
	if err != nil || allowed {
		t.Fatalf("deleted owner's stored grant = %v, %v; want denied", allowed, err)
	}
}

func TestPrivatePhotoBlockRevokesBothDirectionsAtomicallyIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	ctx := context.Background()
	for _, pair := range [][2]uuid.UUID{{owner.ID, viewer.ID}, {viewer.ID, owner.ID}} {
		request, _, err := repo.RequestPrivatePhotoAccess(ctx, pair[0], pair[1], time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := repo.RespondPrivatePhotoAccess(ctx, request.PublicID, pair[0], domainmedia.PrivatePhotoAccessApproved, time.Now().UTC()); err != nil {
			t.Fatal(err)
		}
	}
	engagements := NewEngagementRepository(db)
	for _, enabled := range []bool{true, false} {
		intent, _ := domainuser.NewSetInteractionState(domainuser.InteractionBlock, enabled)
		if _, err := engagements.ApplyReciprocalUserInteraction(ctx, owner.ID, viewer.ID, intent); err != nil {
			t.Fatal(err)
		}
	}
	for _, pair := range [][2]uuid.UUID{{owner.ID, viewer.ID}, {viewer.ID, owner.ID}} {
		allowed, err := repo.HasApprovedPrivatePhotoAccess(ctx, pair[0], pair[1])
		if err != nil || allowed {
			t.Fatalf("old grant revived after unblock = %v, %v", allowed, err)
		}
	}
}

func TestPrivatePhotoConcurrentUploadsRespectQuotaIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	owner, _ := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	file := privatePhotoPNG(t)
	results := make(chan error, 6)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Go(func() {
			_, err := repo.AddPrivatePhoto(context.Background(), owner.ID, file, 2)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if !errors.Is(err, ports.ErrPrivatePhotoLimitReached) {
			t.Fatal(err)
		}
	}
	if success != 2 {
		t.Fatalf("concurrent uploads accepted = %d; want 2", success)
	}
}

func TestPrivatePhotoConcurrentApprovalAndBlockCannotRestoreGrantIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	engagements := NewEngagementRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	block, _ := domainuser.NewSetInteractionState(domainuser.InteractionBlock, true)
	unblock, _ := domainuser.NewSetInteractionState(domainuser.InteractionBlock, false)
	for attempt := 0; attempt < 12; attempt++ {
		request, _, err := repo.RequestPrivatePhotoAccess(ctx, owner.ID, viewer.ID, time.Now().UTC())
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		approval := make(chan error, 1)
		blocking := make(chan error, 1)
		go func() {
			<-start
			_, _, err := repo.RespondPrivatePhotoAccess(ctx, request.PublicID, owner.ID, domainmedia.PrivatePhotoAccessApproved, time.Now().UTC())
			approval <- err
		}()
		go func() {
			<-start
			_, err := engagements.ApplyReciprocalUserInteraction(ctx, viewer.ID, owner.ID, block)
			blocking <- err
		}()
		close(start)
		if err := <-blocking; err != nil {
			t.Fatal(err)
		}
		if err := <-approval; err != nil && !errors.Is(err, ports.ErrPrivatePhotoAccessBlocked) {
			t.Fatal(err)
		}
		if _, _, err := repo.RequestPrivatePhotoAccess(ctx, owner.ID, viewer.ID, time.Now().UTC()); !errors.Is(err, ports.ErrPrivatePhotoAccessBlocked) {
			t.Fatalf("request while blocked = %v", err)
		}
		if _, err := engagements.ApplyReciprocalUserInteraction(ctx, viewer.ID, owner.ID, unblock); err != nil {
			t.Fatal(err)
		}
		allowed, err := repo.HasApprovedPrivatePhotoAccess(ctx, owner.ID, viewer.ID)
		if err != nil || allowed {
			t.Fatalf("concurrent approval survived block/unblock = %v, %v", allowed, err)
		}
	}
}

func TestPrivatePhotoGrantRevocationFailureRollsBackBlockIntegration(t *testing.T) {
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	ctx := context.Background()
	request, _, err := repo.RequestPrivatePhotoAccess(ctx, owner.ID, viewer.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.RespondPrivatePhotoAccess(ctx, request.PublicID, owner.ID, domainmedia.PrivatePhotoAccessApproved, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE FUNCTION reject_photo_grant_update() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'forced grant update failure'; END $$`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TRIGGER reject_photo_grant_update BEFORE UPDATE ON engagement_details
		FOR EACH ROW WHEN (OLD.kind = 'private_photo_access') EXECUTE FUNCTION reject_photo_grant_update()`).Error; err != nil {
		t.Fatal(err)
	}
	block, _ := domainuser.NewSetInteractionState(domainuser.InteractionBlock, true)
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(ctx, viewer.ID, owner.ID, block); err == nil {
		t.Fatal("block committed despite grant revocation failure")
	}
	blocked, err := repo.ArePrivatePhotoUsersBlocked(ctx, owner.ID, viewer.ID)
	if err != nil || blocked {
		t.Fatalf("failed transaction persisted block = %v, %v", blocked, err)
	}
}

func TestPrivatePhotoProcessedURLsUseCurrentGrantIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	ctx := context.Background()
	photo, err := repo.AddPrivatePhoto(ctx, owner.ID, privatePhotoPNG(t), 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.mediaRepo.ProcessClaimedMedia(photo); err != nil {
		t.Fatalf("process real PNG upload: %v", err)
	}
	service := usecases.NewPrivatePhotoService(repo, nil, nil)
	ownerPrincipal := ports.PrivatePhotoPrincipal{ID: owner.ID, PublicID: owner.PublicID}
	result, err := service.Fetch(ctx, ownerPrincipal, owner.PublicID)
	if err != nil || len(result.Photos) != 1 || result.Photos[0].File.URL == "" {
		t.Fatalf("processed album = %+v, %v", result, err)
	}
	variantPath := "." + result.Photos[0].File.URL
	if _, err := os.Stat(variantPath); err != nil {
		t.Fatalf("processed URL has no file: %v", err)
	}
	prefix := strings.TrimSuffix(photo.File.StoragePath, filepath.Ext(photo.File.StoragePath))
	access := usecases.NewMediaAccessService(repo.mediaRepo, repo)
	check := func(path string, viewerID *int64, want bool) usecases.MediaAccessDecision {
		t.Helper()
		decision, err := access.Authorize(ctx, prefix, viewerID, path)
		if err != nil || decision.Allowed != want {
			t.Fatalf("URL authorization = %+v, %v; want allowed=%v", decision, err, want)
		}
		return decision
	}
	check(variantPath, nil, false)
	check(variantPath, &viewer.PublicID, false)
	request, _, err := repo.RequestPrivatePhotoAccess(ctx, owner.ID, viewer.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.RespondPrivatePhotoAccess(ctx, request.PublicID, owner.ID, domainmedia.PrivatePhotoAccessApproved, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	check(variantPath, &viewer.PublicID, true)
	check(photo.File.StoragePath, &viewer.PublicID, false)
	if _, _, err := repo.RevokePrivatePhotoAccess(ctx, owner.ID, viewer.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	check(variantPath, &viewer.PublicID, false)
	if _, err := service.MoveProfilePhoto(ctx, ownerPrincipal, photo.PublicID, "public"); err != nil {
		t.Fatal(err)
	}
	if decision := check(variantPath, nil, true); !decision.NoStore {
		t.Fatal("public photo that can move to private is cacheable")
	}
	if _, err := service.MoveProfilePhoto(ctx, ownerPrincipal, photo.PublicID, "private"); err != nil {
		t.Fatal(err)
	}
	check(variantPath, nil, false)
	check(variantPath, &viewer.PublicID, false)
}

func TestPrivatePhotoUploadValidatesRealImageAndDimensionsIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	owner, _ := privatePhotoTestUsers(t, db)
	repo := privatePhotoTestRepository(t, db)
	valid := privatePhotoPNG(t)
	source, err := valid.Open()
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	oversized := append([]byte(nil), body...)
	binary.BigEndian.PutUint32(oversized[16:20], 12001)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	for _, test := range []struct {
		name string
		file *domainmedia.UploadedFile
		want error
	}{
		{"fake image", upload.FromBytes("photo.png", "image/png", []byte("not an image")), domainmedia.ErrPrivatePhotoImageRequired},
		{"wrong MIME", upload.FromBytes("photo.jpg", "image/jpeg", body), domainmedia.ErrPrivatePhotoImageRequired},
		{"oversized dimensions", upload.FromBytes("photo.png", "image/png", oversized), domainmedia.ErrPrivatePhotoImageDimensions},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := repo.AddPrivatePhoto(context.Background(), owner.ID, test.file, 10); !errors.Is(err, test.want) {
				t.Fatalf("invalid upload = %v; want %v", err, test.want)
			}
		})
	}
	if files := storedUploadFiles(t); len(files) != 0 {
		t.Fatalf("rejected uploads left files behind: %v", files)
	}
}
