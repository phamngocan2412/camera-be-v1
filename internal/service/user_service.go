package service

import (
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetProfile(userID int) (*models.UserProfile, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}
	return &models.UserProfile{
		ID:          user.ID,
		Email:       user.Email,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		PhoneNumber: user.PhoneNumber,
		CountryCode: user.CountryCode,
	}, nil
}

func (s *UserService) UpdateProfile(userID int, req models.UpdateProfileRequest) (*models.UserProfile, error) {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	updated := false
	if req.Email != "" && req.Email != user.Email {
		if _, err := s.repo.FindByEmail(req.Email); err == nil {
			return nil, errors.New("email already exists")
		}
		user.Email = req.Email
		updated = true
	}

	if updated {
		if err := s.repo.Update(user); err != nil {
			return nil, err
		}
	}

	return &models.UserProfile{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
	}, nil
}

func (s *UserService) ChangePassword(userID int, req models.ChangePasswordRequest) error {
	user, err := s.repo.FindByID(userID)
	if err != nil {
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return errors.New("old password incorrect")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.PasswordHash = string(hashed)
	return s.repo.Update(user)
}
