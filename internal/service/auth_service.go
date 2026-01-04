package service

import (
	"errors"
	"fmt"
	"math/rand"
	"net/smtp"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/phamngocan2412/camera-be-v1/internal/config"
	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/repository"
)

type OTPData struct {
	Code       string
	ExpiresAt  time.Time
	Attempts   int
	LastSentAt time.Time
}

type AuthService struct {
	repo       repository.UserRepository
	jwtSecret  string
	otpMap     map[string]OTPData
	mu         sync.RWMutex
	smtpConfig config.SMTPConfig
}

func NewAuthService(repo repository.UserRepository, secret string, smtpConfig config.SMTPConfig) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtSecret:  secret,
		otpMap:     make(map[string]OTPData),
		smtpConfig: smtpConfig,
	}
}

func (s *AuthService) Register(email, password, firstName, lastName, phoneNumber, countryCode string) (*models.AuthResponse, error) {
	// Check if email already exists
	existingUser, err := s.repo.FindByEmail(email)
	if err == nil {
		if !existingUser.EmailVerified {
			// Logic for existing unverified user: OVERWRITE old data with new registration details
			hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}

			existingUser.PasswordHash = string(hashed)
			existingUser.FirstName = firstName
			existingUser.LastName = lastName
			existingUser.PhoneNumber = phoneNumber
			existingUser.CountryCode = countryCode
			existingUser.CreatedAt = time.Now()

			if err := s.repo.Update(existingUser); err != nil {
				return nil, err
			}

			// Resend OTP
			if err := s.RequestOTP(email); err != nil {
				fmt.Printf("[WARNING] Failed to resend OTP email: %v\n", err)
			}

			// Return success response so Frontend treats it as a new registration
			return &models.AuthResponse{
				ID:          existingUser.ID,
				Email:       existingUser.Email,
				FirstName:   existingUser.FirstName,
				LastName:    existingUser.LastName,
				PhoneNumber: existingUser.PhoneNumber,
				CountryCode: existingUser.CountryCode,
				Token:       "",
			}, nil
		}
		return nil, errors.New("email already exists")
	}

	// Check if phone number already exists
	if _, err := s.repo.FindByPhoneNumber(phoneNumber); err == nil {
		return nil, errors.New("phone number already exists")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hashed),
		FirstName:    firstName,
		LastName:     lastName,
		PhoneNumber:  phoneNumber,
		CountryCode:  countryCode,
	}
	if err := s.repo.Create(user); err != nil {
		return nil, err
	}

	// Send OTP email for verification
	if err := s.RequestOTP(email); err != nil {
		// Log error but don't fail registration
		// User can still verify via OTP shown in logs
		fmt.Printf("[WARNING] Failed to send OTP email: %v\n", err)
	}

	// Return response without token - user must verify OTP first
	return &models.AuthResponse{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		PhoneNumber: user.PhoneNumber,
		CountryCode: user.CountryCode,
		Token:       "", // No token until email is verified
		IsVerified:  false,
	}, nil
}

func (s *AuthService) Login(email, password string) (*models.AuthResponse, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("wrong password")
	}

	// Check if email is verified
	fmt.Printf("[DEBUG] Login Check - User ID: %d, Email: %s, Verified: %v\n", user.ID, user.Email, user.EmailVerified)
	if !user.EmailVerified {
		return nil, errors.New("email not verified")
	}

	token, err := s.generateToken(user.ID, user.Email, user.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		PhoneNumber: user.PhoneNumber,
		CountryCode: user.CountryCode,
		Token:       token,
		IsVerified:  user.EmailVerified,
	}, nil
}

func (s *AuthService) generateToken(userID int, email string, tokenVersion int) (string, error) {
	claims := jwt.MapClaims{
		"user_id":       userID,
		"email":         email,
		"token_version": tokenVersion,
		"exp":           time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) RequestOTP(email string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check Rate Limiting (1 minute)
	if existing, ok := s.otpMap[email]; ok {
		if time.Since(existing.LastSentAt) < 1*time.Minute {
			return errors.New("please wait 1 minute before requesting a new OTP")
		}
	}

	// Generate 6 digit OTP
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	otp := fmt.Sprintf("%06d", rnd.Intn(1000000))

	// Store in map with expiry
	s.otpMap[email] = OTPData{
		Code:       otp,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		Attempts:   0,
		LastSentAt: time.Now(),
	}

	// DEBUG: Print OTP to console for testing
	fmt.Printf("\n========== OTP DEBUG ==========\n")
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("OTP Code: %s\n", otp)
	fmt.Printf("Expires: %s\n", time.Now().Add(5*time.Minute).Format("15:04:05"))
	fmt.Printf("===============================\n\n")

	// Send Email
	return s.SendVerificationEmail(email, otp)
}

func (s *AuthService) VerifyOTP(email, code string) (*models.AuthResponse, error) {
	s.mu.Lock() // Use Lock instead of RLock because we update Attempts
	data, exists := s.otpMap[email]

	if !exists {
		s.mu.Unlock()
		return nil, errors.New("otp not found or expired")
	}

	if time.Now().After(data.ExpiresAt) {
		delete(s.otpMap, email)
		s.mu.Unlock()
		return nil, errors.New("otp expired")
	}

	if data.Code != code {
		data.Attempts++
		s.otpMap[email] = data
		if data.Attempts >= 5 {
			delete(s.otpMap, email)
			s.mu.Unlock()
			return nil, errors.New("too many failed attempts, please request a new OTP")
		}
		s.mu.Unlock()
		return nil, errors.New("invalid otp")
	}

	// Clear OTP after success
	delete(s.otpMap, email)
	s.mu.Unlock()

	// Fetch user from database
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := s.repo.Update(user); err != nil {
		return nil, errors.New("failed to verify email")
	}

	// Generate token for verified user
	token, err := s.generateToken(user.ID, user.Email, user.TokenVersion)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		PhoneNumber: user.PhoneNumber,
		CountryCode: user.CountryCode,
		Token:       token,
	}, nil
}

