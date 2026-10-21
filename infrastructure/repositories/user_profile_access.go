package repositories

import (
	"context"
	"core/application/ports"
	"core/application/types"
	"core/constants"
	"core/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Visibility is evaluated at read time through composable GORM scopes.
func profileViewerScope(query *gorm.DB, viewer uuid.UUID) *gorm.DB {
	query = query.Where("users.deleted_at IS NULL AND users.is_bot = FALSE").
		Where("users.user_role NOT IN ?", []constants.UserRole{constants.UserRoleBanned, constants.UserRoleDeleted, constants.UserRolePending}).
		Where("users.domain = (?)", userDomainSubquery(query, viewer))
	return userAudienceScope(userBlockScope(query, viewer, "users"), viewer, "users")
}

func (r *UserRepository) CanViewUserProfile(ctx context.Context, owner, viewer uuid.UUID) (bool, error) {
	var count int64
	err := profileViewerScope(r.db.WithContext(ctx).Model(&models.User{}).Table("users"), viewer).Where("users.id = ?", owner).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) FetchUserProfileForViewer(ctx context.Context, username string, viewer uuid.UUID) (*types.PublicUserProfile, error) {
	var row publicUserProjectionRow
	query := profileViewerScope(r.userProfileProjectionQuery(ctx, username), viewer)
	if err := query.Take(&row).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	result := row.profileView()
	return &result, nil
}
