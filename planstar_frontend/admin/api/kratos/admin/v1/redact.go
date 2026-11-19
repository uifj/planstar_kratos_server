package v1

// This file makes credential-bearing requests safe to log: the Kratos logging
// middleware prefers Redacter over fmt.Stringer, so request bodies never carry
// plaintext passwords into logs.

// Redact renders the login request without the password.
func (x *LoginRequest) Redact() string {
	if x == nil {
		return ""
	}
	switch {
	case x.Username != "":
		return "username:" + x.Username + " password:***"
	case x.Email != "":
		return "email:" + x.Email + " password:***"
	default:
		return "password:***"
	}
}

// Redact renders the password change request without either password.
func (x *ChangePasswordRequest) Redact() string {
	if x == nil {
		return ""
	}
	return "old_password:*** new_password:***"
}

// Redact renders the create request without the password.
func (x *CreateAdminRequest) Redact() string {
	if x == nil {
		return ""
	}
	if x.Admin == nil {
		return "password:***"
	}
	return "admin:" + x.Admin.Name + " password:***"
}

// Redact renders the update request without the password.
func (x *UpdateAdminRequest) Redact() string {
	if x == nil {
		return ""
	}
	if x.Admin == nil {
		return "password:***"
	}
	return "admin:" + x.Admin.Id + " password:***"
}