func (s *AuthService) SendVerificationEmail(toEmail, otp string) error {
	auth := smtp.PlainAuth("", s.smtpConfig.Email, s.smtpConfig.Password, s.smtpConfig.Host)

	// HTML Body
	htmlBody := fmt.Sprintf(`
	<!DOCTYPE html>
	<html>
	<head>
		<style>
			body { font-family: Arial, sans-serif; background-color: #f4f4f4; padding: 20px; }
			.container { background-color: white; padding: 30px; border-radius: 8px; box-shadow: 0 0 10px rgba(0,0,0,0.1); max-width: 500px; margin: auto; }
			.header { text-align: center; color: #333; }
			.otp-box { font-size: 32px; font-weight: bold; color: #007bff; text-align: center; margin: 20px 0; letter-spacing: 5px; }
			.footer { text-align: center; color: #aaa; font-size: 12px; margin-top: 20px; }
		</style>
	</head>
	<body>
		<div class="container">
			<h2 class="header">Verification Request</h2>
			<p>Hello,</p>
			<p>Your verification code is:</p>
			<div class="otp-box">%s</div>
			<p>This code will expire in 5 minutes.</p>
			<p class="footer">If you didn't request this, please ignore this email.</p>
		</div>
	</body>
	</html>
	`, otp)

	msg := []byte(fmt.Sprintf("To: %s\r\n"+
		"Subject: Camera Security Verification Code\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n%s", toEmail, htmlBody))

	addr := fmt.Sprintf("%s:%d", s.smtpConfig.Host, s.smtpConfig.Port)
	if err := smtp.SendMail(addr, auth, s.smtpConfig.Email, []string{toEmail}, msg); err != nil {
		return err
	}
	return nil
}

func (s *AuthService) ForgotPassword(email string) error {
	_, err := s.repo.FindByEmail(email)
	if err != nil {
		// User not found - strict security might say "return nil" to avoid enumeration,
		// but typically for UX we might want to say "user not found" or just send nothing.
		// For this implementation, let's treat "user not found" as an error to the caller,
		// or handle it gracefully. The handler can decide whether to expose it.
		// If we want to mimic "always success" for security, we return nil here if ErrRecordNotFound.
		return errors.New("user not found")
	}

	// User exists, send OTP
	// We can reuse RequestOTP logic, but RequestOTP checks nothing, just generates/sends.
	return s.RequestOTP(email)
}

func (s *AuthService) ResetPassword(email, otp, newPassword string) error {
	// Verify OTP first
	s.mu.RLock()
	data, exists := s.otpMap[email]
	s.mu.RUnlock()

	if !exists {
		return errors.New("otp not found or expired")
	}

	if time.Now().After(data.ExpiresAt) {
		s.mu.Lock()
		delete(s.otpMap, email)
		s.mu.Unlock()
		return errors.New("otp expired")
	}

	if data.Code != otp {
		return errors.New("invalid otp")
	}

	// OTP is valid. Now update password.
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return errors.New("user not found")
	}

	// Check if new password is same as old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)); err == nil {
		return errors.New("new password cannot be the same as the old password")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashed)
	user.TokenVersion++ // Revoke all old tokens
	// Optionally mark as verified if they reset password via email OTP?
	// Usually resetting password via email implies email ownership.
	// user.EmailVerified = true

	if err := s.repo.Update(user); err != nil {
		return err
	}

	// Clear OTP
	s.mu.Lock()
	delete(s.otpMap, email)
	s.mu.Unlock()

	return nil
}

func (s *AuthService) VerifyResetOTP(email, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, exists := s.otpMap[email]
	if !exists {
		return errors.New("otp not found or expired")
	}

	if time.Now().After(data.ExpiresAt) {
		delete(s.otpMap, email)
		return errors.New("otp expired")
	}

	if data.Code != code {
		data.Attempts++
		s.otpMap[email] = data
		if data.Attempts >= 5 {
			delete(s.otpMap, email)
			return errors.New("too many failed attempts, please request a new OTP")
		}
		return errors.New("invalid otp")
	}

	// OTP is valid. We do NOT delete it here, so it can be used for the actual reset.
	return nil
}
