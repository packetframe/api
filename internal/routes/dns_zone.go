package routes

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

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

	if err := db.ZoneAdd(Database, z.Zone, user.ID); err != nil {
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
