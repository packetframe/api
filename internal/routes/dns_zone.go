package routes

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/miekg/dns"

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
	var z db.Zone
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(z); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorization(c, z.Zone); err != nil {
		return err
	}

	// Find zone
	zDb, err := db.ZoneFind(Database, dns.Fqdn(z.Zone))
	if err != nil {
		return internalServerError(c, err)
	}
	if zDb == nil {
		return response(c, http.StatusNotFound, "Zone doesn't exist", nil)
	}

	if err := db.ZoneDelete(Database, zDb.ID); err != nil {
		// TODO: zone already deleted?
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Zone added", nil)
}

// ZoneUserAdd handles a PUT request to add a user to a zone
func ZoneUserAdd(c *fiber.Ctx) error {
	var z db.Zone
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(z); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorization(c, z.Zone); err != nil {
		return err
	}

	// Find zone
	zDb, err := db.ZoneFind(Database, dns.Fqdn(z.Zone))
	if err != nil {
		return internalServerError(c, err)
	}
	if zDb == nil {
		return response(c, http.StatusNotFound, "Zone doesn't exist", nil)
	}

	for _, user := range z.Users {
		if err := db.ZoneUserAdd(Database, z.Zone, user); err != nil {
			if errors.Is(err, db.ErrUserExistingZoneMember) {
				return response(c, http.StatusBadRequest, err.Error(), nil)
			} else if errors.Is(err, db.ErrUserNotFound) {
				return response(c, http.StatusBadRequest, err.Error(), nil)
			} else {
				return internalServerError(c, err)
			}
		}
	}

	return response(c, http.StatusOK, "User added to zone", nil)
}

// ZoneUserDelete handles a DELETE request to remove a user from a zone
func ZoneUserDelete(c *fiber.Ctx) error {
	var z db.Zone
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(z); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorization(c, z.Zone); err != nil {
		return err
	}

	// Find zone
	zDb, err := db.ZoneFind(Database, dns.Fqdn(z.Zone))
	if err != nil {
		return internalServerError(c, err)
	}
	if zDb == nil {
		return response(c, http.StatusNotFound, "Zone doesn't exist", nil)
	}

	for _, user := range z.Users {
		uDoc, err := db.UserFindByEmail(Database, user)
		if err != nil {
			return response(c, http.StatusNotFound, "User doesn't exist", nil)
		}
		if err := db.ZoneUserDelete(Database, zDb.ID, uDoc.ID); err != nil {
			if errors.Is(err, db.ErrUserExistingZoneMember) {
				return response(c, http.StatusBadRequest, err.Error(), nil)
			} else if errors.Is(err, db.ErrUserNotFound) {
				return response(c, http.StatusBadRequest, err.Error(), nil)
			} else {
				return internalServerError(c, err)
			}
		}
	}

	return response(c, http.StatusOK, "User removed from zone", nil)
}
