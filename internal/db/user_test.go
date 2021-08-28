package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbUserList(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	// Add 2 more users
	err = UserAdd(db, "user2@example.com", "password2")
	assert.Nil(t, err)
	err = UserAdd(db, "user3@example.com", "password3")
	assert.Nil(t, err)

	// Assert that there are 3 users
	users, err := UserList(db)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(users))
}

func TestDbUserAddDelete(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	// Find user1
	user1, err := UserFind(db, "user1@example.com")
	assert.Nil(t, err)

	// Delete user1
	err = UserDelete(db, user1.ID)
	assert.Nil(t, err)

	// Assert that user1 no longer exists
	user1, err = UserFind(db, "user1@example.com")
	assert.Nil(t, err)
	assert.Nil(t, user1)
}

func TestDbUserGroupAddDelete(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	// Find user1
	user1, err := UserFind(db, "user1@example.com")
	assert.Nil(t, err)

	// Add admin group to user1
	err = UserGroupAdd(db, user1.ID, GroupAdmin)
	assert.Nil(t, err)

	// Find user1
	user1, err = UserFind(db, "user1@example.com")
	assert.Nil(t, err)

	// Remove the admin group from user1
	err = UserGroupDelete(db, user1.ID, GroupAdmin)
	assert.Nil(t, err)

	// Assert that user1 is no longer part of the admin group
	user1, err = UserFind(db, "user1@example.com")
	assert.Nil(t, err)
	assert.NotContains(t, user1.Groups, GroupAdmin)
}

func TestDbUserResetPassword(t *testing.T) {
	db, err := dbSetup()
	assert.Nil(t, err)

	// Find user1
	user1, err := UserFind(db, "user1@example.com")
	assert.Nil(t, err)

	oldPassword := user1.PasswordHash

	err = UserResetPassword(db, user1.ID, "new-password")
	assert.Nil(t, err)

	// Find user1
	user1, err = UserFind(db, "user1@example.com")
	assert.Nil(t, err)

	// Assert that there are 3 users
	assert.NotEqual(t, oldPassword, user1.PasswordHash)
}
