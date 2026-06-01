package post

import "testing"

func TestKindValidation(t *testing.T) {
	if _, ok := ParseKind("news"); !ok {
		t.Fatal("news should be a valid post kind")
	}

	if Kind("unknown").IsValid() {
		t.Fatal("unknown should not be a valid post kind")
	}
}
