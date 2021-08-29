package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestZoneAddListFindDelete(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com")
	assert.Nil(t, err)
	err = ZoneAdd(db, "example2.com")
	assert.Nil(t, err)
	err = ZoneAdd(db, "example3.com")
	assert.Nil(t, err)

	zones, err := ZoneList(db)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(zones))

	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	err = ZoneDelete(db, example1.ID)
	assert.Nil(t, err)

	zones, err = ZoneList(db)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(zones))
}

func TestZoneRotateDNSSECKey(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com")
	assert.Nil(t, err)

	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	oldDNSSECKey := example1.DNSSEC.Private

	err = ZoneRotateDNSSECKey(db, example1.ID)
	assert.Nil(t, err)

	example1, err = ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	assert.NotEqual(t, oldDNSSECKey, example1.DNSSEC.Private)
}

// TestZoneAddDuplicate tests that adding a duplicate zone fails
func TestZoneAddDuplicate(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com")
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com")
	assert.NotNil(t, err)
}

// TestZoneUserAdd tests adding a user to a zone
func TestZoneUserAdd(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	// Add and find example1.com
	err = ZoneAdd(db, "example1.com")
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	// Create and add user1 to example1.com
	err = UserAdd(db, "user1@example.com", "password1")
	assert.Nil(t, err)
	user1, err := UserFind(db, "user1@example.com")
	assert.Nil(t, err)
	err = ZoneUserAdd(db, example1.ID, user1.ID)
	assert.Nil(t, err)

	// Add user1 again
	err = ZoneUserAdd(db, example1.ID, user1.ID)
	assert.NotNil(t, err)

	// List zone users
	example1, err = ZoneGet(db, example1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(example1.Users))
	assert.Equal(t, user1.ID, example1.Users[0])

	// Delete user
	err = ZoneUserDelete(db, example1.ID, user1.ID)
	assert.Nil(t, err)
}

// TestZoneSetSerial tests setting the serial of a zone
func TestZoneSetSerial(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	// Add and find example1.com
	err = ZoneAdd(db, "example1.com")
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	oldSerial := example1.Serial

	// Wait one second to allow for UNIX timestamp to change
	time.Sleep(time.Second)

	// Set serial
	err = ZoneSetSerial(db, example1.ID)
	assert.Nil(t, err)

	// Check new serial
	example1, err = ZoneGet(db, example1.ID)
	assert.Nil(t, err)
	assert.NotEqual(t, oldSerial, example1.Serial)
}

// TestZoneUserGetZones tests getting the zones of a user
func TestZoneUserGetZones(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	// Add and find example1.com
	err = ZoneAdd(db, "example1.com")
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	// Create and add user1 to example1.com
	err = UserAdd(db, "user1@example.com", "password1")
	assert.Nil(t, err)
	user1, err := UserFind(db, "user1@example.com")
	assert.Nil(t, err)
	err = ZoneUserAdd(db, example1.ID, user1.ID)
	assert.Nil(t, err)

	// Find zones for user
	zones, err := ZoneUserGetZones(db, user1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(zones))
	assert.Equal(t, example1.ID, zones[0].ID)
}