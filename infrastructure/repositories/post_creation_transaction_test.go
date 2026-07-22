package repositories

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"core/application/ports"
	domainpost "core/domain/post"
	"core/helpers"
	"core/models"
	"core/models/media"
	"core/models/post"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type postCreationUploadedFile struct {
	name        string
	contentType string
	body        []byte
}

func (f postCreationUploadedFile) Filename() string    { return f.name }
func (f postCreationUploadedFile) Size() int64         { return int64(len(f.body)) }
func (f postCreationUploadedFile) ContentType() string { return f.contentType }
func (f postCreationUploadedFile) Open() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(f.body)), nil
}

type recordingTransactionPool struct {
	tx         *recordingTransaction
	beginErr   error
	beginCount int
	queryCount int
	queryErr   error
}

func (p *recordingTransactionPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}

func (p *recordingTransactionPool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.tx.ExecContext(ctx, query, args...)
}

func (p *recordingTransactionPool) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	p.queryCount++
	if p.queryErr != nil {
		return nil, p.queryErr
	}
	return nil, errors.New("unexpected query")
}

func (p *recordingTransactionPool) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (p *recordingTransactionPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	p.beginCount++
	if p.beginErr != nil {
		return nil, p.beginErr
	}
	return p.tx, nil
}

type recordingTransaction struct {
	executedSQL     []string
	executedArgs    [][]any
	commitCount     int
	rollbackCount   int
	queryCount      int
	mediaInserted   bool
	panicAfterMedia bool
}

type successfulSQLResult struct{}

func (successfulSQLResult) LastInsertId() (int64, error) { return 1, nil }
func (successfulSQLResult) RowsAffected() (int64, error) { return 1, nil }

func (tx *recordingTransaction) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}

func (tx *recordingTransaction) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	if tx.panicAfterMedia && tx.mediaInserted {
		panic("forced failure after media persistence")
	}
	tx.executedSQL = append(tx.executedSQL, query)
	tx.executedArgs = append(tx.executedArgs, append([]any(nil), args...))
	if strings.Contains(strings.ToLower(query), `insert into "medias"`) {
		tx.mediaInserted = true
	}
	return successfulSQLResult{}, nil
}

func TestAddMediaExplicitlyPersistsProtectedVisibility(t *testing.T) {
	t.Chdir(t.TempDir())
	tx := &recordingTransaction{}
	pool := &recordingTransactionPool{tx: tx}
	database := newRecordingPostCreationDB(t, pool)
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMediaRepository(database, node)

	created, err := repo.AddMedia(uuid.New(), media.OwnerUser, uuid.New(), media.RolePrivatePhoto, postCreationFile())
	if err != nil {
		t.Fatalf("AddMedia(private photo) error = %v", err)
	}
	if created == nil || created.IsPublic {
		t.Fatalf("AddMedia(private photo) visibility = %#v, want protected", created)
	}

	foundExplicitFalse := false
	for index, query := range tx.executedSQL {
		normalizedQuery := strings.ToLower(query)
		if !strings.Contains(normalizedQuery, `update "medias" set`) || !strings.Contains(normalizedQuery, `"is_public"=`) {
			continue
		}
		for _, argument := range tx.executedArgs[index] {
			if value, ok := argument.(bool); ok && !value {
				foundExplicitFalse = true
			}
		}
	}
	if !foundExplicitFalse {
		t.Fatalf("protected media insert did not explicitly persist is_public=false: SQL=%v args=%v", tx.executedSQL, tx.executedArgs)
	}
}

func (tx *recordingTransaction) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	tx.queryCount++
	return nil, errors.New("unexpected query")
}

func (tx *recordingTransaction) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (tx *recordingTransaction) Commit() error {
	tx.commitCount++
	return nil
}

func (tx *recordingTransaction) Rollback() error {
	tx.rollbackCount++
	return nil
}

func newRecordingPostCreationDB(t *testing.T, pool *recordingTransactionPool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:             pool,
		WithoutReturning: true,
	}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open recording database: %v", err)
	}
	return db
}

func newPostCreationRepository(t *testing.T, db *gorm.DB) *PostRepository {
	t.Helper()
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatalf("create snowflake node: %v", err)
	}
	mediaRepo := NewMediaRepository(db, node)
	userRepo := &UserRepository{db: db}
	return NewPostRepository(db, node, mediaRepo, userRepo, nil)
}

