package i18n

import (
	"strings"
)

// Supported languages
const (
	LangEN = "en"
	LangVI = "vi"
)

var messages = map[string]map[string]string{
	LangEN: {
		"email_exists":           "Email already exists",
		"phone_exists":           "Phone number already exists",
		"user_not_found":         "User not found",
		"wrong_password":         "Wrong password",
		"email_not_verified":     "Email not verified",
		"otp_sent":               "Verification code sent",
		"otp_verified":           "OTP verified",
		"otp_expired":            "OTP expired",
		"invalid_otp":            "Invalid OTP",
		"otp_not_found":          "OTP not found or expired",
		"rate_limit_exceeded":    "Please wait 1 minute before requesting a new OTP",
		"pending_verification":   "Your account is pending verification, we have sent a new code to your email",
		"too_many_attempts":      "Too many failed attempts, please request a new OTP",
		"password_reset_success": "Password reset successfully",
		"same_password":          "New password cannot be the same as the old password",
		"forgot_password_msg":    "If this email exists, a verification code has been sent",
		"invalid_phone_format":   "Invalid phone number format",
		"invalid_phone":          "Invalid phone number",
		"invalid_credentials":    "Invalid email or password",
	},
	LangVI: {
		"email_exists":           "Email đã tồn tại",
		"phone_exists":           "Số điện thoại đã tồn tại",
		"user_not_found":         "Không tìm thấy người dùng",
		"wrong_password":         "Sai mật khẩu",
		"email_not_verified":     "Email chưa được xác thực",
		"otp_sent":               "Mã xác thực đã được gửi",
		"otp_verified":           "Mã OTP hợp lệ",
		"otp_expired":            "Mã OTP đã hết hạn",
		"invalid_otp":            "Mã OTP không hợp lệ",
		"otp_not_found":          "Mã OTP không tìm thấy hoặc đã hết hạn",
		"rate_limit_exceeded":    "Vui lòng đợi 1 phút trước khi yêu cầu mã OTP mới",
		"pending_verification":   "Tài khoản của bạn đang chờ xác thực, chúng tôi đã gửi lại mã mới vào hộp thư",
		"too_many_attempts":      "Quá nhiều lần thử sai, vui lòng yêu cầu mã OTP mới",
		"password_reset_success": "Đặt lại mật khẩu thành công",
		"same_password":          "Mật khẩu mới không được trùng với mật khẩu cũ",
		"forgot_password_msg":    "Nếu email tồn tại, mã xác thực đã được gửi",
		"invalid_phone_format":   "Định dạng số điện thoại không hợp lệ",
		"invalid_phone":          "Số điện thoại không hợp lệ",
		"invalid_credentials":    "Email hoặc mật khẩu không đúng",
	},
}

func GetMessage(langHeader string, key string) string {
	// 1. Normalization: Get first 2 characters
	lang := "en" // Default fallback
	if len(langHeader) >= 2 {
		lang = strings.ToLower(langHeader[:2])
	}

	// 2. Fallback: Check if language is supported, otherwise default to "en"
	if _, ok := messages[lang]; !ok {
		lang = "en"
	}

	// 3. Get message
	if msgMap, ok := messages[lang]; ok {
		if msg, exists := msgMap[key]; exists {
			return msg
		}
	}

	// 4. Missing Key Handling: Return the key itself
	return key
}
