package routes

import (
	"errors"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"

	"github.com/packetframe/api/internal/auth"
	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

var (
	ErrInvalidCredentials = errors.New("invalid username and/or password")
	ErrServerAuth         = errors.New("unable to process authentication request")
)

// AuthLogin handles a login POST request
func AuthLogin(c *fiber.Ctx) error {
	var u db.User
	if err := c.BodyParser(&u); err != nil {
		return c.Status(http.StatusUnprocessableEntity).SendString("Invalid request")
	}
	if err := validation.Validate(u); err != nil {
		return c.Status(http.StatusBadRequest).JSON(err)
	}

	user, err := db.UserFind(database, u.Email)
	if err != nil {
		// TODO: Sentry
		return c.Status(http.StatusInternalServerError).SendString(ErrServerAuth.Error())
	}
	if user == nil {
		return c.Status(http.StatusUnprocessableEntity).SendString(ErrInvalidCredentials.Error())
	}

	// Validate password hash
	if !auth.ValidHash(user.PasswordHash, u.Password) {
		return c.Status(http.StatusUnprocessableEntity).SendString(ErrInvalidCredentials.Error())
	}

	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["api_key"] = u.APIKey
	claims["exp"] = time.Now().Add(time.Minute * 15).Unix()
	at := jwt.NewWithClaims(jwt.SigningMethodES512, claims)
	token, err := at.SignedString([]byte("MY_RANDOM_JWT_SECRET"))
	if err != nil {
		// TODO: Sentry
		return c.Status(http.StatusInternalServerError).SendString(ErrServerAuth.Error())
	}
	return c.Status(http.StatusOK).SendString(token)
}
