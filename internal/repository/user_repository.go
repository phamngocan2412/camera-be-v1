package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/phamngocan2412/camera-be-v1/internal/models"
	"github.com/phamngocan2412/camera-be-v1/internal/platform/db"
)

type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByPhoneNumber(phoneNumber string) (*models.User, error)
	FindByID(id int) (*models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
}

type GORMUserRepository struct {
	db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) *GORMUserRepository {
	return &GORMUserRepository{db: db}
}

func (r *GORMUserRepository) FindByEmail(email string) (*models.User, error) {
	var u db.User
	if err := r.db.Where("email = ?", email).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &models.User{
		ID:            int(u.ID),
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		PhoneNumber:   u.PhoneNumber,
		CountryCode:   u.CountryCode,
		PasswordHash:  u.PasswordHash,
		EmailVerified: u.EmailVerified,
	}, nil
}

func (r *GORMUserRepository) FindByPhoneNumber(phoneNumber string) (*models.User, error) {
	var u db.User
	if err := r.db.Where("phone_number = ?", phoneNumber).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &models.User{
		ID:            int(u.ID),
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		PhoneNumber:   u.PhoneNumber,
		CountryCode:   u.CountryCode,
		PasswordHash:  u.PasswordHash,
		EmailVerified: u.EmailVerified,
	}, nil
}

func (r *GORMUserRepository) FindByID(id int) (*models.User, error) {
	var u db.User
	if err := r.db.First(&u, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}
	return &models.User{
		ID:            int(u.ID),
		Email:         u.Email,
		PasswordHash:  u.PasswordHash,
		EmailVerified: u.EmailVerified,
	}, nil
}

func (r *GORMUserRepository) Create(user *models.User) error {
	gormUser := db.User{
		Email:         user.Email,
		PasswordHash:  user.PasswordHash,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		PhoneNumber:   user.PhoneNumber,
		CountryCode:   user.CountryCode,
		EmailVerified: user.EmailVerified,
	}
	if err := r.db.Create(&gormUser).Error; err != nil {
		// Check for unique constraint violation (duplicate email)
		errStr := err.Error()
		if errors.Is(err, gorm.ErrDuplicatedKey) ||
			strings.Contains(errStr, "duplicate key") ||
			strings.Contains(errStr, "SQLSTATE 23505") {
			// Check if it's the email constraint
			if strings.Contains(errStr, "idx_users_email") ||
				strings.Contains(errStr, "users_email_key") {
				return errors.New("email already exists")
			}
			// Generic duplicate error (likely email since that's the main unique field)
			return errors.New("email already exists")
		}
		return err
	}
	user.ID = int(gormUser.ID)
	return nil
}

func (r *GORMUserRepository) Update(user *models.User) error {
	return r.db.Model(&db.User{ID: uint(user.ID)}).
		Updates(map[string]interface{}{
			"email":          user.Email,
			"password_hash":  user.PasswordHash,
			"first_name":     user.FirstName,
			"last_name":      user.LastName,
			"phone_number":   user.PhoneNumber,
			"country_code":   user.CountryCode,
			"email_verified": user.EmailVerified,
			"created_at":     user.CreatedAt,
		}).Error
}
