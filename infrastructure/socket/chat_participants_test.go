package socket

import "testing"

func TestChatParticipantChangesUpdateEveryAuthenticatedConnection(t *testing.T) {
	clearConnections()
	defer clearConnections()
	first, second, left, anonymous := newFakeSocketConn("one"), newFakeSocketConn("two"), newFakeSocketConn("left"), newFakeSocketConn("anonymous")
	for _, conn := range []*fakeSocketConn{first, second, left, anonymous} {
		rememberConnection(conn)
		conn.Join("group")
	}
	rememberPublicID(first.ID(), 10)
	rememberPublicID(second.ID(), 10)
	rememberPublicID(left.ID(), 20)
	applyChatParticipantRooms("group", map[int64]bool{10: true})
	if !first.hasRoom("group") || !second.hasRoom("group") || left.hasRoom("group") || anonymous.hasRoom("group") {
		t.Fatal("membership was not applied to every connection")
	}
	applyChatParticipantRooms("group", map[int64]bool{20: true})
	if first.hasRoom("group") || second.hasRoom("group") || !left.hasRoom("group") {
		t.Fatal("rejoin/removal did not refresh live rooms")
	}
}
