package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

// RecordAdd handles a POST request to add a DNS record
func RecordAdd(c *fiber.Ctx) error {
	var r db.Record
	if err := c.BodyParser(&r); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(r); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorization(c, r.Zone.Zone); err != nil {
		return err
	}

	// Add the record
	if err := db.RecordAdd(Database, &r); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Record added", nil)
}

// RecordList handles a GET request to list records for a zone
func RecordList(c *fiber.Ctx) error {
	var z struct {
		ZoneID string `json:"zone"`
	}
	if err := c.BodyParser(&z); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(z); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	if err := checkUserAuthorizationByID(c, z.ZoneID); err != nil {
		return err
	}

	// List records for zone
	records, err := db.RecordList(Database, z.ZoneID)
	if err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Zone added", map[string]interface{}{"records": records})
}
