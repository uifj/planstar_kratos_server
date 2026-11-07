package biz

import (
	"github.com/go-kratos/kratos/v3/errors"
	v1 "planstar/admin/api/kratos/admin/v1"
)

var (
	// ErrAdminNotFound error admin not found.
	ErrAdminNotFound = errors.NotFound(v1.ErrorReason_ADMIN_NOT_FOUND.String(), "admin not found")
	// ErrInvalidCredentials is returned for any failed login, regardless of
	// whether the user exists, to avoid leaking which accounts are registered.
	ErrInvalidCredentials = errors.Unauthorized(v1.ErrorReason_INVALID_CREDENTIALS.String(), "invalid credentials")
	// ErrInvalidAdminID error admin id is not a valid UUID.
	ErrInvalidAdminID = errors.BadRequest(v1.ErrorReason_INVALID_ADMIN_ID.String(), "invalid admin id")
	// ErrInvalidOldPassword is returned when the current password supplied to
	// ChangePassword does not match the stored hash.
	ErrInvalidOldPassword = errors.BadRequest(v1.ErrorReason_INVALID_PASSWORD.String(), "invalid old password")
	// ErrWeakPassword is returned when the replacement password is too short.
	ErrWeakPassword = errors.BadRequest(v1.ErrorReason_INVALID_PASSWORD.String(), "new password must be at least 8 characters")

	// Console errors. The business service owns the blacklist, so an ops
	// outage is reported as 503 instead of falling back to local rules.
	ErrOpsUnavailable  = errors.ServiceUnavailable("OPS_UNAVAILABLE", "business ops service unavailable")
	ErrInvalidStatus   = errors.BadRequest("INVALID_STATUS", "invalid status")
	ErrInvalidDecision = errors.BadRequest("INVALID_DECISION", "decision must be approve or reject")
	ErrReportNotFound  = errors.NotFound("REPORT_NOT_FOUND", "report not found")
	ErrProjectNotFound = errors.NotFound("PROJECT_NOT_FOUND", "project not found")
	ErrUserNotFound    = errors.NotFound("USER_NOT_FOUND", "user not found")
)
