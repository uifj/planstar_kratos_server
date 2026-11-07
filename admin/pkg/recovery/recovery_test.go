package recovery

import (
	"fmt"
	"strings"
	"testing"

	v1 "planstar/admin/api/kratos/admin/v1"
)

func TestDescribeRedactsCredentials(t *testing.T) {
	tests := []struct {
		name    string
		req     any
		secrets []string
	}{
		{
			name:    "login",
			req:     &v1.LoginRequest{Username: "admin", Password: "Login@Secret2026"},
			secrets: []string{"Login@Secret2026"},
		},
		{
			name:    "change password",
			req:     &v1.ChangePasswordRequest{OldPassword: "Old@Secret2026", NewPassword: "New@Secret2026"},
			secrets: []string{"Old@Secret2026", "New@Secret2026"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Describe(tt.req)
			if !strings.Contains(got, "***") {
				t.Fatalf("Describe(%T) = %q, want it to contain ***", tt.req, got)
			}
			for _, secret := range tt.secrets {
				if strings.Contains(got, secret) {
					t.Fatalf("Describe(%T) = %q leaks the password", tt.req, got)
				}
			}
		})
	}
}

func TestDescribeFallsBackToTypeName(t *testing.T) {
	type plainRequest struct {
		Password string
	}
	req := plainRequest{Password: "Plain@Secret2026"}

	got := Describe(req)
	if got != fmt.Sprintf("%T", req) {
		t.Fatalf("Describe(%T) = %q, want the type name only", req, got)
	}
	if strings.Contains(got, req.Password) {
		t.Fatalf("Describe(%T) = %q leaks the password", req, got)
	}
}
