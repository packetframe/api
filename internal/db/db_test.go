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

	// Drop tables
	for _, table := range []string{"users", "zones", "records"} {
		err = db.Exec("DELETE FROM " + table).Error
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func TestDbConnect(t *testing.T) {
	_, err := dbSetup()
	assert.Nil(t, err)
}
