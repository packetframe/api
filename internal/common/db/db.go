package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestSetup sets up the test environment by opening a database connection, dropping all tables, and inserting test data
func TestSetup() (*gorm.DB, error) {
	db, err := Connect("host=localhost user=api password=api dbname=api port=5432 sslmode=disable")
	if err != nil {
		return nil, err
	}

	// Drop tables
	for _, table := range []string{"records", "users", "zones"} {
		err = db.Exec("DELETE FROM " + table).Error
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

// Connect opens a connection to the database and runs migrations
func Connect(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Create UUID extension
	err = db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`).Error
	if err != nil {
		return nil, err
	}

	// Run schema migrations
	if err := migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}

// migrate runs migrations on all models
func migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Zone{}, &Record{})
}
