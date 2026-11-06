package service

import v1 "github.com/go-kratos/kratos-layout/api/mall/v1"

// okCheck is the uniform success reply for the mall's command RPCs. The
// source project returned this from every mutating admin/shop handler.
func okCheck() *v1.CheckResponse { return &v1.CheckResponse{Success: true} }
