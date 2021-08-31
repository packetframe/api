package routes

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/auth"
	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

var (
	errInvalidCredentials = "invalid username and/or password"
)

// findUser finds a user by Authorization header or JWT cookie and returns nil if a user isn't found
func findUser(c *fiber.Ctx) (*db.User, error) {
	// Get the Authorization header as string and trim the "Token " prefix
	token := strings.TrimPrefix(string(c.Request().Header.Peek("Authorization")), "Token ")
	if token == "" {
		return nil, nil
	}

	user, err := db.UserFindByIdentifier(Database, token)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// AuthSignup handles a signup POST request
func AuthSignup(c *fiber.Ctx) error {
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

	if err := db.UserAdd(Database, u.Email, u.Password); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "User created successfully", nil)
}

// AuthLogin handles a login POST request
func AuthLogin(c *fiber.Ctx) error {
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

	// Validate password hash
	if !auth.ValidHash(user.PasswordHash, u.Password) {
		return response(c, http.StatusUnauthorized, errInvalidCredentials, nil)
	}

	// TODO: Set Token in HTTPONLY cookie
	return response(c, http.StatusOK, "Authentication success", fiber.Map{"token": user.Token})
}
