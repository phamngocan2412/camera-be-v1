package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nyaruka/phonenumbers"

	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/service"
)

type AuthHandler struct {
	service *service.AuthService
}

func NewAuthHandler(s *service.AuthService) *AuthHandler {
	return &AuthHandler{service: s}
}

// Register godoc
// @Summary      Register a new user
// @Description  Register a new user with email and password
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.RegisterRequest  true  "Registration request"
// @Success      201      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Failure      409      {object}  map[string]string
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate Phone Number
	var parsedNumber *phonenumbers.PhoneNumber
	var err error

	// Check if CountryCode looks like a calling code (starts with +)
	if len(req.CountryCode) > 0 && req.CountryCode[0] == '+' {
		// Concatenate: +84 + 090... -> +84090...
		// parse with default region empty, trusting the + prefix
		fullNumber := req.CountryCode + req.PhoneNumber
		parsedNumber, err = phonenumbers.Parse(fullNumber, "")
	} else {
		// Treat CountryCode as Region Code (e.g. "VN")
		parsedNumber, err = phonenumbers.Parse(req.PhoneNumber, req.CountryCode)
	}

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone number format"})
		return
	}
	if !phonenumbers.IsValidNumber(parsedNumber) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid phone number"})
		return
	}

	// Format phone number to E.164 standard before saving
	formattedNum := phonenumbers.Format(parsedNumber, phonenumbers.E164)
	req.PhoneNumber = formattedNum

	res, err := h.service.Register(req.Email, req.Password, req.FirstName, req.LastName, req.PhoneNumber, req.CountryCode)
	if err != nil {
		if err.Error() == "pending_verification" {
			c.JSON(http.StatusOK, gin.H{"message": "Tài khoản của bạn đang chờ xác thực, chúng tôi đã gửi lại mã mới vào hộp thư"})
			return
		}
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
		} else if err.Error() == "phone number already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "phone number already exists"})
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusCreated, res)
}

// Login godoc
// @Summary      Login user
// @Description  Login with email and password to get JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.LoginRequest  true  "Login request"
// @Success      200      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Failure      401      {object}  map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.Login(req.Email, req.Password)
	if err != nil {
		if err.Error() == "email not verified" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Email not verified"})
			return
		}
		// Generic error message for security (User Enumeration protection)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}
	c.JSON(http.StatusOK, res)
}

// RequestOTP godoc
// @Summary      Request OTP verification code
// @Description  Send a 6-digit OTP code to user's email for verification
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string}  true  "Email address"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /auth/request-otp [post]
func (h *AuthHandler) RequestOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.RequestOTP(req.Email); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification code sent"})
}

// VerifyOTP godoc
// @Summary      Verify OTP code
// @Description  Verify the OTP code and return user authentication token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string,otp=string}  true  "Email and OTP code"
// @Success      200      {object}  models.AuthResponse
// @Failure      400      {object}  map[string]string
// @Router       /auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		OTP   string `json:"otp" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authResponse, err := h.service.VerifyOTP(req.Email, req.OTP)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, authResponse)
}

// ForgotPassword godoc
// @Summary      Request password reset
// @Description  Send OTP to email for password reset checking if user exists
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.ForgotPasswordRequest  true  "Forgot Password request"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      404      {object}  map[string]string
// @Router       /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Security: Always return 200 OK even if user not found
	if err := h.service.ForgotPassword(req.Email); err != nil {
		// Log the error internally but don't expose it
		// fmt.Printf("Forgot password checking error: %v\n", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "If this email exists, a verification code has been sent"})
}

// ResetPassword godoc
// @Summary      Reset password
// @Description  Reset password using OTP
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      models.ResetPasswordRequest  true  "Reset Password request"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Router       /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ResetPassword(req.Email, req.OTP, req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// VerifyResetOTP godoc
// @Summary      Verify OTP for Password Reset
// @Description  Check if the OTP is valid without consuming it (for multi-step reset flow)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string,otp=string}  true  "Email and OTP code"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Router       /auth/verify-reset-otp [post]
func (h *AuthHandler) VerifyResetOTP(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		OTP   string `json:"otp" binding:"required,len=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.VerifyResetOTP(req.Email, req.OTP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP verified"})
}
