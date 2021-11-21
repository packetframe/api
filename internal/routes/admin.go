package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/util"
	"github.com/packetframe/api/internal/validation"
)

// checkAdminUserAuth checks if a user is an administrator and returns a gofiber response or nil if user is an admin. If it returns true, the user is authorized.
func checkAdminUserAuth(c *fiber.Ctx) (bool, *db.User, error) {
	// Check if user exists
	user, err := findUser(c)
	if err != nil {
		return false, nil, internalServerError(c, err)
	}
	if user == nil {
		return false, nil, response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}
	if !util.StrSliceContains(user.Groups, db.GroupAdmin) {
		return false, nil, response(c, http.StatusUnauthorized, "Unauthorized", nil)
	}

	// If user exists and is admin, return
	return true, user, nil
}

// AdminUserList handles a GET request to list all users
func AdminUserList(c *fiber.Ctx) error {
	ok, _, err := checkAdminUserAuth(c)
	if err != nil || !ok {
		return err
	}

	users, err := db.UserList(Database)
	if err != nil {
		return internalServerError(c, err)
	}

	return response(c, http.StatusOK, "Users retrieved successfully", map[string]interface{}{"users": users})
}

// AdminUserGroupAdd handles a PUT request to add a group to a user
func AdminUserGroupAdd(c *fiber.Ctx) error {
	// Make sure the user is an admin
	ok, _, err := checkAdminUserAuth(c)
	if err != nil || !ok {
		return err
	}

	var r struct {
		UserID string `json:"user"`
		Group  string `json:"group"`
	}
	if err := c.BodyParser(&r); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(r); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Add the group
	if err := db.UserGroupAdd(Database, r.UserID, r.Group); err != nil {
		return response(c, http.StatusBadRequest, err.Error(), nil)
	}

	return response(c, http.StatusOK, "Group added successfully", nil)
}

// AdminUserGroupRemove handles a DELETE request to remove a group from a user
func AdminUserGroupRemove(c *fiber.Ctx) error {
	// Make sure the user is an admin
	ok, _, err := checkAdminUserAuth(c)
	if err != nil || !ok {
		return err
	}

	var r struct {
		UserID string `json:"user"`
		Group  string `json:"group"`
	}
	if err := c.BodyParser(&r); err != nil {
		return response(c, http.StatusUnprocessableEntity, "Invalid request", nil)
	}
	if err := validation.Validate(r); err != nil {
		return response(c, http.StatusBadRequest, "Invalid JSON data", map[string]interface{}{"reason": err})
	}

	// Add the group
	if err := db.UserGroupDelete(Database, r.UserID, r.Group); err != nil {
		return response(c, http.StatusBadRequest, err.Error(), nil)
	}

	return response(c, http.StatusOK, "Group removed successfully", nil)
}
