package media

import "strings"

// FileAccessPolicy contains only the business facts needed to decide whether
// an uploaded file may be served. Authentication and persistence stay outside
// the domain.
type FileAccessPolicy struct {
	MediaPublic      bool
	AttachedToPost   bool
	PostPublished    bool
	Audience         string
	ChatMedia        bool
	PrivilegedViewer bool
	OwnerViewer      bool
	ChatParticipant  bool
}

func (p FileAccessPolicy) PubliclyAccessible() bool {
	if !p.MediaPublic || p.ChatMedia {
		return false
	}
	if !p.AttachedToPost {
		return true
	}
	if !p.PostPublished {
		return false
	}
	audience := strings.ToLower(strings.TrimSpace(p.Audience))
	return audience == "" || audience == "public"
}

func (p FileAccessPolicy) Accessible() bool {
	return p.PubliclyAccessible() || p.PrivilegedViewer || p.OwnerViewer || (p.ChatMedia && p.ChatParticipant)
}
