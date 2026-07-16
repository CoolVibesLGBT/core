package user

import (
	"encoding/hex"
	"errors"
	"math/big"
)

var ErrNegativePreferenceBit = errors.New("bitIndex must be non-negative")

type PreferenceFlags string

func (f PreferenceFlags) Set(bitIndex int) (PreferenceFlags, error) {
	flags, err := f.bigInt()
	if err != nil {
		return "", err
	}
	if bitIndex < 0 {
		return "", ErrNegativePreferenceBit
	}

	flags.SetBit(flags, bitIndex, 1)
	return PreferenceFlags(hex.EncodeToString(flags.Bytes())), nil
}

func (f PreferenceFlags) Unset(bitIndex int) (PreferenceFlags, error) {
	flags, err := f.bigInt()
	if err != nil {
		return "", err
	}
	if bitIndex < 0 {
		return "", ErrNegativePreferenceBit
	}

	flags.SetBit(flags, bitIndex, 0)
	return PreferenceFlags(hex.EncodeToString(flags.Bytes())), nil
}

func (f PreferenceFlags) IsSet(bitIndex int) (bool, error) {
	flags, err := f.bigInt()
	if err != nil {
		return false, err
	}
	if bitIndex < 0 {
		return false, ErrNegativePreferenceBit
	}

	return flags.Bit(bitIndex) == 1, nil
}

func (f PreferenceFlags) bigInt() (*big.Int, error) {
	flags := big.NewInt(0)
	if f == "" {
		return flags, nil
	}

	bytes, err := hex.DecodeString(string(f))
	if err != nil {
		return nil, err
	}
	flags.SetBytes(bytes)
	return flags, nil
}
