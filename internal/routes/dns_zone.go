package routes

import (
	"fmt"
	"github.com/miekg/dns"
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

// ZoneDelete handles a DELETE request to delete a zone
func ZoneDelete(c *fiber.Ctx) error {
	var z db.Zone
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(z); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Find user
	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	// Find zone
	zDb, err := db.ZoneFind(Database, dns.Fqdn(z.Zone))
	if err != nil {
		return internalServerError(c, err)
	}
	if zDb == nil {
		return response(c, http.StatusNotFound, "Zone doesn't exist", nil)
	}

	// Check if user is authorized for zone
	authorized, err := db.ZoneUserAuthorized(Database, zDb.ID, user.ID)
	if err != nil {
		return internalServerError(c, err)
	}
	if !authorized {
		fmt.Printf("user id %s zone users %+v", user.ID, z.Users)
		return response(c, http.StatusForbidden, "Forbidden", nil)
	}

	fmt.Println("deleting " + zDb.ID)
	if err := db.ZoneDelete(Database, zDb.ID); err != nil {
		// TODO: zone already deleted?
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Zone added", nil)
}
