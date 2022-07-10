package routes

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/api/validation"
	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/common/util"
)

func checkProxyRecord(r db.Record, ctx *fiber.Ctx, userGroups []string) error {
	if !util.StrSliceContains(userGroups, db.GroupAdmin) {
		return response(ctx, http.StatusForbidden, "You are not authorized to create proxied records. Please contact Packetframe support.", nil)
	}
	if strings.Contains(r.Label, "*") {
		return response(ctx, http.StatusBadRequest, "Proxied records cannot contain wildcards", nil)
	}
	if !(r.Type == "A" || r.Type == "AAAA") {
		return response(ctx, http.StatusBadRequest, "Proxied records must be A or AAAA", nil)
	}
	return nil
}

// RecordAdd handles a POST request to add a DNS record
func RecordAdd(c *fiber.Ctx) error {
	var r db.Record
	if err := c.BodyParser(&r); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	r.ID = ""
	if err := validation.Validate(r); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	user, ok, err := checkUserAuthorizationByID(c, r.ZoneID)
	if err != nil || !ok {
		return err
	}

	if r.Proxy {
		if err := checkProxyRecord(r, c, user.Groups); err != nil {
			return err
		}
	}

	// Validate if SCRIPT record
	if r.Type == "SCRIPT" {
		if err := db.ScriptValidate(r.Value, r.Label); err != nil {
			return response(c, http.StatusBadRequest, "Error compiling DNS script: "+err.Error(), nil)
		}
	}

	// Add the record
	if err := db.RecordAdd(Database, &r); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Record added", nil)
}

// RecordList handles a GET request to list records for a zone
func RecordList(c *fiber.Ctx) error {
	zoneID := c.Params("id")

	// Check if user is authorized for zone
	if _, ok, err := checkUserAuthorizationByID(c, zoneID); err != nil || !ok {
		return err
	}

	// List records for zone
	records, err := db.RecordList(Database, zoneID)
	if err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Zone added", map[string]interface{}{"records": records})
}

// RecordDelete handles a DELETE request to delete a DNS record
func RecordDelete(c *fiber.Ctx) error {
	var r struct {
		ZoneID   string `json:"zone"`
		RecordID string `json:"record"`
	}
	if err := c.BodyParser(&r); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(r); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	if _, ok, err := checkUserAuthorizationByID(c, r.ZoneID); err != nil || !ok {
		return err
	}

	// Delete the record
	deleted, err := db.RecordDelete(Database, r.RecordID)
	if err != nil {
		return internalServerError(c, err)
	}
	if !deleted {
		return response(c, http.StatusOK, "Record doesn't exist, nothing to delete", nil)
	}

	return response(c, http.StatusOK, "Record deleted", nil)
}

// RecordUpdate handles a PUT request to update a DNS record
func RecordUpdate(c *fiber.Ctx) error {
	var r db.Record
	if err := c.BodyParser(&r); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(r); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Check if user is authorized for zone
	user, ok, err := checkUserAuthorizationByID(c, r.ZoneID)
	if err != nil || !ok {
		return err
	}

	if r.Proxy {
		if err := checkProxyRecord(r, c, user.Groups); err != nil {
			return err
		}
	}

	// Validate if SCRIPT record
	if r.Type == "SCRIPT" {
		if err := db.ScriptValidate(r.Value, r.Label); err != nil {
			return response(c, http.StatusBadRequest, "Error compiling DNS script: "+err.Error(), nil)
		}
	}

	// Update the record
	// TODO: This doesn't update the proxy field
	if err := db.RecordUpdate(Database, &r); err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Record updated", nil)
}
