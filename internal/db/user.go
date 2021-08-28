package db

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/auth"
)

var (
	GroupEnabled = "ENABLED" // User is permitted to make API requests
	GroupAdmin   = "ADMIN"   // User is permitted to modify all resources
)

type User struct {
	ID           string         `gorm:"primaryKey"`
	Email        string         `gorm:"uniqueIndex"`
	Groups       pq.StringArray `gorm:"type:text[]"`
	PasswordHash []byte
	APIKey       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt
}

func (u *User) BeforeCreate(_ *gorm.DB) (err error) {
	// Set a UUID before user creation
	u.ID = uuid.New().String()
	return
}

// UserAdd creates a new User
func UserAdd(db *gorm.DB, email string, password string) error {
	passwordHash, err := auth.Hash(password)
	if err != nil {
		return err
	}
	apiKey, err := auth.RandomString(48)
	if err != nil {
		return err
	}
	return db.Create(&User{
		Email:        email,
		PasswordHash: passwordHash,
		APIKey:       apiKey,
		Groups:       []string{GroupEnabled},
	}).Error
}

// UserFind finds a user by email and returns nil if no user exists
func UserFind(db *gorm.DB, email string) (*User, error) {
	var user User
	err := db.Limit(1).Find(&user, "email = ?", email).Error
	if err != nil {
		return nil, err
	}
	if user.ID == "" {
		return nil, nil
	}

	return &user, nil
}

// UserDelete deletes a user
func UserDelete(db *gorm.DB, uuid string) error {
	return db.Where("id = ?", uuid).Delete(&User{}).Error
}

// UserList gets a list of all users
func UserList(db *gorm.DB) ([]User, error) {
	var records []User
	res := db.Find(&records)
	if res.Error != nil {
		return nil, res.Error
	}
	return records, nil
}

// UserGroupAdd adds a role to a Group
func UserGroupAdd(db *gorm.DB, uuid string, group string) error {
	var user User
	if err := db.First(&user, "id = ?", uuid).Error; err != nil {
		return err
	}

	groupFound := false
	for _, g := range user.Groups {
		if g == group {
			groupFound = true
			break
		}
	}
	if !groupFound {
		user.Groups = append(user.Groups, group)
	}

	return db.Save(&user).Error
}

// UserGroupDelete removes a role from a Group
func UserGroupDelete(db *gorm.DB, uuid string, group string) error {
	var user User
	if err := db.First(&user, "id = ?", uuid).Error; err != nil {
		return err
	}

	for i, g := range user.Groups {
		if g == group {
			user.Groups = append(user.Groups[:i], user.Groups[i+1:]...)
		}
	}

	return db.Save(&user).Error
}

// UserResetPassword resets a User's password
func UserResetPassword(db *gorm.DB, uuid string, password string) error {
	var user User
	if err := db.First(&user, "id = ?", uuid).Error; err != nil {
		return err
	}

	passwordHash, err := auth.Hash(password)
	if err != nil {
		return err
	}
	user.PasswordHash = passwordHash
	return db.Save(&user).Error
}
