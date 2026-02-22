package push

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/hkdf"
)

const MaxRecordSize uint32 = 4096

var ErrMaxPadExceeded = errors.New("payload exceeded maximum length")

var saltFunc = func() ([]byte, error) {
	salt := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, salt)
	return salt, err
}

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Options struct {
	HTTPClient      HTTPClient
	RecordSize      uint32
	Subscriber      string
	Topic           string
	TTL             int
	Urgency         Urgency
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VapidExpiration time.Time
}

type Keys struct {
	Auth   string `json:"auth"`
	P256dh string `json:"p256dh"`
}

type Subscription struct {
	Endpoint string `json:"endpoint"`
	Keys     Keys   `json:"keys"`
}

func SendNotification(message []byte, s *Subscription, options *Options) (*http.Response, error) {
	return SendNotificationWithContext(context.Background(), message, s, options)
}

func SendNotificationWithContext(ctx context.Context, message []byte, s *Subscription, options *Options) (*http.Response, error) {
	authSecret, err := decodeSubscriptionKey(s.Keys.Auth)
	if err != nil {
		return nil, err
	}

	dh, err := decodeSubscriptionKey(s.Keys.P256dh)
	if err != nil {
		return nil, err
	}

	salt, err := saltFunc()
	if err != nil {
		return nil, err
	}

	// ---- Modern ECDH ----
	curve := ecdh.P256()

	localPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	localPub := localPriv.PublicKey()
	localPublicKey := localPub.Bytes()

	remotePub, err := curve.NewPublicKey(dh)
	if err != nil {
		return nil, errors.New("invalid public key: not on curve")
	}

	sharedSecret, err := localPriv.ECDH(remotePub)
	if err != nil {
		return nil, errors.New("ecdh shared secret derivation failed")
	}

	hash := sha256.New

	prkInfo := bytes.NewBuffer([]byte("WebPush: info\x00"))
	prkInfo.Write(dh)
	prkInfo.Write(localPublicKey)

	prkHKDF := hkdf.New(hash, sharedSecret, authSecret, prkInfo.Bytes())
	ikm, err := getHKDFKey(prkHKDF, 32)
	if err != nil {
		return nil, err
	}

	contentHKDF := hkdf.New(hash, ikm, salt, []byte("Content-Encoding: aes128gcm\x00"))
	contentEncryptionKey, err := getHKDFKey(contentHKDF, 16)
	if err != nil {
		return nil, err
	}

	nonceHKDF := hkdf.New(hash, ikm, salt, []byte("Content-Encoding: nonce\x00"))
	nonce, err := getHKDFKey(nonceHKDF, 12)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(contentEncryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	recordSize := options.RecordSize
	if recordSize == 0 {
		recordSize = MaxRecordSize
	}

	recordLength := int(recordSize) - 16

	recordBuf := bytes.NewBuffer(salt)

	rs := make([]byte, 4)
	binary.BigEndian.PutUint32(rs, recordSize)
	recordBuf.Write(rs)
	recordBuf.Write([]byte{byte(len(localPublicKey))})
	recordBuf.Write(localPublicKey)

	messageCopy := make([]byte, len(message))
	copy(messageCopy, message)

	dataBuf := bytes.NewBuffer(messageCopy)
	dataBuf.Write([]byte("\x02"))

	if err := pad(dataBuf, recordLength-recordBuf.Len()); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, dataBuf.Bytes(), nil)
	recordBuf.Write(ciphertext)

	req, err := http.NewRequest("POST", s.Endpoint, recordBuf)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", strconv.Itoa(options.TTL))

	if options.Topic != "" {
		req.Header.Set("Topic", options.Topic)
	}

	if isValidUrgency(options.Urgency) {
		req.Header.Set("Urgency", string(options.Urgency))
	}

	expiration := options.VapidExpiration
	if expiration.IsZero() {
		expiration = time.Now().Add(12 * time.Hour)
	}

	vapidHeader, err := getVAPIDAuthorizationHeader(
		s.Endpoint,
		options.Subscriber,
		options.VAPIDPublicKey,
		options.VAPIDPrivateKey,
		expiration,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", vapidHeader)

	client := options.HTTPClient
	if client == nil {
		client = &http.Client{}
	}

	return client.Do(req)
}

func decodeSubscriptionKey(key string) ([]byte, error) {
	buf := bytes.NewBufferString(key)
	if rem := len(key) % 4; rem != 0 {
		buf.WriteString(strings.Repeat("=", 4-rem))
	}

	b, err := base64.StdEncoding.DecodeString(buf.String())
	if err == nil {
		return b, nil
	}

	return base64.URLEncoding.DecodeString(buf.String())
}

func getHKDFKey(r io.Reader, length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := io.ReadFull(r, key)
	return key, err
}

func pad(payload *bytes.Buffer, maxPadLen int) error {
	if payload.Len() > maxPadLen {
		return ErrMaxPadExceeded
	}
	padding := make([]byte, maxPadLen-payload.Len())
	payload.Write(padding)
	return nil
}
