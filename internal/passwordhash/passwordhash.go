package passwordhash

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

const (
	Scheme     = "pbkdf2-sha256"
	iterations = 120000
	saltBytes  = 16
	keyBytes   = 32
)

func New(password string) (string, string, error) {
	salt := make([]byte, saltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", "", err
	}
	saltHex := hex.EncodeToString(salt)
	key := pbkdf2Key([]byte(normalizePassword(password)), salt, iterations, keyBytes)
	return saltHex, fmt.Sprintf("%s$%d$%s$%s", Scheme, iterations, saltHex, hex.EncodeToString(key)), nil
}

func Verify(legacySaltHex string, encodedHash string, password string) (bool, bool) {
	encodedHash = strings.TrimSpace(encodedHash)
	if strings.HasPrefix(encodedHash, Scheme+"$") {
		return verifyPBKDF2(encodedHash, password), false
	}
	legacy := LegacyHash(legacySaltHex, password)
	if subtle.ConstantTimeCompare([]byte(legacy), []byte(encodedHash)) == 1 {
		return true, true
	}
	return false, false
}

func LegacyHash(saltHex string, password string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(saltHex) + ":" + normalizePassword(password)))
	return hex.EncodeToString(sum[:])
}

func verifyPBKDF2(encodedHash string, password string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 4 || parts[0] != Scheme {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 10000 || iter > 2000000 {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil || len(salt) < 8 {
		return false
	}
	want, err := hex.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}
	got := pbkdf2Key([]byte(normalizePassword(password)), salt, iter, len(want))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func pbkdf2Key(password []byte, salt []byte, iter int, keyLen int) []byte {
	hLen := sha256.Size
	numBlocks := (keyLen + hLen - 1) / hLen
	out := make([]byte, 0, numBlocks*hLen)
	var counter [4]byte
	for block := 1; block <= numBlocks; block++ {
		binary.BigEndian.PutUint32(counter[:], uint32(block))
		mac := hmac.New(sha256.New, password)
		mac.Write(salt)
		mac.Write(counter[:])
		u := mac.Sum(nil)
		t := append([]byte(nil), u...)
		for i := 1; i < iter; i++ {
			mac = hmac.New(sha256.New, password)
			mac.Write(u)
			u = mac.Sum(nil)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		out = append(out, t...)
	}
	return out[:keyLen]
}

func normalizePassword(password string) string {
	return strings.TrimSpace(password)
}
