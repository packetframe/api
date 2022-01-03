package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/api/auth"
	"github.com/packetframe/api/internal/api/validation"
	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/common/util"
)

var (
	errInvalidCredentials = "invalid username and/or password"
	errUserDisabled       = "this user is disabled, please contact info@packetframe.com"
)

var (
	SMTPHost string
	SMTPUser string
	SMTPPass string
)

// findUser finds a user by Authorization header or cookie and returns nil if a user isn't found
func findUser(c *fiber.Ctx) (*db.User, error) {
	// Get the Authorization header as string and trim the "Token " prefix
	token := strings.TrimPrefix(string(c.Request().Header.Peek("Authorization")), "Token ")

	if token == "" {
		token = c.Cookies("token")
	}

	// If the token is still empty (both header and cookie are undefined), then exit out
	if token == "" {
		return nil, nil
	}

	user, err := db.UserFindByAuth(Database, token)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// UserSignup handles a signup POST request
func UserSignup(c *fiber.Ctx) error {
	var u db.User
	if err := c.BodyParser(&u); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(u); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	user, err := db.UserFindByEmail(Database, u.Email)
	if err != nil {
		return internalServerError(c, err)
	}
	if user != nil { // User already exists
		return response(c, http.StatusConflict, "User already exists", nil)
	}

	if err := db.UserAdd(Database, u.Email, u.Password, u.Refer); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "User created successfully", nil)
}

// UserLogin handles a login POST request
func UserLogin(c *fiber.Ctx) error {
	var u db.User
	if err := c.BodyParser(&u); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(u); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	user, err := db.UserFindByEmail(Database, u.Email)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, errInvalidCredentials, nil)
	}

	// Check enabled group
	if !util.StrSliceContains(user.Groups, db.GroupEnabled) {
		return response(c, http.StatusForbidden, errUserDisabled, nil)
	}

	// Validate password hash
	if !auth.ValidHash(user.PasswordHash, u.Password) {
		return response(c, http.StatusUnauthorized, errInvalidCredentials, nil)
	}

	// Set Token in HTTPONLY cookie
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    user.Token,
		Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		HTTPOnly: true,
	})

	return response(c, http.StatusOK, "Authentication success", fiber.Map{"token": user.Token})
}

// UserLogout handles a GET request to log the user out
func UserLogout(c *fiber.Ctx) error {
	// Known workaround https://github.com/gofiber/fiber/issues/1127
	c.ClearCookie("token")
	c.Cookie(&fiber.Cookie{
		Name:     "token",
		Value:    "",
		Expires:  time.Now(),
		HTTPOnly: true,
	})
	return response(c, http.StatusOK, "Logout success", nil)
}

// UserDelete handles a DELETE request to delete a user
func UserDelete(c *fiber.Ctx) error {
	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	if err := db.UserDelete(Database, user.Email); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "User deleted successfully", nil)
}

// UserChangePassword handles a POST request to change a user's password
func UserChangePassword(c *fiber.Ctx) error {
	var p struct {
		Password string `json:"password" validate:"required,min=8,max=256"`
	}
	if err := c.BodyParser(&p); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(p); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	if err := db.UserResetPassword(Database, user.Email, p.Password); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Password reset successfully", nil)
}

// UserInfo handles a GET request to get a user's info
func UserInfo(c *fiber.Ctx) error {
	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	return response(c, http.StatusOK, "User retrieved successfully", map[string]interface{}{
		"user":  user,
		"admin": util.StrSliceContains(user.Groups, db.GroupAdmin),
	})
}

// UserRequestPasswordReset handles a POST request to request a password reset
func UserRequestPasswordReset(c *fiber.Ctx) error {
	var p struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := c.BodyParser(&p); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(p); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	user, err := db.UserFindByEmail(Database, p.Email)
	if user == nil || err != nil {
		return response(c, http.StatusUnauthorized, "Unable to find user with this email", nil)
	}

	token, err := db.UserCreatePasswordResetToken(Database, user.Email)
	if err != nil {
		return internalServerError(c, err)
	}

	if err := util.SendEmail(SMTPHost, SMTPUser, SMTPPass, user.Email, "Packetframe password reset", fmt.Sprintf(`Hello,

A password reset has been requested for your account. If this wasn't you, you can safely ignore this email.

If you'd like to reset your password, visit https://packetframe.com/dashboard/password_reset/?email=%s&token=%s

Packetframe`, user.Email, token)); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Password reset requested successfully, check your email", nil)
}

// UserConfirmPasswordReset handles a POST request to change a user's password
func UserConfirmPasswordReset(c *fiber.Ctx) error {
	var p struct {
		Email    string `json:"email"`
		Token    string `json:"token"`
		Password string `json:"password" validate:"required,min=8,max=256"`
	}
	if err := c.BodyParser(&p); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(p); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	if err := db.UserValidatePasswordResetToken(Database, p.Email, p.Token); err != nil {
		return response(c, http.StatusForbidden, "Unauthorized: "+err.Error(), nil)
	}

	if err := db.UserResetPassword(Database, p.Email, p.Password); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Password reset successfully", nil)
}
