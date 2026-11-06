package secret

import "testing"

func TestHashAndVerify(t *testing.T) {
	encoded, err := HashPassword("123456")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !VerifyPassword("123456", encoded) {
		t.Fatal("valid password should verify")
	}
	if VerifyPassword("654321", encoded) {
		t.Fatal("wrong password must not verify")
	}
	if VerifyPassword("123456", "") {
		t.Fatal("empty hash must not verify")
	}
	if VerifyPassword("123456", "not-a-hash") {
		t.Fatal("malformed hash must not verify")
	}
	other, _ := HashPassword("123456")
	if encoded == other {
		t.Fatal("salt should make hashes differ")
	}
}
