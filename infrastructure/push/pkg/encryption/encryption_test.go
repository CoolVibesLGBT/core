package encryption

import "testing"

func TestPrepareSaltProducesSixteenBytes(t *testing.T) {
	service := &Service{}
	salt, err := service.prepareSalt()
	if err != nil {
		t.Fatalf("prepareSalt() error = %v", err)
	}

	if len(salt) != 16 {
		t.Fatalf("salt length = %d; want 16", len(salt))
	}
}
