package db

import (
	"log"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// User struct đại diện cho bảng users
type User struct {
	ID            uint      `gorm:"primaryKey"`
	Email         string    `gorm:"uniqueIndex;not null"`
	PasswordHash  string    `gorm:"column:password_hash;not null"`
	FirstName     string    `gorm:"column:first_name;not null"`
	LastName      string    `gorm:"column:last_name;not null"`
	PhoneNumber   string    `gorm:"uniqueIndex;not null"`
	CountryCode   string    `gorm:"column:country_code;not null"`
	EmailVerified bool      `gorm:"column:email_verified;default:false"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
}

func NewDatabase(dsn string) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	const maxRetries = 10
	const retryInterval = 2 * time.Second

	for i := 1; i <= maxRetries; i++ {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("[Lần %d/%d] Đang thử kết nối lại DB... Lỗi: %v", i, maxRetries, err)
		time.Sleep(retryInterval)
	}

	if err != nil {
		return nil, err
	}

	// --- PHẦN THÊM MỚI: AUTO MIGRATE ---
	log.Println("Đang tiến hành AutoMigrate dữ liệu...")
	err = db.AutoMigrate(&User{}) // Tự động tạo bảng users nếu chưa có
	if err != nil {
		log.Printf("Lỗi AutoMigrate: %v", err)
		return nil, err
	}
	// ------------------------------------

	sqlDB, _ := db.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	log.Println("Kết nối database và Migrate thành công!")
	return db, nil
}
