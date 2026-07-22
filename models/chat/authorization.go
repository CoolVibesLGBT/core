package chat

import (
	"core/constants"

	"github.com/google/uuid"
)

// CanModerate reports whether an active participant may perform a
// conversation-wide mutation. Active membership is deliberately checked by
// the repository before this aggregate policy is evaluated.
func (c Chat) CanModerate(actorID uuid.UUID, userRole constants.UserRole, participantRole ParticipantRole) bool {
	if actorID == uuid.Nil {
		return false
	}
	if actorID == c.CreatorID {
		return true
	}
	if userRole == constants.UserRoleAdmin || userRole == constants.UserRoleSuperAdmin {
		return true
	}
	return participantRole == ParticipantRoleAdmin || participantRole == ParticipantRoleOwner
}

// CanDeleteMessage permits the message author or a chat moderator to remove a
// message for everyone. It intentionally does not grant this power to an
// ordinary participant.
func (c Chat) CanDeleteMessage(actorID, authorID uuid.UUID, userRole constants.UserRole, participantRole ParticipantRole) bool {
	return actorID != uuid.Nil && (actorID == authorID || c.CanModerate(actorID, userRole, participantRole))
}
