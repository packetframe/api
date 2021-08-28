package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect opens a connection to the database and runs migrations
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

// migrate runs migrations on all models
func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{})
}
