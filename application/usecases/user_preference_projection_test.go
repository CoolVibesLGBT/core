package usecases

import (
	"core/models"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestAuthUserPreferenceChoicesSurviveProjection(t *testing.T) {
	var flags big.Int
	selected := []int{0, 63, 511, 784, 849}
	for _, bit := range selected {
		flags.SetBit(&flags, bit, 1)
	}
	user := &models.User{PreferencesFlags: hex.EncodeToString(flags.Bytes())}
	owner := AuthUserResponse(user)
	if !reflect.DeepEqual(owner.SelectedPreferenceBits, selected) {
		t.Fatalf("selected preference bits = %v, want %v", owner.SelectedPreferenceBits, selected)
	}
	encoded, err := json.Marshal(owner)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), `"preferences_flags"`) {
		t.Fatal("owner projection exposed the raw stored bitset")
	}
	public, err := json.Marshal(UserProfileResponse(user))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(public), `"selected_preference_bits"`) {
		t.Fatal("owner's selected preference bits leaked into a public profile")
	}

	user.PreferencesFlags = ""
	encoded, err = json.Marshal(AuthUserResponse(user))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"selected_preference_bits":[]`) {
		t.Fatal("empty selection must explicitly clear the client's previous choices")
	}
}
