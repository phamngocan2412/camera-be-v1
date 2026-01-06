package service

import (
	"context"
	"fmt"
	"math/rand"
	"net/smtp"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/phamngocan2412/camera-be-v1/internal/config"
	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/repository"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo       repository.UserRepository
	jwtSecret  string
	redis      *redis.Client
	logger     *zap.Logger
	smtpConfig config.SMTPConfig
}

func NewAuthService(repo repository.UserRepository, secret string, rdb *redis.Client, logger *zap.Logger, smtpConfig config.SMTPConfig) *AuthService {
	return &AuthService{
		repo:       repo,
		jwtSecret:  secret,
		redis:      rdb,
		logger:     logger,
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
				s.logger.Error("failed to hash password", zap.Error(err))
				return nil, err
			}

			existingUser.PasswordHash = string(hashed)
			existingUser.FirstName = firstName
			existingUser.LastName = lastName
			existingUser.PhoneNumber = phoneNumber
			existingUser.CountryCode = countryCode
			existingUser.CreatedAt = time.Now()

			if err := s.repo.Update(existingUser); err != nil {
				s.logger.Error("failed to update unverified user", zap.Error(err))
				return nil, err
			}

			// Resend OTP (Async)
			go func() {
				if err := s.RequestOTP(email); err != nil {
					s.logger.Warn("failed to resend OTP email (async)", zap.Error(err))
				}
			}()

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
		return nil, ErrEmailExists
	}

	// Check if phone number already exists
	if _, err := s.repo.FindByPhoneNumber(phoneNumber); err == nil {
		return nil, ErrPhoneExists
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
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
		s.logger.Error("failed to create user", zap.Error(err))
		return nil, err
	}

	// Send OTP email for verification (Async)
	go func() {
		if err := s.RequestOTP(email); err != nil {
			s.logger.Warn("failed to send OTP email (async)", zap.Error(err))
		}
	}()

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
		return nil, ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrWrongPassword
	}

	// Check if email is verified
	s.logger.Debug("login check", zap.Int("user_id", user.ID), zap.String("email", user.Email), zap.Bool("verified", user.EmailVerified))
	if !user.EmailVerified {
		return nil, ErrEmailNotVerified
	}

	token, err := s.generateToken(user.ID, user.Email, user.TokenVersion)
	if err != nil {
		s.logger.Error("failed to generate token", zap.Error(err))
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
	ctx := context.Background()
	key := "otp:" + email

	// Check Rate Limiting (TTL check)
	// If key exists and TTL is > 4 minutes (meaning requested less than 1 min ago given 5 min expiry), block?
	// Simpler: Set a separate rate limit key or just check if OTP exists.
	// For now, adhere to "wait 1 minute".
	// We can store a separate key "rate_limit:email" with 1 min TTL.
	rateKey := "rate_limit:" + email
	if exists, _ := s.redis.Exists(ctx, rateKey).Result(); exists > 0 {
		return ErrRateLimitExceeded
	}

	// Generate 6 digit OTP
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	otp := fmt.Sprintf("%06d", rnd.Intn(1000000))

	// Store in Redis with 5 minute expiry
	// Also set rate limit key for 1 minute
	pipe := s.redis.Pipeline()
	pipe.Set(ctx, key, otp, 5*time.Minute)
	pipe.Set(ctx, rateKey, "1", 1*time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		s.logger.Error("failed to save OTP to redis", zap.Error(err))
		return err
	}

	// DEBUG: Log OTP for dev environment if needed, but use Debug level
	s.logger.Debug("OTP Generated", zap.String("email", email), zap.String("otp", otp))

	// Send Email
	return s.SendVerificationEmail(email, otp)
}

func (s *AuthService) VerifyOTP(email, code string) (*models.AuthResponse, error) {
	ctx := context.Background()
	key := "otp:" + email

	// Get OTP from Redis
	storedOTP, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, ErrOTPNotFound
	} else if err != nil {
		s.logger.Error("redis error", zap.Error(err))
		return nil, err
	}

	if storedOTP != code {
		// Attempts? Redis doesn't natively track attempts in the value unless we use a hash or separate key.
		// For simplicity, we just fail. If strict attempt limiting is needed, we'd use 'incr' on a separate key.
		return nil, ErrInvalidOTP
	}

	// Clear OTP after success
	s.redis.Del(ctx, key)

	// Fetch user from database
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Mark email as verified
	user.EmailVerified = true
	if err := s.repo.Update(user); err != nil {
		s.logger.Error("failed to verify email", zap.Error(err))
		return nil, err
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
		s.logger.Error("failed to send email", zap.Error(err))
		return err
	}
	return nil
}

func (s *AuthService) ForgotPassword(email string) error {
	_, err := s.repo.FindByEmail(email)
	if err != nil {
		return ErrUserNotFound
	}
	go func() {
		if err := s.RequestOTP(email); err != nil {
			s.logger.Warn("failed to send forgot password otp (async)", zap.Error(err))
		}
	}()
	return nil
}

func (s *AuthService) ResetPassword(email, otp, newPassword string) error {
	// Verify OTP first
	ctx := context.Background()
	key := "otp:" + email

	storedOTP, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrOTPNotFound
	} else if err != nil {
		return err
	}

	if storedOTP != otp {
		return ErrInvalidOTP
	}

	// OTP is valid. Now update password.
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return ErrUserNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(newPassword)); err == nil {
		return ErrSamePassword
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashed)
	user.TokenVersion++ // Revoke all old tokens

	if err := s.repo.Update(user); err != nil {
		return err
	}

	// Clear OTP
	s.redis.Del(ctx, key)

	return nil
}

func (s *AuthService) VerifyResetOTP(email, code string) error {
	ctx := context.Background()
	key := "otp:" + email

	storedOTP, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrOTPNotFound
	} else if err != nil {
		return err
	}

	if storedOTP != code {
		return ErrInvalidOTP
	}

	// OTP is valid.
	return nil
}
