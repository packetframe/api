package db

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestZoneAddListFindDelete(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com", "user1@example.com")
	assert.Nil(t, err)
	err = ZoneAdd(db, "example2.com", "user1@example.com")
	assert.Nil(t, err)
	err = ZoneAdd(db, "example3.com", "user1@example.com")
	assert.Nil(t, err)

	zones, err := ZoneList(db)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(zones))

	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	deleted, err := ZoneDelete(db, example1.ID)
	assert.Nil(t, err)
	assert.True(t, deleted)

	zones, err = ZoneList(db)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(zones))
}

func TestZoneRotateDNSSECKey(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com", "user1@example.com")
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
	db, err := TestSetup()
	assert.Nil(t, err)

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com", "user1@example.com")
	assert.Nil(t, err)

	err = ZoneAdd(db, "example1.com", "user1@example.com")
	assert.NotNil(t, err)
}

// TestZoneUserAddListDelete tests adding a user to a zone
func TestZoneUserAddListDelete(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Create user1@example.com
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)
	user1, err := UserFindByEmail(db, "user1@example.com")
	assert.Nil(t, err)

	// Create user2@example.com
	err = UserAdd(db, "user2@example.com", "password2", "example referrer")
	assert.Nil(t, err)
	user2, err := UserFindByEmail(db, "user2@example.com")
	assert.Nil(t, err)

	// Add and find example1.com
	err = ZoneAdd(db, "example1.com", user1.Email)
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	// Add user1 again
	err = ZoneUserAdd(db, example1.ID, user1.Email)
	assert.NotNil(t, err)

	// Add user2
	err = ZoneUserAdd(db, example1.ID, user2.Email)
	assert.Nil(t, err)

	// List zone users
	example1, err = ZoneGet(db, example1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(example1.Users))
	assert.Contains(t, example1.Users, user1.ID)
	assert.Contains(t, example1.Users, user2.ID)

	// Remove user
	err = ZoneUserDelete(db, example1.ID, user2.Email)
	assert.Nil(t, err)

	// List zone users again
	example1, err = ZoneGet(db, example1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(example1.Users))
	assert.Contains(t, example1.Users, user1.ID)
	assert.NotContains(t, example1.Users, user2.ID)
}

// TestZoneSetSerial tests setting the serial of a zone
func TestZoneSetSerial(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Add user1
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)

	// Add and find example1.com
	err = ZoneAdd(db, "example1.com", "user1@example.com")
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
	db, err := TestSetup()
	assert.Nil(t, err)

	// Create and add user1 to example1.com
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)
	user1, err := UserFindByEmail(db, "user1@example.com")
	assert.Nil(t, err)

	// Add and find example1.com
	err = ZoneAdd(db, "example1.com", user1.Email)
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	// Find zones for user
	zones, err := ZoneUserGetZones(db, user1.ID)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(zones))
	assert.Equal(t, example1.ID, zones[0].ID)
}

// TestZoneUserAuthorized tests checking if a user is authorized for a zone
func TestZoneUserAuthorized(t *testing.T) {
	db, err := TestSetup()
	assert.Nil(t, err)

	// Create and add user1 to example1.com
	err = UserAdd(db, "user1@example.com", "password1", "example referrer")
	assert.Nil(t, err)
	user1, err := UserFindByEmail(db, "user1@example.com")
	assert.Nil(t, err)

	// Add and find example1.com
	err = ZoneAdd(db, "example1.com", user1.Email)
	assert.Nil(t, err)
	example1, err := ZoneFind(db, "example1.com")
	assert.Nil(t, err)
	assert.NotNil(t, example1)

	// Test zone authorization
	err = ZoneUserAuthorized(db, example1.ID, user1.ID)
	assert.Nil(t, err)

	// Test zone authorization on random ID
	err = ZoneUserAuthorized(db, "not-a-real-zone", user1.ID)
	assert.NotNil(t, err)
}

func TestSuffixList(t *testing.T) {
	suffixes, err := SuffixList()
	assert.Nil(t, err)
	assert.Greater(t, len(suffixes), 9000)
	assert.Contains(t, suffixes, "com")
	assert.Contains(t, suffixes, "workers.dev")
}
