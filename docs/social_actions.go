package docs

import (
	"core/constants"
	domainuser "core/domain/user"
)

func socialActionDocuments() []ActionDoc {
	docs := []ActionDoc{
		{Action: constants.CMD_USER_FRIEND_REQUEST, Tag: "Friends", Summary: "Request friendship", Description: "Creates a pending friendship request. Responses identify the original sender as requester_id and the recipient as recipient_id; user is the other participant relative to the session. Replays and crossed requests keep the original direction and remain pending; recipient approval is always required.", Auth: AuthRequired, Params: []Param{intParam("user_id", "Recipient public ID.", true, nil)}},
		{Action: constants.CMD_USER_FRIEND_RESPOND, Tag: "Friends", Summary: "Respond to friendship request", Description: "Only the recipient may accept or deny. A decision for an old request cannot approve a later request.", Auth: AuthRequired, Params: []Param{intParam("request_id", "Current request public ID.", true, nil), enumParam("decision", "Decision.", true, string(domainuser.FriendshipAccepted), string(domainuser.FriendshipDenied))}},
		{Action: constants.CMD_USER_FRIEND_CANCEL, Tag: "Friends", Summary: "Cancel outgoing friend request", Auth: AuthRequired, Params: []Param{intParam("request_id", "Outgoing request public ID.", true, nil)}},
		{Action: constants.CMD_USER_FRIEND_REMOVE, Tag: "Friends", Summary: "Remove friendship", Description: "Either participant can remove an accepted friendship. Friends-only profile access is revoked immediately.", Auth: AuthRequired, Params: []Param{intParam("request_id", "Friendship public ID.", true, nil)}},
		{Action: constants.CMD_USER_FRIEND_FETCH, Tag: "Friends", Summary: "List my friends", Description: "requester_id and recipient_id retain the request direction; user is the other participant relative to the authenticated session.", Auth: AuthRequired, Params: []Param{stringParam("cursor", "Opaque continuation cursor.", false), intParam("limit", "Maximum item count.", false, 10)}},
		{Action: constants.CMD_USER_FRIEND_REQUESTS, Tag: "Friends", Summary: "List pending friend requests", Description: "Incoming requests have the authenticated user as recipient; outgoing requests have the authenticated user as requester. user always represents the other participant.", Auth: AuthRequired, Params: []Param{enumParam("scope", "Request direction. Defaults to incoming.", false, string(domainuser.FriendshipScopeIncoming), string(domainuser.FriendshipScopeOutgoing)), stringParam("cursor", "Opaque continuation cursor.", false), intParam("limit", "Maximum item count.", false, 10)}},
		matchCursorDoc(constants.CMD_MATCH_FETCH, "List matches", "Alias of match.fetch.matched."),
		{Action: constants.CMD_MATCH_UPDATE, Tag: "Matches", Summary: "Update reaction", Description: "Alias of match.create. Applying the same reaction again is idempotent.", Auth: AuthRequired, Params: []Param{intParam("public_id", "Target public ID.", true, nil), enumParam("reaction", "Reaction.", true, "like", "dislike")}},
		{Action: constants.CMD_MATCH_DELETE, Tag: "Matches", Summary: "Remove match", Description: "Removes both directions of the match and likes atomically. Repeating removal is safe.", Auth: AuthRequired, Params: []Param{intParam("public_id", "Target public ID.", true, nil)}},
	}
	for _, action := range []string{constants.CMD_USER_FETCH_FOLLOWERS, constants.CMD_USER_FETCH_FOLLOWINGS} {
		docs = append(docs, ActionDoc{Action: action, Tag: "Users", Summary: "List followers or followings", Description: "Defaults to the authenticated user. Applies profile visibility and participant blocking rules.", Auth: AuthRequired, Params: []Param{intParam("user_id", "Optional profile owner public ID.", false, nil), stringParam("cursor", "Opaque continuation cursor.", false), intParam("limit", "Maximum item count.", false, 10)}})
	}
	return docs
}