func postCreationAuthor() *models.User {
	return &models.User{
		ID:              uuid.New(),
		PublicID:        42,
		Domain:          models.CoolVibes,
		DefaultLanguage: "en",
	}
}

func postCreationFile() ports.UploadedFile {
	return postCreationUploadedFile{
		name:        "rollback.txt",
		contentType: "text/plain",
		body:        []byte("rollback me"),
	}
}

func storedUploadFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir("static/uploads", func(path string, entry fs.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("walk stored uploads: %v", err)
	}
	return files
}

func TestAddMediaRemovesStoredFileWhenPersistenceCannotBegin(t *testing.T) {
	t.Chdir(t.TempDir())
	persistErr := errors.New("forced begin failure")
	tx := &recordingTransaction{}
	pool := &recordingTransactionPool{tx: tx, beginErr: persistErr}
	db := newRecordingPostCreationDB(t, pool)
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatal(err)
	}
	repo := NewMediaRepository(db, node)

	_, err = repo.AddMedia(uuid.New(), media.OwnerPost, uuid.New(), media.RolePost, postCreationFile())
	if !errors.Is(err, persistErr) {
		t.Fatalf("AddMedia() error = %v, want %v", err, persistErr)
	}
	if files := storedUploadFiles(t); len(files) != 0 {
		t.Fatalf("failed media persistence left files behind: %v", files)
	}
}

func TestCreateContentablePostLateFailureRollsBackSharedTransactionAndMedia(t *testing.T) {
	t.Chdir(t.TempDir())
	tx := &recordingTransaction{}
	pool := &recordingTransactionPool{tx: tx}
	repo := newPostCreationRepository(t, newRecordingPostCreationDB(t, pool))

	_, err := repo.CreateContentablePost(context.Background(), ports.FormData{
		Values: map[string][]string{
			"title":  {"transaction rollback"},
			"extras": {"{"},
		},
		Files: []ports.UploadedFile{postCreationFile()},
	}, postCreationAuthor(), string(post.PostKindPost), nil)
	if err == nil {
		t.Fatal("CreateContentablePost() error = nil, want invalid extras failure")
	}
	if !tx.mediaInserted {
		t.Fatal("test did not reach media persistence before the late failure")
	}
	if pool.beginCount != 1 {
		t.Fatalf("database transaction begins = %d, want one shared transaction", pool.beginCount)
	}
	if tx.commitCount != 0 || tx.rollbackCount != 1 {
		t.Fatalf("commit/rollback counts = %d/%d, want 0/1", tx.commitCount, tx.rollbackCount)
	}
	if files := storedUploadFiles(t); len(files) != 0 {
		t.Fatalf("rolled back post left files behind: %v", files)
	}
}

func TestCreateContentablePostRejectsInvalidPollDefinitionAtomically(t *testing.T) {
	tests := []struct {
		name       string
		pollValues map[string][]string
		wantErr    error
	}{
		{
			name: "unknown kind",
			pollValues: map[string][]string{
				"polls[0].question":   {"Question"},
				"polls[0].kind":       {"surprise"},
				"polls[0].options[0]": {"A"},
				"polls[0].options[1]": {"B"},
			},
			wantErr: domainpost.ErrInvalidPollKind,
		},
		{
			name: "invalid maximum",
			pollValues: map[string][]string{
				"polls[0].question":       {"Question"},
				"polls[0].kind":           {"multiple"},
				"polls[0].max_selectable": {"3"},
				"polls[0].options[0]":     {"A"},
				"polls[0].options[1]":     {"B"},
			},
			wantErr: domainpost.ErrInvalidPollMaximum,
		},
		{
			name: "duplicate options",
			pollValues: map[string][]string{
				"polls[0].question":   {"Question"},
				"polls[0].kind":       {"single"},
				"polls[0].options[0]": {"Same"},
				"polls[0].options[1]": {" same "},
			},
			wantErr: domainpost.ErrDuplicatePollOption,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			tx := &recordingTransaction{}
			pool := &recordingTransactionPool{tx: tx}
			repository := newPostCreationRepository(t, newRecordingPostCreationDB(t, pool))
			values := map[string][]string{"title": {"poll validation"}}
			for key, value := range test.pollValues {
				values[key] = value
			}

			_, err := repository.CreateContentablePost(context.Background(), ports.FormData{Values: values}, postCreationAuthor(), string(post.PostKindPost), nil)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("CreateContentablePost() error = %v; want %v", err, test.wantErr)
			}
			if tx.commitCount != 0 || tx.rollbackCount != 1 {
				t.Fatalf("invalid poll commit/rollback = %d/%d; want 0/1", tx.commitCount, tx.rollbackCount)
			}
		})
	}
}

