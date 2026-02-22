package encryption

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"core/push/pkg/utils"
)

var (
	// Generate cek_info according to RFC8291 3.4:
	cekInfo = []byte("Content-Encoding: aes128gcm\x00")

	// Generate nonce_info according to RFC8291 3.4:
	nonceInfo = []byte("Content-Encoding: nonce\x00")
)

// prepareInputData return auth secret, user's public key, error.
func (s *Service) prepareInputData(auth, p256dh string) ([]byte, []byte, error) {
	authSecret, err := utils.ParseBase64Key(auth)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode auth secret: %w", err)
	}

	uaPublicKey, err := utils.ParseBase64Key(p256dh)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode user public key: %w", err)
	}

	return authSecret, uaPublicKey, nil
}

// ecdhExchange return ECDH exchange return shared secret and error.
func (s *Service) ecdhExchange(uaPublicKey []byte) ([]byte, error) {
	publicKey, err := ecdh.P256().NewPublicKey(uaPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to pearse public key: %w", err)
	}

	sharedSecret, err := s.privateKey.ECDH(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate shared secret: %w", err)
	}

	return sharedSecret, nil
}

// prepareIKM return IKM and error.
func (s *Service) prepareIKM(sharedSecret, authSecret, uaPublicKey []byte) ([]byte, error) {
	// Generate key_info according to RFC8291 3.4
	keyInfo := make([]byte, 144)
	copy(keyInfo[:14], "WebPush: info\x00")
	copy(keyInfo[14:79], uaPublicKey)
	copy(keyInfo[79:], s.publicKey)

	// Generate PRK_key and IKM according to RFC8291 3.4:
	// First 3 arguments to hkdf.New() response for PRK
	//
	// PRK = HMAC-SHA-256(auth_secret, ecdh_secret)
	// IKM = HKDF-Expand(PRK, key_info, 32)
	ikm, err := utils.HkdfExtractAndExpand(32, sharedSecret, authSecret, keyInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to extract and expand HKDF ikm: %w", err)
	}

	return ikm, nil
}

// prepareSalt return a 16-octet salt.
func (s *Service) prepareSalt() ([]byte, error) {
	salt := make([]byte, 0, 16)

	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	return salt, nil
}

// prepareNonceAndGCM return ready to use nonce, GCM and error.
func (s *Service) prepareNonceAndGCM(salt, ikm []byte) ([]byte, cipher.AEAD, error) {
	// Generate CEK according to RFC8291 3.4:
	// First 3 arguments to hkdf.New() response for PRK
	//
	// PRK = HMAC-SHA-256(salt, IKM)
	// CEK = HMAC-SHA-256(PRK, cek_info || 0x01)[0..15]
	cek, err := utils.HkdfExtractAndExpand(16, ikm, salt, cekInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract and expand HKDF nonce: %w", err)
	}

	// Generate NONCE according to RFC8291 3.4:
	// First 3 arguments to hkdf.New() response for PRK
	//
	// PRK = HMAC-SHA-256(salt, IKM)
	// NONCE = HMAC-SHA-256(PRK, nonce_info || 0x01)[0..11]
	nonce, err := utils.HkdfExtractAndExpand(12, ikm, salt, nonceInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract and expand HKDF nonce: %w", err)
	}

	// Cipher block
	c, err := aes.NewCipher(cek)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// GCM encryptor
	gcm, err := cipher.NewGCMWithNonceSize(c, 12)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return nonce, gcm, nil
}

// messageHeader generate and write an 86-octet header to buf.
func (s *Service) messageHeader(buf *bytes.Buffer, salt []byte, recordSize uint32) {
	// An 86-octet produced from the salt, record size of 4096, and application server public key
	header := buf.AvailableBuffer()[:86]

	// Writing salt
	copy(header, salt)

	// Writing record size as big endian uint32
	binary.BigEndian.PutUint32(header[16:21], recordSize)

	// Write key length (IDK why, in X9.62 it doesn't need)
	copy(header[20:], []byte{0x41})

	// Writing application server public key defined in X9.62
	copy(header[21:], s.publicKey)

	// Write header to dst buf.
	buf.Write(header)
}

// messageBody prepare, cipher and write plaintext to buf.
func (s *Service) messageBody(buf *bytes.Buffer, gcm cipher.AEAD, nonce, plaintext []byte) {
	// Get available buffer for reuse.
	pbuf := buf.AvailableBuffer()

	// Copy original plaintext.
	pbuf = append(pbuf, plaintext...)

	// Append padding delimiter.
	pbuf = append(pbuf, byte(0x02))

	// Encrypt the payload and get ciphertext.
	pbuf = gcm.Seal(pbuf[:0], nonce, pbuf, nil)

	// Write ciphertext to dst buf.
	buf.Write(pbuf)
}
