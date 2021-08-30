package routes

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"

	"github.com/packetframe/api/internal/auth"
	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

var (
	errInvalidCredentials = "invalid username and/or password"
)

// AuthSignup handles a signup POST request
func AuthSignup(c *fiber.Ctx) error {
	var u db.User
	if err := c.BodyParser(&u); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(u); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	user, err := db.UserFind(database, u.Email)
	if err != nil {
		return internalServerError(c, err)
	}
	if user != nil { // User already exists
		return response(c, http.StatusConflict, "User already exists", nil)
	}

	if err := db.UserAdd(database, u.Email, u.Password); err != nil {
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

	user, err := db.UserFind(database, u.Email)
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

	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["api_key"] = u.APIKey
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodES512, claims)
	token, err := at.SignedString([]byte("MY_RANDOM_JWT_SECRET"))
	if err != nil {
		return internalServerError(c, err)
	}
	return response(c, http.StatusOK, "Authentication succeeded", fiber.Map{"token": token})
}
