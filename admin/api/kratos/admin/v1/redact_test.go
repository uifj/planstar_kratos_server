package v1

import (
	"strings"
	"testing"
)

const (
	testPassword    = "SuperSecret123"
	testOldPassword = "SuperSecretOld123"
	testNewPassword = "SuperSecretNew123"
)

func TestRedactNeverLeaksPassword(t *testing.T) {
	tests := []struct {
		name     string
		req      interface{ Redact() string }
		wantPart string
		secrets  []string
	}{
		{
			name:     "login by username",
			req:      &LoginRequest{Username: "admin", Password: testPassword},
			wantPart: "username:admin",
			secrets:  []string{testPassword},
		},
		{
			name:     "login by email",
			req:      &LoginRequest{Email: "admin@example.com", Password: testPassword},
			wantPart: "email:admin@example.com",
			secrets:  []string{testPassword},
		},
		{
			name:     "login with no identity",
			req:      &LoginRequest{Password: testPassword},
			wantPart: "password:***",
			secrets:  []string{testPassword},
		},
		{
			name:     "change password",
			req:      &ChangePasswordRequest{OldPassword: testOldPassword, NewPassword: testNewPassword},
			wantPart: "old_password:*** new_password:***",
			secrets:  []string{testOldPassword, testNewPassword},
		},
		{
			name:     "create admin",
			req:      &CreateAdminRequest{Admin: &Admin{Name: "admin", Password: testPassword}},
			wantPart: "admin:admin",
			secrets:  []string{testPassword},
		},
		{
			name:     "update admin",
			req:      &UpdateAdminRequest{Admin: &Admin{Id: "11111111-1111-1111-1111-111111111111", Password: testPassword}},
			wantPart: "admin:11111111-1111-1111-1111-111111111111",
			secrets:  []string{testPassword},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.req.Redact()
			if !strings.Contains(got, "***") {
				t.Fatalf("redacted output %q does not contain ***", got)
			}
			if !strings.Contains(got, tt.wantPart) {
				t.Fatalf("redacted output %q does not contain %q", got, tt.wantPart)
			}
			for _, secret := range tt.secrets {
				if strings.Contains(got, secret) {
					t.Fatalf("redacted output %q leaks the password", got)
				}
			}
		})
	}
}

func TestRedactNilReceiver(t *testing.T) {
	var (
		login     *LoginRequest
		change    *ChangePasswordRequest
		create    *CreateAdminRequest
		update    *UpdateAdminRequest
		reacters  = []interface{ Redact() string }{login, change, create, update}
		wantEmpty = ""
	)
	for _, r := range reacters {
		if got := r.Redact(); got != wantEmpty {
			t.Fatalf("nil receiver redacted to %q, want empty", got)
		}
	}
}
