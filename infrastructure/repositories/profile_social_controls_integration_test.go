package repositories

import (
	"context"
	"core/application/ports"
	"core/application/usecases"
	domainuser "core/domain/user"
	"core/helpers"
	"core/models"
	"core/models/notifications"
	"errors"
	"testing"
)

func TestProfileFriendshipStatusAndDirectPhotoSharingIntegration(t *testing.T) {
	db := socialGraphIntegrationDB(t)
	owner, viewer := privatePhotoTestUsers(t, db)
	ctx := context.Background()
	node, _ := helpers.NewDefaultNode()
	friends := NewFriendshipRepository(db, node)
	status, err := friends.FindFriendship(ctx, owner.ID, viewer.PublicID)
	if err != nil || status != nil {
		t.Fatalf("unexpected initial friendship: %+v %v", status, err)
	}
	request, err := friends.RequestFriendship(ctx, viewer.ID, owner.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	status, err = friends.FindFriendship(ctx, owner.ID, viewer.PublicID)
	if err != nil || status.RequesterPublicID != viewer.PublicID || status.RecipientPublicID != owner.PublicID || status.PublicID != request.PublicID {
		t.Fatalf("request direction/public identity lost: %+v %v", status, err)
	}
	if _, err = friends.ChangeFriendship(ctx, owner.ID, request.PublicID, domainuser.FriendshipAccepted); err != nil {
		t.Fatal(err)
	}
	status, err = friends.FindFriendship(ctx, viewer.ID, owner.PublicID)
	if err != nil || status.Status != domainuser.FriendshipAccepted {
		t.Fatal("accepted status not persisted")
	}
	photos := privatePhotoTestRepository(t, db)
	service := usecases.NewPrivatePhotoService(photos, nil, nil)
	principal := ports.PrivatePhotoPrincipal{ID: owner.ID, PublicID: owner.PublicID}
	access, err := service.AccessForViewer(ctx, principal, viewer.PublicID)
	if err != nil || access.Status != "none" {
		t.Fatal("unexpected initial photo grant")
	}
	for i := 0; i < 2; i++ {
		access, err = service.GrantAccess(ctx, principal, viewer.PublicID)
		if err != nil || access.Status != "approved" {
			t.Fatalf("direct share failed: %+v %v", access, err)
		}
	}
	var requests, responses int64
	db.Model(&notifications.Notification{}).Where("type = ?", notifications.NotificationTypePrivatePhotoAccessRequest).Count(&requests)
	db.Model(&notifications.Notification{}).Where("type = ? AND user_id = ?", notifications.NotificationTypePrivatePhotoAccessResponse, viewer.ID).Count(&responses)
	if requests != 0 || responses != 1 {
		t.Fatalf("direct share generated false/duplicate requests: requests=%d responses=%d", requests, responses)
	}
	if allowed, err := photos.HasApprovedPrivatePhotoAccess(ctx, owner.ID, viewer.ID); err != nil || !allowed {
		t.Fatal("direct share does not authorize viewer")
	}
	if _, err := service.RevokeAccess(ctx, principal, viewer.PublicID); err != nil {
		t.Fatal(err)
	}
	if allowed, err := photos.HasApprovedPrivatePhotoAccess(ctx, owner.ID, viewer.ID); err != nil || allowed {
		t.Fatal("hide did not revoke viewer")
	}
	if _, err := service.GrantAccess(ctx, principal, viewer.PublicID); err != nil {
		t.Fatal("owner cannot explicitly share again", err)
	}
	block, _ := domainuser.NewSetInteractionState(domainuser.InteractionBlock, true)
	if _, err := NewEngagementRepository(db).ApplyReciprocalUserInteraction(ctx, viewer.ID, owner.ID, block); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GrantAccess(ctx, principal, viewer.PublicID); !errors.Is(err, usecases.ErrPrivatePhotoForbidden) {
		t.Fatalf("blocked sharing accepted: %v", err)
	}
	if _, err := friends.FindFriendship(ctx, owner.ID, viewer.PublicID); !errors.Is(err, ports.ErrNotFound) {
		t.Fatalf("blocked relation exposed: %v", err)
	}
	var pending int64
	if err := db.Model(&models.PrivatePhotoAccessRequest{}).Where("owner_id = ? AND viewer_id = ? AND status = 'pending'", owner.ID, viewer.ID).Count(&pending).Error; err != nil || pending != 0 {
		t.Fatal("direct share left fake pending request")
	}
}
