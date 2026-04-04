package helpers

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

func MD5Hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// Önerilen parametreler (2024+)
const (
	argonTime    uint32 = 3         // iterasyon sayısı
	argonMemory  uint32 = 64 * 1024 // 64 MB (kB cinsinden)
	argonThreads uint8  = 2
	argonKeyLen  uint32 = 32
	saltLen             = 16
)

// generateRandomBytes
func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func HashPasswordArgon2id(password string) (string, error) {
	salt, err := randomBytes(saltLen)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64Salt, b64Hash)

	return encoded, nil
}

// ComparePasswordArgon2id compares an Argon2id encoded hash with a plaintext password.
func ComparePasswordArgon2id(encodedHash, password string) (bool, error) {
	// encodedHash format example (like output of argon2id):
	// $argon2id$v=19$m=65536,t=3,p=2$base64(salt)$base64(hash)

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false, errors.New("invalid encoded hash format")
	}

	// Parse parameters
	var memory uint32
	var iterations uint32
	var parallelism uint8

	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, err
	}

	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil {
		return false, err
	}

	hash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	// Hash password with same params
	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(hash)))

	// Compare hashes
	if bytes.Equal(hash, computedHash) {
		return true, nil
	}

	return false, nil
}

func Base164Decode(e string) string {
	// Orijinal alfabedeki karakterler (Kiril ve Latin karışık - Sıralama çok kritiktir)
	const alphabet = "АВСDЕFGHIJKLМNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789.,~"

	// 1. Girdiyi temizle: Alfabede olmayan karakterleri kaldır
	// JS: e = e.replace(/[^АВСЕМA-Za-z0-9\.\,\~]/g, "");
	re := regexp.MustCompile(`[^АВСЕМA-Za-z0-9\.\,\~]`)
	e = re.ReplaceAllString(e, "")

	// 2. Alfabe için bir arama tablosu oluştur (Hızlı erişim için)
	alphaRunes := []rune(alphabet)
	alphaMap := make(map[rune]int)
	for i, r := range alphaRunes {
		alphaMap[r] = i
	}

	runes := []rune(e)
	var decodedBytes []byte
	r := 0

	// 3. Ana kod çözme döngüsü (Base64 bit kaydırma mantığı)
	for r < len(runes) {
		getIdx := func() int {
			if r >= len(runes) {
				return -1
			}
			char := runes[r]
			r++
			if idx, ok := alphaMap[char]; ok {
				return idx
			}
			return -1
		}

		i := getIdx()
		o := getIdx()
		a := getIdx()
		s := getIdx()

		// En az iki karakter (i ve o) mevcut olmalıdır
		if i == -1 || o == -1 {
			break
		}

		// İlk byte: i << 2 | o >> 4
		decodedBytes = append(decodedBytes, byte(i<<2|o>>4))

		// İkinci byte: (15 & o) << 4 | a >> 2
		// 64 indexi '~' karakterine denk gelir ve padding (dolgu) olarak kabul edilir
		if a != -1 && a != 64 {
			decodedBytes = append(decodedBytes, byte((15&o)<<4|a>>2))
		}

		// Üçüncü byte: (3 & a) << 6 | s
		if s != -1 && s != 64 {
			decodedBytes = append(decodedBytes, byte((3&a)<<6|s))
		}
	}

	// 4. JavaScript'in unescape(n) işlemini uygula ve sonucu döndür
	return jsUnescape(string(decodedBytes))
}

// jsUnescape, JavaScript'in deprecated unescape() fonksiyonunun davranışını taklit eder.
// %xx ve %uXXXX formatındaki kaçış dizilerini çözer.
func jsUnescape(s string) string {
	var res strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' {
			// %uXXXX formatını kontrol et (Unicode)
			if i+5 < len(s) && s[i+1] == 'u' {
				if val, err := strconv.ParseUint(s[i+2:i+6], 16, 16); err == nil {
					res.WriteRune(rune(val))
					i += 5
					continue
				}
			} else if i+2 < len(s) {
				// %xx formatını kontrol et (ASCII/Hex)
				if val, err := strconv.ParseUint(s[i+1:i+3], 16, 8); err == nil {
					res.WriteByte(byte(val))
					i += 2
					continue
				}
			}
		}
		res.WriteByte(s[i])
	}
	return res.String()
}
