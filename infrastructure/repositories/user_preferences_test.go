package repositories

import (
	"core/models"
	"encoding/hex"
	"math/big"
	"testing"
)

func TestUpdatePreferenceFlagsUsesStoredItemBitAndSingleChoiceInvariant(t *testing.T) {
	category := models.PreferenceCategory{
		AllowMultiple: false,
		Items: []models.PreferenceItem{
			{BitIndex: 3},
			{BitIndex: 9},
		},
	}
	var initial big.Int
	initial.SetBit(&initial, 3, 1)

	encoded, err := updatePreferenceFlags(hex.EncodeToString(initial.Bytes()), category, category.Items[1], true)
	if err != nil {
		t.Fatalf("updatePreferenceFlags() error = %v", err)
	}
	bytes, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString() error = %v", err)
	}
	var flags big.Int
	flags.SetBytes(bytes)
	if flags.Bit(3) != 0 || flags.Bit(9) != 1 {
		t.Fatalf("single-choice bits = bit3:%d bit9:%d; want 0,1", flags.Bit(3), flags.Bit(9))
	}
}

func TestUpdatePreferenceFlagsRejectsInvalidStoredBit(t *testing.T) {
	_, err := updatePreferenceFlags("", models.PreferenceCategory{AllowMultiple: true}, models.PreferenceItem{BitIndex: -1}, true)
	if err == nil {
		t.Fatal("negative stored bit index was accepted")
	}
}
