package database

import (
	"auction/internal/config"
	"auction/internal/models"
	"errors"
	"log"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Open(cfg config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	if err := db.AutoMigrate(
		&models.User{},
		&models.Item{},
		&models.Auction{},
		&models.Bid{},
		&models.Payment{},
		&models.Review{},
	); err != nil {
		return nil, err
	}
	if err := seedAdmin(db, cfg); err != nil {
		return nil, err
	}
	return db, nil
}

func seedAdmin(db *gorm.DB, cfg config.Config) error {
	var n int64
	if err := db.Model(&models.User{}).Where("username = ?", cfg.AdminUser).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u := models.User{
		Username:     cfg.AdminUser,
		PasswordHash: string(hash),
		DisplayName:  "Administrator",
		IsAdmin:      true,
	}
	if err := db.Create(&u).Error; err != nil {
		return err
	}
	log.Printf("seeded admin user %q (change ADMIN_PASS in production)", cfg.AdminUser)
	return nil
}

// IsNotFound reports whether err is gorm.ErrRecordNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
