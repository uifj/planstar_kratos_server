package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/pbkdf2"
)

func main() {
	salt, _ := hex.DecodeString("781f496fd9233c152837110ac2ce8a79")
	want, _ := hex.DecodeString("0408f25f5393afa7259e9fad0fceb1a2f3e406641d221b0b5562fb9ad46f8406")
	dk := pbkdf2.Key([]byte("12345678"), salt, 10000, len(want), sha512.New)
	fmt.Printf("match=%v\n", hex.EncodeToString(dk) == hex.EncodeToString(want))
	// also try keyLen=32
	dk32 := pbkdf2.Key([]byte("12345678"), salt, 10000, 32, sha512.New)
	fmt.Printf("match32=%v\n", hex.EncodeToString(dk32) == "0408f25f5393afa7259e9fad0fceb1a2f3e406641d221b0b5562fb9ad46f8406")
}