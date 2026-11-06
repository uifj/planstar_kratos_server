package data

import (
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/go-kratos/kratos-layout/internal/biz"
	"github.com/go-kratos/kratos-layout/internal/conf"
	"golang.org/x/crypto/pbkdf2"
)

const (
	defaultPBKDF2Iterations = 10000
	pbkdf2SaltBytes         = 16
	pbkdf2KeyBytes          = 64
)

// pbkdf2Hasher stores self-describing hashes:
//
//	pbkdf2_sha512$<iterations>$<salt-hex>$<key-hex>
//
// Keeping the iteration count in the hash lets the configured value be raised
// later without invalidating existing accounts.
type pbkdf2Hasher struct{ iterations int }

func NewPasswordHasher(m *conf.Mall) biz.PasswordHasher {
	iter := defaultPBKDF2Iterations
	if m != nil {
		if a := m.GetAuth(); a != nil && a.GetPbkdf2Iterations() > 0 {
			iter = int(a.GetPbkdf2Iterations())
		}
	}
	return &pbkdf2Hasher{iterations: iter}
}

func (h *pbkdf2Hasher) Hash(password string) (string, error) {
	salt := make([]byte, pbkdf2SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := pbkdf2.Key([]byte(password), salt, h.iterations, pbkdf2KeyBytes, sha512.New)
	return fmt.Sprintf("pbkdf2_sha512$%d$%s$%s",
		h.iterations, hex.EncodeToString(salt), hex.EncodeToString(key)), nil
}

func (h *pbkdf2Hasher) Verify(hash, password string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2_sha512" {
		return false
	}
	var iter int
	if _, err := fmt.Sscanf(parts[1], "%d", &iter); err != nil || iter <= 0 {
		return false
	}
	salt, err := hex.DecodeString(parts[2])
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}
	got := pbkdf2.Key([]byte(password), salt, iter, len(want), sha512.New)
	return subtle.ConstantTimeCompare(got, want) == 1
}
