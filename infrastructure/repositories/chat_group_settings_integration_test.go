package repositories

import (
	"context"
	"core/adapters/media"
	"core/application/usecases"
	"core/constants"
	"core/helpers"
	"core/models"
	"core/models/chat"
	"core/models/post"
	modelutils "core/models/utils"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm/clause"
	"strings"
	"testing"
)

func TestChatGroupSettingsAndInvitesIntegration(t *testing.T) {
	t.Chdir(t.TempDir())
	db := privatePhotoIntegrationDB(t)
	if err := db.AutoMigrate(&post.Post{}, &chat.Chat{}, &chat.ChatParticipant{}, &chat.GroupInvite{}); err != nil {
		t.Fatal(err)
	}
	owner, member := privatePhotoTestUsers(t, db)
	outsider := models.User{ID: uuid.New(), PublicID: 30, UserName: "joiner", UserRole: constants.UserRoleUser}
	if err := db.Omit(clause.Associations).Create(&outsider).Error; err != nil {
		t.Fatal(err)
	}
	node, err := helpers.NewDefaultNode()
	if err != nil {
		t.Fatal(err)
	}
	repository := NewChatRepository(db, node, nil, nil, nil)
	ctx := context.Background()
	title, description := modelutils.MakeLocalizedString("en", "Photo Group"), modelutils.MakeLocalizedString("en", "Group details")
	entity, err := repository.CreateGroupChatWithAvatar(ctx, owner.ID, []uuid.UUID{member.ID}, title, description, privatePhotoPNG(t))
	if err != nil {
		t.Fatal(err)
	}
	if entity.Avatar == nil || entity.AvatarID == nil || entity.Avatar.OwnerID != entity.ID || entity.Avatar.IsPublic || entity.Visibility != "private" {
		t.Fatal("group avatar/default visibility not persisted")
	}
	prefix := strings.TrimSuffix(entity.Avatar.File.StoragePath, ".png")
	mediaRepository := NewMediaRepository(db, node)
	access := usecases.NewMediaAccessService(mediaRepository)
	allowed, err := access.Authorize(ctx, prefix, &member.PublicID)
	if err != nil || !allowed.Allowed {
		t.Fatalf("member cannot view group avatar: %+v, %v", allowed, err)
	}
	allowed, err = access.Authorize(ctx, prefix, &outsider.PublicID)
	if err != nil || allowed.Allowed {
		t.Fatalf("private group avatar leaked: %+v, %v", allowed, err)
	}
	if _, err := repository.UpdateGroup(ctx, &member, entity.ID, "Forbidden", "", nil, false); !errors.Is(err, chat.ErrPermissionDenied) {
		t.Fatalf("member edited group: %v", err)
	}
	if _, err := repository.UpdateGroup(ctx, &owner, entity.ID, "Photo Group", "", media.FromBytes("bad.png", "image/png", []byte("invalid")), false); !errors.Is(err, chat.ErrInvalidGroupAvatar) {
		t.Fatalf("invalid image accepted: %v", err)
	}
	invitation, err := repository.ExportGroupInvite(ctx, &owner, entity.ID, false)
	if err != nil || len(invitation.Token) != 43 {
		t.Fatalf("export invite: %v", err)
	}
	again, err := repository.ExportGroupInvite(ctx, &owner, entity.ID, false)
	if err != nil || again.Token != invitation.Token {
		t.Fatal("invite changed without revocation")
	}
	if _, err := repository.ExportGroupInvite(ctx, &member, entity.ID, true); !errors.Is(err, chat.ErrPermissionDenied) {
		t.Fatal("member revoked invite")
	}
	preview, count, err := repository.PreviewGroup(ctx, invitation.Token, "")
	if err != nil || preview.ID != entity.ID || count != 2 {
		t.Fatalf("private invite preview: %v", err)
	}
	joined, err := repository.JoinGroup(ctx, &outsider, invitation.Token, "")
	if err != nil || len(joined.Participants) != 3 {
		t.Fatalf("join failed: %v", err)
	}
	if _, err := repository.JoinGroup(ctx, &outsider, invitation.Token, ""); err != nil {
		t.Fatal(err)
	}
	uses, err := repository.ExportGroupInvite(ctx, &owner, entity.ID, false)
	if err != nil || uses.Uses != 1 {
		t.Fatal("repeat join incremented invitation use")
	}
	rotated, err := repository.ExportGroupInvite(ctx, &owner, entity.ID, true)
	if err != nil || rotated.Token == invitation.Token || rotated.Uses != 0 {
		t.Fatal("revocation did not issue a new link")
	}
	if _, _, err := repository.PreviewGroup(ctx, invitation.Token, ""); !errors.Is(err, chat.ErrGroupLinkInvalid) {
		t.Fatal("revoked link still works")
	}
	updated, err := repository.UpdateGroup(ctx, &owner, entity.ID, "Public Group", "Updated description", nil, false, chat.GroupOptions{Visibility: "public", PublicUsername: "public_group"})
	if err != nil || updated.PublicUsername == nil || *updated.PublicUsername != "public_group" {
		t.Fatalf("public settings: %v", err)
	}
	if _, _, err := repository.PreviewGroup(ctx, "", "public_group"); err != nil {
		t.Fatal(err)
	}
	allowed, err = access.Authorize(ctx, prefix, nil)
	if err != nil || !allowed.Allowed || !allowed.Public || !allowed.NoStore {
		t.Fatalf("public group avatar unavailable: %+v, %v", allowed, err)
	}
	updated, err = repository.UpdateGroup(ctx, &owner, entity.ID, "Private Group", "Private description", nil, true, chat.GroupOptions{Visibility: "private"})
	if err != nil || updated.PublicUsername != nil || updated.AvatarID != nil || updated.Avatar != nil {
		t.Fatalf("remove avatar/private settings: %+v %v", updated, err)
	}
	if _, _, err := repository.PreviewGroup(ctx, "", "public_group"); !errors.Is(err, chat.ErrGroupLinkInvalid) {
		t.Fatal("old public link remains active")
	}
	allowed, err = access.Authorize(ctx, prefix, nil)
	if err != nil || allowed.Allowed {
		t.Fatal("old group avatar is still public")
	}
}
