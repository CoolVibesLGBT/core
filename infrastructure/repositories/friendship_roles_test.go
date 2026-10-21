package repositories

import (
	"context"
	"core/application/ports"
	domainuser "core/domain/user"
	"core/helpers"
	"core/models"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func TestFriendshipRolesStayDirectionalAcrossRetriesAndNewRequestsIntegration(t *testing.T) {
	for _, senderIsLower := range []bool{true, false} {
		name := "sender-has-higher-uuid"
		if senderIsLower {
			name = "sender-has-lower-uuid"
		}
		t.Run(name, func(t *testing.T) {
			db := socialGraphIntegrationDB(t)
			lower := uuid.MustParse("00000000-0000-4000-8000-000000000001")
			higher := uuid.MustParse("00000000-0000-4000-8000-000000000002")
			sender := models.User{ID: higher, PublicID: 10, UserName: "sender"}
			recipient := models.User{ID: lower, PublicID: 20, UserName: "recipient"}
			if senderIsLower {
				sender.ID, recipient.ID = lower, higher
			}
			if err := db.Omit(clause.Associations).Create(&[]models.User{sender, recipient}).Error; err != nil {
				t.Fatal(err)
			}
			node, err := helpers.NewDefaultNode()
			if err != nil {
				t.Fatal(err)
			}
			repo := NewFriendshipRepository(db, node)
			ctx := context.Background()
			assertRoles := func(record ports.FriendshipRecord, requester, target, other int64) {
				t.Helper()
				if record.RequesterPublicID != requester || record.RecipientPublicID != target || int64(record.OtherUser.PublicID) != other {
					t.Fatalf("requester/recipient/other=%d/%d/%d want %d/%d/%d", record.RequesterPublicID, record.RecipientPublicID, record.OtherUser.PublicID, requester, target, other)
				}
			}
			request, err := repo.RequestFriendship(ctx, sender.ID, recipient.PublicID)
			if err != nil {
				t.Fatal(err)
			}
			assertRoles(request, 10, 20, 20)
			crossed, err := repo.RequestFriendship(ctx, recipient.ID, sender.PublicID)
			if err != nil || crossed.PublicID != request.PublicID {
				t.Fatalf("crossed request=%+v err=%v", crossed, err)
			}
			assertRoles(crossed, 10, 20, 10)
			for _, list := range []struct {
				actor models.User
				scope domainuser.FriendshipScope
				other int64
			}{
				{sender, domainuser.FriendshipScopeOutgoing, 20}, {recipient, domainuser.FriendshipScopeIncoming, 10},
			} {
				rows, _, err := repo.ListFriendships(ctx, list.actor.ID, list.scope, nil, 10)
				if err != nil || len(rows) != 1 {
					t.Fatalf("%s=%+v err=%v", list.scope, rows, err)
				}
				assertRoles(rows[0], 10, 20, list.other)
			}
			if _, err := repo.ChangeFriendship(ctx, sender.ID, request.PublicID, domainuser.FriendshipAccepted); !errors.Is(err, domainuser.ErrFriendshipForbidden) {
				t.Fatalf("requester accepted own request: %v", err)
			}
			cancelled, err := repo.ChangeFriendship(ctx, sender.ID, request.PublicID, domainuser.FriendshipCancelled)
			if err != nil {
				t.Fatal(err)
			}
			assertRoles(cancelled, 10, 20, 20)
			// A new request can reverse roles even though the canonical pair is unchanged.
			reversed, err := repo.RequestFriendship(ctx, recipient.ID, sender.PublicID)
			if err != nil || reversed.PublicID == request.PublicID {
				t.Fatalf("new reverse request=%+v err=%v", reversed, err)
			}
			assertRoles(reversed, 20, 10, 10)
			if _, err := repo.ChangeFriendship(ctx, recipient.ID, request.PublicID, domainuser.FriendshipAccepted); !errors.Is(err, ports.ErrNotFound) {
				t.Fatalf("stale request ID remained active: %v", err)
			}
			if _, err := repo.ChangeFriendship(ctx, recipient.ID, reversed.PublicID, domainuser.FriendshipAccepted); !errors.Is(err, domainuser.ErrFriendshipForbidden) {
				t.Fatalf("new requester accepted own request: %v", err)
			}
			accepted, err := repo.ChangeFriendship(ctx, sender.ID, reversed.PublicID, domainuser.FriendshipAccepted)
			if err != nil || accepted.Status != domainuser.FriendshipAccepted {
				t.Fatalf("recipient acceptance=%+v err=%v", accepted, err)
			}
			assertRoles(accepted, 20, 10, 20)
			for _, actor := range []models.User{sender, recipient} {
				rows, _, err := repo.ListFriendships(ctx, actor.ID, domainuser.FriendshipScopeFriends, nil, 10)
				if err != nil || len(rows) != 1 {
					t.Fatalf("friends=%+v err=%v", rows, err)
				}
				other := sender.PublicID
				if actor.ID == sender.ID {
					other = recipient.PublicID
				}
				assertRoles(rows[0], 20, 10, other)
			}
			var stored models.Friendship
			if err := db.Where("public_id = ?", reversed.PublicID).Take(&stored).Error; err != nil {
				t.Fatal(err)
			}
			if stored.FirstUserID != lower || stored.SecondUserID != higher {
				t.Fatal("request reversal changed the canonical storage pair")
			}
			if _, err := repo.friendshipRecords(ctx, []models.Friendship{stored}, uuid.New()); !errors.Is(err, domainuser.ErrFriendshipForbidden) {
				t.Fatalf("outsider received participant view: %v", err)
			}
		})
	}
}
