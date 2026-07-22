package chat

import (
	"core/constants"
	"testing"

	"github.com/google/uuid"
)

func TestChatMutationAuthorizationPolicy(t *testing.T) {
	creatorID := uuid.New()
	authorID := uuid.New()
	memberID := uuid.New()
	entity := Chat{CreatorID: creatorID}

	tests := []struct {
		name            string
		actorID         uuid.UUID
		userRole        constants.UserRole
		participantRole ParticipantRole
		moderate        bool
		deleteMessage   bool
	}{
		{name: "creator", actorID: creatorID, userRole: constants.UserRoleUser, participantRole: ParticipantRoleMember, moderate: true, deleteMessage: true},
		{name: "message author", actorID: authorID, userRole: constants.UserRoleUser, participantRole: ParticipantRoleMember, deleteMessage: true},
		{name: "chat admin", actorID: memberID, userRole: constants.UserRoleUser, participantRole: ParticipantRoleAdmin, moderate: true, deleteMessage: true},
		{name: "chat owner", actorID: memberID, userRole: constants.UserRoleUser, participantRole: ParticipantRoleOwner, moderate: true, deleteMessage: true},
		{name: "system admin", actorID: memberID, userRole: constants.UserRoleAdmin, participantRole: ParticipantRoleMember, moderate: true, deleteMessage: true},
		{name: "system super admin", actorID: memberID, userRole: constants.UserRoleSuperAdmin, participantRole: ParticipantRoleMember, moderate: true, deleteMessage: true},
		{name: "ordinary member", actorID: memberID, userRole: constants.UserRoleUser, participantRole: ParticipantRoleMember},
		{name: "system moderator is not chat admin", actorID: memberID, userRole: constants.UserRoleModerator, participantRole: ParticipantRoleMember},
		{name: "nil actor", actorID: uuid.Nil, userRole: constants.UserRoleAdmin, participantRole: ParticipantRoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := entity.CanModerate(tt.actorID, tt.userRole, tt.participantRole); got != tt.moderate {
				t.Fatalf("CanModerate() = %v, want %v", got, tt.moderate)
			}
			if got := entity.CanDeleteMessage(tt.actorID, authorID, tt.userRole, tt.participantRole); got != tt.deleteMessage {
				t.Fatalf("CanDeleteMessage() = %v, want %v", got, tt.deleteMessage)
			}
		})
	}
}
