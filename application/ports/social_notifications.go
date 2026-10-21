package ports

// TransactionalSocialNotifications distinguishes adapters that persist social
// notifications with the mutation, so legacy adapters can keep their fallback.
type TransactionalSocialNotifications interface {
	SocialNotificationsTransactional() bool
}
