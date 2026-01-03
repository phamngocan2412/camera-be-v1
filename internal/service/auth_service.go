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
	Code      string
	ExpiresAt time.Time
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

	token, err := s.generateToken(user.ID, user.Email)
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

func (s *AuthService) generateToken(userID int, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

func (s *AuthService) RequestOTP(email string) error {
	// Generate 6 digit OTP
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	otp := fmt.Sprintf("%06d", rnd.Intn(1000000))

	// Store in map with expiry
	s.mu.Lock()
	s.otpMap[email] = OTPData{
		Code:      otp,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	s.mu.Unlock()

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
	s.mu.RLock()
	data, exists := s.otpMap[email]
	s.mu.RUnlock()

	if !exists {
		return nil, errors.New("otp not found or expired")
	}

	if time.Now().After(data.ExpiresAt) {
		s.mu.Lock()
		delete(s.otpMap, email)
		s.mu.Unlock()
		return nil, errors.New("otp expired")
	}

	if data.Code != code {
		return nil, errors.New("invalid otp")
	}

	// Clear OTP after success
	s.mu.Lock()
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
	token, err := s.generateToken(user.ID, user.Email)
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
