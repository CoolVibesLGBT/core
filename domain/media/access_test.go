package media

import "testing"

func TestFileAccessPolicy(t *testing.T) {
	tests := []struct {
		name       string
		policy     FileAccessPolicy
		public     bool
		accessible bool
	}{
		{name: "standalone public", policy: FileAccessPolicy{MediaPublic: true}, public: true, accessible: true},
		{name: "published public post", policy: FileAccessPolicy{MediaPublic: true, AttachedToPost: true, PostPublished: true, Audience: "public"}, public: true, accessible: true},
		{name: "private audience", policy: FileAccessPolicy{MediaPublic: true, AttachedToPost: true, PostPublished: true, Audience: "private"}},
		{name: "hidden post", policy: FileAccessPolicy{MediaPublic: true, AttachedToPost: true, Audience: "public"}},
		{name: "owner", policy: FileAccessPolicy{OwnerViewer: true}, accessible: true},
		{name: "moderator", policy: FileAccessPolicy{PrivilegedViewer: true}, accessible: true},
		{name: "approved private photo grant", policy: FileAccessPolicy{PrivatePhotoGrant: true}, accessible: true},
		{name: "chat participant", policy: FileAccessPolicy{MediaPublic: true, ChatMedia: true, ChatParticipant: true}, accessible: true},
		{name: "chat outsider", policy: FileAccessPolicy{MediaPublic: true, ChatMedia: true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.policy.PubliclyAccessible(); got != test.public {
				t.Fatalf("PubliclyAccessible() = %v, want %v", got, test.public)
			}
			if got := test.policy.Accessible(); got != test.accessible {
				t.Fatalf("Accessible() = %v, want %v", got, test.accessible)
			}
		})
	}
}
