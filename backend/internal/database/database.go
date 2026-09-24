package database

import (
	"fmt"

	"github.com/kelvins-io/api-gateway-manager/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	return db, nil
}

func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.Space{},
		&model.SpaceMember{},
		&model.Gateway{},
		&model.APIGroup{},
		&model.API{},
		&model.APIVersion{},
	)
}
