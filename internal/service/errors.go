package service

import "errors"

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrEmailExists         = errors.New("email already exists")
	ErrPhoneExists         = errors.New("phone number already exists")
	ErrOTPExpired          = errors.New("otp expired")
	ErrInvalidOTP          = errors.New("invalid otp")
	ErrOTPNotFound         = errors.New("otp not found or expired")
	ErrRateLimitExceeded   = errors.New("please wait 1 minute before requesting a new OTP")
	ErrPendingVerification = errors.New("pending_verification")
	ErrEmailNotVerified    = errors.New("email not verified")
	ErrWrongPassword       = errors.New("wrong password")
	ErrSamePassword        = errors.New("new password cannot be the same as the old password")
)