func TestCreateContentablePostPanicRollsBackAndCompensatesMedia(t *testing.T) {
	t.Chdir(t.TempDir())
	tx := &recordingTransaction{panicAfterMedia: true}
	pool := &recordingTransactionPool{tx: tx}
	repo := newPostCreationRepository(t, newRecordingPostCreationDB(t, pool))

	var recovered any
	func() {
		defer func() { recovered = recover() }()
		_, _ = repo.CreateContentablePost(context.Background(), ports.FormData{
			Values: map[string][]string{"title": {"panic rollback"}},
			Files:  []ports.UploadedFile{postCreationFile()},
		}, postCreationAuthor(), string(post.PostKindPost), nil)
	}()
	if recovered == nil {
		t.Fatal("CreateContentablePost() did not propagate the forced panic")
	}
	if tx.rollbackCount != 1 || tx.commitCount != 0 {
		t.Fatalf("commit/rollback counts after panic = %d/%d, want 0/1", tx.commitCount, tx.rollbackCount)
	}
	if files := storedUploadFiles(t); len(files) != 0 {
		t.Fatalf("panic rollback left files behind: %v", files)
	}
}

func TestPostRepositoryTransactionScopeClonesDBBackedCollaborators(t *testing.T) {
	originalPool := &recordingTransactionPool{tx: &recordingTransaction{}}
	scopedPool := &recordingTransactionPool{tx: &recordingTransaction{}}
	originalDB := newRecordingPostCreationDB(t, originalPool)
	scopedDB := newRecordingPostCreationDB(t, scopedPool)
	repo := newPostCreationRepository(t, originalDB)

	scoped := repo.transactionScoped(scopedDB)
	if scoped == repo || scoped.mediaRepo == repo.mediaRepo || scoped.userRepo == repo.userRepo {
		t.Fatal("transactionScoped() mutated or reused a DB-backed repository instance")
	}
	if scoped.db != scopedDB || scoped.mediaRepo.db != scopedDB || scoped.userRepo.db != scopedDB {
		t.Fatal("transaction-scoped repositories do not share the supplied transaction")
	}
	if repo.db != originalDB || repo.mediaRepo.db != originalDB || repo.userRepo.db != originalDB {
		t.Fatal("transactionScoped() mutated the original repository graph")
	}
}

func TestPostCommitCommentSideEffectsDoNotOpenAnEngagementTransaction(t *testing.T) {
	notificationLookupErr := errors.New("notification lookup unavailable")
	pool := &recordingTransactionPool{tx: &recordingTransaction{}, queryErr: notificationLookupErr}
	db := newRecordingPostCreationDB(t, pool)
	engagementRepo := NewEngagementRepository(db)
	repo := &PostRepository{userRepo: &UserRepository{db: db, engagementRepo: engagementRepo}}
	author := postCreationAuthor()
	creation := &contentablePostCreation{
		post: &post.Post{ID: uuid.New(), PublicID: 200},
		parentPost: &post.Post{
			ID:       uuid.New(),
			PublicID: 100,
			AuthorID: uuid.New(),
		},
	}

	// Engagement is part of the comment creation transaction now. Post-commit
	// work may query notification recipients, but must never begin a second
	// aggregate transaction.
	repo.runPostCommitCommentSideEffects(context.Background(), author, creation)
	if pool.beginCount != 0 {
		t.Fatalf("post-commit transaction begins = %d; want 0", pool.beginCount)
	}
	if pool.queryCount+pool.tx.queryCount == 0 {
		t.Fatal("notification lookup was not attempted")
	}
}

func TestRemoveStoredUploadRefusesOutsidePath(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("static/uploads", 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join("outside.txt")
	if err := os.WriteFile(outside, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := removeStoredUpload(outside)
	if err == nil || !strings.Contains(err.Error(), "outside upload root") {
		t.Fatalf("removeStoredUpload() error = %v, want path safety error", err)
	}
	if _, statErr := os.Stat(outside); statErr != nil {
		t.Fatalf("outside file was removed: %v", statErr)
	}
}
