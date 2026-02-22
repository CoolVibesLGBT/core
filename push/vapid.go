package push

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"math/big"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const p256KeySize = 32

// GenerateVAPIDKeys creates a new ES256 VAPID key pair
func GenerateVAPIDKeys() (privateKeyB64 string, publicKeyB64 string, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	// Private key (32 bytes padded)
	dBytes := leftPad(priv.D.Bytes(), p256KeySize)
	privateKeyB64 = base64.RawURLEncoding.EncodeToString(dBytes)

	// Public key (uncompressed 65 bytes)
	xBytes := leftPad(priv.X.Bytes(), p256KeySize)
	yBytes := leftPad(priv.Y.Bytes(), p256KeySize)

	public := make([]byte, 1+2*p256KeySize)
	public[0] = 0x04
	copy(public[1:33], xBytes)
	copy(public[33:], yBytes)

	publicKeyB64 = base64.RawURLEncoding.EncodeToString(public)

	return
}

func leftPad(b []byte, size int) []byte {
	if len(b) >= size {
		return b
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out
}

func parseVAPIDPrivateKey(raw []byte) (*ecdsa.PrivateKey, error) {
	if len(raw) != 32 {
		return nil, errors.New("invalid vapid private key length")
	}

	curve := elliptic.P256()

	d := new(big.Int).SetBytes(raw)

	if d.Sign() <= 0 || d.Cmp(curve.Params().N) >= 0 {
		return nil, errors.New("invalid vapid private key value")
	}

	// Construct private key
	priv := &ecdsa.PrivateKey{
		D: d,
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
		},
	}

	// Derive public key using Public() (no ScalarBaseMult call here)
	pub := priv.Public().(*ecdsa.PublicKey)
	priv.PublicKey = *pub

	return priv, nil
}

// getVAPIDAuthorizationHeader builds Authorization header for Web Push
func getVAPIDAuthorizationHeader(
	endpoint string,
	subscriber string,
	vapidPublicKey string,
	vapidPrivateKey string,
	expiration time.Time,
) (string, error) {

	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	if !strings.HasPrefix(subscriber, "https:") && !strings.HasPrefix(subscriber, "mailto:") {
		subscriber = "mailto:" + subscriber
	}

	claims := jwt.MapClaims{
		"aud": u.Scheme + "://" + u.Host,
		"exp": expiration.Unix(),
		"sub": subscriber,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	decodedPriv, err := decodeVapidKey(vapidPrivateKey)
	if err != nil {
		return "", err
	}

	privKey, err := parseVAPIDPrivateKey(decodedPriv)
	if err != nil {
		return "", err
	}

	jwtString, err := token.SignedString(privKey)
	if err != nil {
		return "", err
	}

	// Public key must already be raw 65-byte uncompressed
	decodedPub, err := decodeVapidKey(vapidPublicKey)
	if err != nil {
		return "", err
	}

	return "vapid t=" + jwtString +
		", k=" + base64.RawURLEncoding.EncodeToString(decodedPub), nil
}

func decodeVapidKey(key string) ([]byte, error) {
	key = strings.TrimSpace(key)

	if b, err := base64.RawURLEncoding.DecodeString(key); err == nil {
		return b, nil
	}

	return base64.URLEncoding.DecodeString(key)
}
