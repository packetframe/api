package db

import (
	"errors"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/auth"
)

var (
	GroupEnabled = "ENABLED" // User is permitted to make API requests
	GroupAdmin   = "ADMIN"   // User is permitted to modify all resources

	ErrUserOwnsZones = errors.New("user has zones without other users, delete or add another user to these zones before deleting this user account")
)

type User struct {
	ID           string         `gorm:"primaryKey,type:uuid;default:uuid_generate_v4()" json:"id"`
	Email        string         `gorm:"uniqueIndex" json:"email" validate:"required,email,min=6,max=32"`
	Password     string         `gorm:"-" json:"password" validate:"required,min=8,max=256"`
	Refer        string         `json:"refer"` // Where did you hear about Packetframe?
	Groups       pq.StringArray `gorm:"type:text[]" json:"groups"`
	PasswordHash []byte         `json:"-"`
	APIKey       string         `json:"-"` // Rotated manually by user if needed
	Token        string         `json:"-"` // Rotated every n minutes (TODO: autorotate this)
	CreatedAt    time.Time      `json:"-"`
	UpdatedAt    time.Time      `json:"-"`
}

// UserAdd creates a new user
func UserAdd(db *gorm.DB, email string, password string, refer string) error {
	passwordHash, err := auth.Hash(password)
	if err != nil {
		return err
	}
	apiKey, err := auth.RandomString(48)
	if err != nil {
		return err
	}
	token, err := auth.RandomString(64)
	if err != nil {
		return err
	}
	return db.Create(&User{
		Email:        email,
		PasswordHash: passwordHash,
		APIKey:       apiKey,
		Token:        token,
		Groups:       []string{GroupEnabled},
		Refer:        refer,
	}).Error
}

// UserFindByEmail finds a user by email and returns nil if no user exists
func UserFindByEmail(db *gorm.DB, email string) (*User, error) {
	var user User
	res := db.First(&user, "email = ?", email)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if res.Error != nil {
		return nil, res.Error
	}

	return &user, nil
}

// UserFindByAuth finds a user by API key and returns nil if no user exists
func UserFindByAuth(db *gorm.DB, id string) (*User, error) {
	var user User
	res := db.Where("api_key = ?", id).Or("token = ?", id).First(&user)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if res.Error != nil {
		return nil, res.Error
	}

	return &user, nil
}

// UserDelete deletes a user
func UserDelete(db *gorm.DB, email string) error {
	// Find user ID
	user, err := UserFindByEmail(db, email)
	if err != nil {
		return err
	}

	// Check if user is the only user in any zones
	var zones []Zone
	tx := db.Find(&zones, "? = ANY (users) AND array_length(users, 1) = 1", user.ID)
	if tx.Error != nil {
		return tx.Error
	}
	if len(zones) > 0 {
		return ErrUserOwnsZones
	}

	// Remove user from zones
	zones = []Zone{}
	tx = db.Find(&zones, "? = ANY (users)", user.ID)
	if tx.Error != nil {
		return tx.Error
	}
	for _, zone := range zones {
		if err := ZoneUserDelete(db, zone.ID, user.Email); err != nil {
			return err
		}
	}

	return db.Where("id = ?", user.ID).Delete(&User{}).Error
}

// UserList gets a list of all users
func UserList(db *gorm.DB) ([]User, error) {
	var users []User
	res := db.Find(&users)
	if res.Error != nil {
		return nil, res.Error
	}
	return users, nil
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
func UserResetPassword(db *gorm.DB, email string, password string) error {
	var user User
	if err := db.First(&user, "email = ?", email).Error; err != nil {
		return err
	}

	// Hash new password
	passwordHash, err := auth.Hash(password)
	if err != nil {
		return err
	}
	user.PasswordHash = passwordHash

	// Generate new token to invalidate all new API requests from old logins
	token, err := auth.RandomString(64)
	if err != nil {
		return err
	}
	user.Token = token

	return db.Save(&user).Error
}
