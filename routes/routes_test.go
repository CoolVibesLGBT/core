package routes

import "testing"

func TestRequestBodyHasNoApplicationLimit(t *testing.T) {
	want := int(^uint(0)>>1) - 1
	if got := requestBodyLimit(); got != want {
		t.Fatalf("body limit = %d, want platform maximum %d", got, want)
	}
}
