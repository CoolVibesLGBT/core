package push

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
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

func parseVAPIDPrivateKeyString(key string) (*ecdsa.PrivateKey, error) {
	key = strings.TrimSpace(key)

	raw, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		// fallback
		raw, err = base64.URLEncoding.DecodeString(key)
		if err != nil {
			return nil, errors.New("failed to decode VAPID private key")
		}
	}

	if len(raw) != 32 {
		return nil, errors.New("VAPID private key must be 32 bytes")
	}

	d := new(big.Int).SetBytes(raw)
	if d.Sign() == 0 {
		return nil, errors.New("VAPID private key is zero")
	}

	curve := elliptic.P256()
	priv := &ecdsa.PrivateKey{
		D: d,
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
		},
	}

	// Public key’i artık güvenli şekilde derive et
	ecdhCurve := ecdh.P256()
	privECDH, err := ecdhCurve.NewPrivateKey(raw)
	if err != nil {
		return nil, err
	}

	pub := privECDH.PublicKey().Bytes()
	if len(pub) != 65 {
		return nil, errors.New("invalid public key length from ECDH")
	}

	// x ve y’yi ayır
	priv.PublicKey.X = new(big.Int).SetBytes(pub[1:33])
	priv.PublicKey.Y = new(big.Int).SetBytes(pub[33:])

	return priv, nil
}

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

	fmt.Println("CODER22 CODER CODER23434222")

	privKey, err := parseVAPIDPrivateKeyString(vapidPrivateKey)
	if err != nil {
		return "", err
	}

	fmt.Println("CODER CODER CODER23434222")

	jwtString, err := token.SignedString(privKey)
	if err != nil {
		return "", err
	}
	fmt.Println("CODER CODER CODER2222")

	decodedPub, err := decodeVapidKey(vapidPublicKey)
	if err != nil {
		return "", err
	}
	fmt.Println("CODER CODER CODER333")

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
