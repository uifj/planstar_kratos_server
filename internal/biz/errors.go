package biz

import (
	v1 "github.com/go-kratos/kratos-layout/api/mall/v1"

	"github.com/go-kratos/kratos/v3/errors"
)

// Typed mall errors. Each reason is declared in
// api/mall/v1/error_reason.proto; biz is the only layer that turns a reason
// into a transport error, so service handlers just return these values.
var (
	ErrUserNotFound          = errors.NotFound(v1.ErrorReason_USER_NOT_FOUND.String(), "user not found")
	ErrUserMobileExists      = errors.Conflict(v1.ErrorReason_USER_MOBILE_EXISTS.String(), "mobile already registered")
	ErrUserPasswordInvalid   = errors.Unauthorized(v1.ErrorReason_USER_PASSWORD_INVALID.String(), "invalid password")
	ErrUserInvalidArgument   = errors.BadRequest(v1.ErrorReason_USER_INVALID_ARGUMENT.String(), "invalid user argument")
	ErrAddressNotFound       = errors.NotFound(v1.ErrorReason_ADDRESS_NOT_FOUND.String(), "address not found")
	ErrGoodsNotFound         = errors.NotFound(v1.ErrorReason_GOODS_NOT_FOUND.String(), "goods not found")
	ErrGoodsSkuNotFound      = errors.NotFound(v1.ErrorReason_GOODS_SKU_NOT_FOUND.String(), "sku not found")
	ErrGoodsInvalidArgument  = errors.BadRequest(v1.ErrorReason_GOODS_INVALID_ARGUMENT.String(), "invalid goods argument")
	ErrCartNotFound          = errors.NotFound(v1.ErrorReason_CART_NOT_FOUND.String(), "cart item not found")
	ErrOrderNotFound         = errors.NotFound(v1.ErrorReason_ORDER_NOT_FOUND.String(), "order not found")
	ErrOrderPriceChanged     = errors.Conflict(v1.ErrorReason_ORDER_PRICE_CHANGED.String(), "goods price changed, please retry")
	ErrOrderStatusConflict   = errors.Conflict(v1.ErrorReason_ORDER_STATUS_CONFLICT.String(), "order status conflict")
	ErrOrderInvalidArgument  = errors.BadRequest(v1.ErrorReason_ORDER_INVALID_ARGUMENT.String(), "invalid order argument")
	ErrInventoryNotEnough    = errors.Conflict(v1.ErrorReason_INVENTORY_NOT_ENOUGH.String(), "inventory not enough")
	ErrPaymentNotFound       = errors.NotFound(v1.ErrorReason_PAYMENT_NOT_FOUND.String(), "payment not found")
	ErrPaymentAmountMismatch = errors.BadRequest(v1.ErrorReason_PAYMENT_AMOUNT_MISMATCH.String(), "payment amount mismatch")
	ErrPaymentStatusConflict = errors.Conflict(v1.ErrorReason_PAYMENT_STATUS_CONFLICT.String(), "payment status conflict")
	ErrCaptchaInvalid        = errors.BadRequest(v1.ErrorReason_CAPTCHA_INVALID.String(), "invalid captcha")
	ErrPermissionDenied      = errors.Forbidden(v1.ErrorReason_PERMISSION_DENIED.String(), "permission denied")
	ErrCategoryNotFound      = errors.NotFound(v1.ErrorReason_CATEGORY_NOT_FOUND.String(), "category not found")
	ErrBrandNotFound         = errors.NotFound(v1.ErrorReason_BRAND_NOT_FOUND.String(), "brand not found")
	ErrTokenInvalid          = errors.Unauthorized(v1.ErrorReason_TOKEN_INVALID.String(), "invalid token")
	ErrInternal              = errors.InternalServer(v1.ErrorReason_INTERNAL_ERROR.String(), "internal error")
)
