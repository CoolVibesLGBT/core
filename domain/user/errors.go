package user

import "errors"

var ErrFriendshipForbidden = errors.New("friendship action is forbidden")
var ErrFriendshipTransition = errors.New("invalid friendship transition")
