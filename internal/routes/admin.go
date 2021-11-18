package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/util"
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
