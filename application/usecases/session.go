package usecases

import (
	"context"
	"core/application/ports"
	"core/constants"
	domainuser "core/domain/user"
	"core/models"
	"errors"
)

var ErrSessionUserNotFound = errors.New("session user not found")

type SessionService struct {
	users        ports.SessionRepository
	tokenDecoder ports.UserTokenDecoder
}

func NewSessionService(users ports.SessionRepository, tokenDecoder ports.UserTokenDecoder) *SessionService {
	return &SessionService{users: users, tokenDecoder: tokenDecoder}
}

type ResolvedSession struct {
	User        *models.User
	hasLocation bool
}

func (s *SessionService) ResolveUser(ctx context.Context, token string) (*ResolvedSession, error) {
	publicID, err := s.tokenDecoder.DecodeUserPublicID(token)
	if err != nil {
		return nil, err
	}
	user, err := s.users.GetSessionUserByPublicID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	if user == nil || !domainuser.CanAuthenticate(domainuser.AccountRole(user.Role), user.IsBot) {
		return nil, ErrSessionUserNotFound
	}
	resolved := mapResolvedSession(user)
	if resolved == nil || resolved.User == nil {
		return nil, ErrSessionUserNotFound
	}
	return resolved, nil
}

func mapResolvedSession(user *ports.SessionUser) *ResolvedSession {
	if user == nil {
		return nil
	}
	return &ResolvedSession{
		User: &models.User{
			ID:               user.ID,
			PublicID:         user.PublicID,
			Domain:           models.DomainKind(user.Domain),
			UserName:         user.UserName,
			DisplayName:      user.DisplayName,
			DefaultLanguage:  user.DefaultLanguage,
			PreferencesFlags: user.PreferencesFlags,
			UserRole:         constants.UserRole(user.Role),
			Balance:          user.Balance,
		},
		hasLocation: user.HasLocation,
	}
}

func (s *SessionService) TrackLocation(ctx context.Context, session *ResolvedSession, ip string) error {
	if session == nil || session.User == nil || session.hasLocation {
		return nil
	}
	if err := s.users.UpdateLocation(ctx, session.User.ID, ip); err != nil {
		return err
	}
	session.hasLocation = true
	return nil
}
