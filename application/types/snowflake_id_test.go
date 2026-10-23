package types

import (
	"encoding/json"
	"testing"
)

func TestSnowflakeIDJSONRoundTrip(t *testing.T) {
	want := SnowflakeID(9223372036854775807)
	encoded, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `"9223372036854775807"` {
		t.Fatalf("encoded = %s", encoded)
	}
	for _, data := range []string{string(encoded), "9223372036854775807"} {
		var got SnowflakeID
		if err := json.Unmarshal([]byte(data), &got); err != nil || got != want {
			t.Fatalf("decode %s: got=%d error=%v", data, got, err)
		}
	}
}

func TestSnowflakeIDRejectsInvalidJSONValues(t *testing.T) {
	for _, data := range []string{`"invalid"`, `"9223372036854775808"`, "1.5", "true", "null", "{}"} {
		got := SnowflakeID(42)
		if err := json.Unmarshal([]byte(data), &got); err == nil || got != 42 {
			t.Fatalf("accepted invalid ID %s: got=%d error=%v", data, got, err)
		}
	}
}
