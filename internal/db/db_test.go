package db

import (
	"gorm.io/gorm"
	"testing"

	"github.com/stretchr/testify/assert"
)

// dbSetup sets up the test environment by opening a database connection, dropping all tables, and inserting test data
func dbSetup() (*gorm.DB, error) {
	db, err := Connect("host=localhost user=api password=api dbname=api port=5432 sslmode=disable")
	if err != nil {
		return nil, err
	}

	// Drop users
	err = db.Exec("DELETE FROM users;").Error
	if err != nil {
		return nil, err
	}

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1")
	if err != nil {
		return nil, err
	}

	return db, nil
}

func TestDbConnect(t *testing.T) {
	_, err := dbSetup()
	assert.Nil(t, err)
}
