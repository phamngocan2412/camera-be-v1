package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/repository"
)

type AuthService struct {
	repo      repository.UserRepository
	jwtSecret string
}

func NewAuthService(repo repository.UserRepository, secret string) *AuthService {
	return &AuthService{repo: repo, jwtSecret: secret}
}

func (s *AuthService) Register(email, password, firstName, lastName, phoneNumber, countryCode string) (*models.AuthResponse, error) {
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

	token, err := s.generateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{ID: user.ID, Email: user.Email, Token: token}, nil
}

func (s *AuthService) Login(email, password string) (*models.AuthResponse, error) {
	user, err := s.repo.FindByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := s.generateToken(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &models.AuthResponse{ID: user.ID, Email: user.Email, Token: token}, nil
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
