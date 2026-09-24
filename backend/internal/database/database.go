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
	if err := renameAccessColumns(db); err != nil {
		return err
	}
	return db.AutoMigrate(
		&model.User{},
		&model.Space{},
		&model.SpaceMember{},
		&model.Gateway{},
		&model.APIGroup{},
		&model.API{},
		&model.APIVersion{},
		&model.Upstream{},
		&model.UpstreamTarget{},
		&model.UpstreamGateway{},
	)
}

func renameAccessColumns(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.API{}) {
		return nil
	}
	pairs := [][2]string{
		{"path", "access_path"},
		{"methods", "access_methods"},
		{"strip_path", "access_strip_path"},
		{"protocol", "service_protocol"},
		{"host_kind", "service_host_kind"},
		{"host", "service_host"},
		{"upstream_id", "service_upstream_id"},
		{"port", "service_port"},
		{"retries", "service_retries"},
		{"connect_timeout", "service_connect_timeout"},
		{"write_timeout", "service_write_timeout"},
		{"read_timeout", "service_read_timeout"},
	}
	for _, pair := range pairs {
		hasOld := db.Migrator().HasColumn(&model.API{}, pair[0])
		hasNew := db.Migrator().HasColumn(&model.API{}, pair[1])
		if hasOld && !hasNew {
			if err := db.Migrator().RenameColumn(&model.API{}, pair[0], pair[1]); err != nil {
				return fmt.Errorf("rename apis.%s: %w", pair[0], err)
			}
		}
	}
	return nil
}
