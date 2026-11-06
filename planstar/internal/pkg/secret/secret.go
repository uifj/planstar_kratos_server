package secret

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// 密码哈希：PBKDF2-HMAC-SHA256（纯标准库），格式 $pbkdf2-sha256$iter$salt$hash。

const pbkdf2Iters = 10000

// HashPassword 生成密码哈希。
func HashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk := pbkdf2SHA256([]byte(pw), salt, pbkdf2Iters, 32)
	return fmt.Sprintf("$pbkdf2-sha256$%d$%s$%s", pbkdf2Iters, hex.EncodeToString(salt), hex.EncodeToString(dk)), nil
}

// VerifyPassword 校验密码。
func VerifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 || parts[1] != "pbkdf2-sha256" {
		return false
	}
	var iters int
	if _, err := fmt.Sscanf(parts[2], "%d", &iters); err != nil {
		return false
	}
	salt, err := hex.DecodeString(parts[3])
	if err != nil {
		return false
	}
	want, err := hex.DecodeString(parts[4])
	if err != nil {
		return false
	}
	got := pbkdf2SHA256([]byte(pw), salt, iters, len(want))
	return hmac.Equal(got, want)
}

func pbkdf2SHA256(pw, salt []byte, iter, keyLen int) []byte {
	prf := hmac.New(sha256.New, pw)
	hashLen := prf.Size()
	numBlocks := (keyLen + hashLen - 1) / hashLen
	var buf [4]byte
	dk := make([]byte, 0, numBlocks*hashLen)
	u := make([]byte, hashLen)
	for block := 1; block <= numBlocks; block++ {
		prf.Reset()
		prf.Write(salt)
		buf[0] = byte(block >> 24)
		buf[1] = byte(block >> 16)
		buf[2] = byte(block >> 8)
		buf[3] = byte(block)
		prf.Write(buf[:4])
		dk = prf.Sum(dk)
		t := dk[len(dk)-hashLen:]
		copy(u, t)
		for n := 2; n <= iter; n++ {
			prf.Reset()
			prf.Write(u)
			u = u[:0]
			u = prf.Sum(u)
			for x := range u {
				t[x] ^= u[x]
			}
		}
	}
	return dk[:keyLen]
}
