package repositories

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"core/application/ports"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type profileTransactionPool struct {
	tx         *profileTransaction
	beginCount int
}

func (p *profileTransactionPool) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}

func (p *profileTransactionPool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.tx.ExecContext(ctx, query, args...)
}

func (p *profileTransactionPool) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (p *profileTransactionPool) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (p *profileTransactionPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	p.beginCount++
	return p.tx, nil
}

type profileTransaction struct {
	executedSQL   []string
	failLocation  error
	commitCount   int
	rollbackCount int
}

type profileSQLResult struct{}

func (profileSQLResult) LastInsertId() (int64, error) { return 1, nil }
func (profileSQLResult) RowsAffected() (int64, error) { return 1, nil }

func (tx *profileTransaction) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("unexpected prepare")
}

func (tx *profileTransaction) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	tx.executedSQL = append(tx.executedSQL, query)
	if tx.failLocation != nil && strings.Contains(strings.ToLower(query), `insert into "locations"`) {
		return nil, tx.failLocation
	}
	return profileSQLResult{}, nil
}

func (tx *profileTransaction) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected query")
}

func (tx *profileTransaction) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return &sql.Row{}
}

func (tx *profileTransaction) Commit() error {
	tx.commitCount++
	return nil
}

func (tx *profileTransaction) Rollback() error {
	tx.rollbackCount++
	return nil
}

func newProfileTransactionDB(t *testing.T, pool *profileTransactionPool) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(postgres.New(postgres.Config{
		Conn:             pool,
		WithoutReturning: true,
	}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open profile transaction database: %v", err)
	}
	return database
}

func TestUpdateUserProfileRollsBackUserWhenLocationPersistenceFails(t *testing.T) {
	locationErr := errors.New("forced location failure")
	tx := &profileTransaction{failLocation: locationErr}
	pool := &profileTransactionPool{tx: tx}
	repository := NewUserRepository(newProfileTransactionDB(t, pool), nil, nil, nil, nil)
	userID := uuid.New()
	username := "new-name"

	err := repository.UpdateUserProfile(context.Background(), ports.UserProfileUpdate{
		UserID:   userID,
		UserName: &username,
		Location: &ports.UserProfileLocationUpdate{Latitude: 41.0082, Longitude: 28.9784},
	})
	if !errors.Is(err, locationErr) {
		t.Fatalf("UpdateUserProfile() error = %v, want %v", err, locationErr)
	}
	if pool.beginCount != 1 || tx.rollbackCount != 1 || tx.commitCount != 0 {
		t.Fatalf("transaction counts = begin:%d rollback:%d commit:%d", pool.beginCount, tx.rollbackCount, tx.commitCount)
	}
	joinedSQL := strings.ToLower(strings.Join(tx.executedSQL, "\n"))
	if !strings.Contains(joinedSQL, `update "users"`) || !strings.Contains(joinedSQL, `insert into "locations"`) {
		t.Fatalf("expected user then location writes inside transaction, SQL=%s", joinedSQL)
	}
}

func TestUpdateUserProfileCommitsUserAndLocationTogether(t *testing.T) {
	tx := &profileTransaction{}
	pool := &profileTransactionPool{tx: tx}
	repository := NewUserRepository(newProfileTransactionDB(t, pool), nil, nil, nil, nil)
	email := "new@example.com"
	website := "https://example.com"

	err := repository.UpdateUserProfile(context.Background(), ports.UserProfileUpdate{
		UserID:  uuid.New(),
		Email:   &email,
		Website: &website,
		Location: &ports.UserProfileLocationUpdate{
			CountryCode: "TR",
			City:        "Istanbul",
			Latitude:    41.0082,
			Longitude:   28.9784,
		},
	})
	if err != nil {
		t.Fatalf("UpdateUserProfile() error = %v", err)
	}
	if pool.beginCount != 1 || tx.rollbackCount != 0 || tx.commitCount != 1 {
		t.Fatalf("transaction counts = begin:%d rollback:%d commit:%d", pool.beginCount, tx.rollbackCount, tx.commitCount)
	}
}
