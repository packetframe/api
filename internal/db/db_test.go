package db

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// dbSetup sets up the test environment by opening a database connection, dropping all tables, and inserting test data
func dbSetup() (*gorm.DB, error) {
	db, err := Connect(os.Getenv("PACKETFRAME_API_TEST_DB"))
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

func TestDbConnect(t *testing.T) {
	_, err := dbSetup()
	assert.Nil(t, err)
}
