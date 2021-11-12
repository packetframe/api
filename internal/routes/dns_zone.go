package routes

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/util"
	"github.com/packetframe/api/internal/validation"
)

var Suffixes []string

// ZoneAdd handles a POST request to add a zone
func ZoneAdd(c *fiber.Ctx) error {
	var z db.Zone
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(z); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	// Suffixes should never be empty because a go routine is updating it
	if len(Suffixes) == 0 {
		return internalServerError(c, errors.New("public suffix list is empty"))
	}

	// Check if the domain is a suffix
	if util.StrSliceContains(Suffixes, strings.TrimSuffix(z.Zone, ".")) {
		return response(c, http.StatusBadRequest, "This zone is a public suffix and requires additional verification. Contact Packetframe for more information.", nil)
	}

	if err := db.ZoneAdd(Database, z.Zone, user.Email); err != nil {
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return response(c, http.StatusConflict, "Zone already exists", nil)
		} else {
			return internalServerError(c, err)
		}
	}

	return response(c, http.StatusOK, "Zone added", nil)
}

// ZoneList handles a GET request to list zones for a user
func ZoneList(c *fiber.Ctx) error {
	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	zones, err := db.ZoneUserGetZones(Database, user.ID)
	if err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Zone added", map[string]interface{}{"zones": zones})
}

// ZoneDelete handles a DELETE request to delete a zone
func ZoneDelete(c *fiber.Ctx) error {
	var z struct {
		ID string `json:"id"`
	}
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorizationByID(c, z.ID); err != nil {
		return err
	}

	deleted, err := db.ZoneDelete(Database, z.ID)
	if !deleted {
		return response(c, http.StatusOK, "Zone doesn't exist, nothing to delete", nil)
	}
	if err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Zone deleted", nil)
}

// ZoneUserAdd handles a PUT request to add a user to a zone
func ZoneUserAdd(c *fiber.Ctx) error {
	var z struct {
		ZoneID    string `json:"zone"`
		UserEmail string `json:"user"`
	}
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorizationByID(c, z.ZoneID); err != nil {
		return err
	}

	if err := db.ZoneUserAdd(Database, z.ZoneID, z.UserEmail); err != nil {
		if errors.Is(err, db.ErrUserExistingZoneMember) {
			return response(c, http.StatusBadRequest, err.Error(), nil)
		} else if errors.Is(err, db.ErrUserNotFound) {
			return response(c, http.StatusBadRequest, err.Error(), nil)
		} else {
			return internalServerError(c, err)
		}
	}

	return response(c, http.StatusOK, "User added to zone", nil)
}

// ZoneUserDelete handles a DELETE request to remove a user from a zone
func ZoneUserDelete(c *fiber.Ctx) error {
	var z struct {
		ZoneID    string `json:"zone"`
		UserEmail string `json:"user"`
	}
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorizationByID(c, z.ZoneID); err != nil {
		return err
	}

	// Delete user
	if err := db.ZoneUserDelete(Database, z.ZoneID, z.UserEmail); err != nil {
		if errors.Is(err, db.ErrUserExistingZoneMember) {
			return response(c, http.StatusBadRequest, err.Error(), nil)
		} else if errors.Is(err, db.ErrUserNotFound) {
			return response(c, http.StatusBadRequest, err.Error(), nil)
		} else {
			return internalServerError(c, err)
		}
	}

	return response(c, http.StatusOK, "User removed from zone", nil)
}
