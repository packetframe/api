package routes

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/common/db"
	"github.com/packetframe/api/internal/common/util"
)

// Database stores the global database for the API
var Database *gorm.DB

var buildMetadata map[string]interface{}

// routes stores a map of route to handler
var routes = []*route{
	{Path: "/meta", Method: http.MethodGet, Handler: meta, Description: "Get API metadata", InvalidJSONTest: false},

	// Authentication
	{Path: "/user/login", Method: http.MethodPost, Handler: UserLogin, Description: "Log a user in", InvalidJSONTest: true},
	{Path: "/user/signup", Method: http.MethodPost, Handler: UserSignup, Description: "Create a new user account", InvalidJSONTest: true},
	{Path: "/user/logout", Method: http.MethodPost, Handler: UserLogout, Description: "Log a user out", InvalidJSONTest: false},

	// User account management
	{Path: "/user/delete", Method: http.MethodDelete, Handler: UserDelete, Description: "Delete a user account", InvalidJSONTest: false},
	{Path: "/user/password", Method: http.MethodPost, Handler: UserChangePassword, Description: "Change a user's password", InvalidJSONTest: true},
	{Path: "/user/info", Method: http.MethodGet, Handler: UserInfo, Description: "Get user info", InvalidJSONTest: false},
	{Path: "/user/request_password_reset", Method: http.MethodPost, Handler: UserRequestPasswordReset, Description: "Request a password reset", InvalidJSONTest: true},
	{Path: "/user/confirm_password_reset", Method: http.MethodPost, Handler: UserConfirmPasswordReset, Description: "Confirm a requested password reset", InvalidJSONTest: true},

	// Zone management
	{Path: "/dns/zones", Method: http.MethodGet, Handler: ZoneList, Description: "List all DNS zones authorized for a user", InvalidJSONTest: false},
	{Path: "/dns/zones", Method: http.MethodPost, Handler: ZoneAdd, Description: "Add a new DNS zone", InvalidJSONTest: true},
	{Path: "/dns/zones", Method: http.MethodDelete, Handler: ZoneDelete, Description: "Delete a DNS zone", InvalidJSONTest: true},
	{Path: "/dns/zones/user", Method: http.MethodPut, Handler: ZoneUserAdd, Description: "Add a user to a DNS zone", InvalidJSONTest: true},
	{Path: "/dns/zones/user", Method: http.MethodDelete, Handler: ZoneUserDelete, Description: "Remove a user from a DNS zone", InvalidJSONTest: true},

	// Record management
	{Path: "/dns/records/:id", Method: http.MethodGet, Handler: RecordList, Description: "List DNS records for a zone", InvalidJSONTest: false},
	{Path: "/dns/records", Method: http.MethodPost, Handler: RecordAdd, Description: "Add a DNS record to a zone", InvalidJSONTest: true},
	{Path: "/dns/records", Method: http.MethodDelete, Handler: RecordDelete, Description: "Delete a DNS record from a zone", InvalidJSONTest: true},
	{Path: "/dns/records", Method: http.MethodPut, Handler: RecordUpdate, Description: "Update a DNS record", InvalidJSONTest: true},

	// Admin
	{Path: "/admin/user/list", Method: http.MethodGet, Handler: AdminUserList, Description: "Get a list of all users", InvalidJSONTest: false},
	{Path: "/admin/user/groups", Method: http.MethodPut, Handler: AdminUserGroupAdd, Description: "Add a group to a user", InvalidJSONTest: false},
	{Path: "/admin/user/groups", Method: http.MethodDelete, Handler: AdminUserGroupRemove, Description: "Remove a group from a user", InvalidJSONTest: false},
	{Path: "/admin/user/impersonate", Method: http.MethodPost, Handler: AdminUserImpersonate, Description: "Log in as another user", InvalidJSONTest: false},

	// Monitor
	{Path: "/admin/status/targets", Method: http.MethodGet, Handler: MonitorTargets, Description: "Get target status", InvalidJSONTest: false},
}

type route struct {
	Description     string
	Path            string
	Method          string
	Handler         func(c *fiber.Ctx) error
	InvalidJSONTest bool
}

// Register registers route handlers
func Register(app *fiber.App, meta map[string]interface{}) {
	buildMetadata = meta

	for _, route := range routes {
		switch route.Method {
		case http.MethodGet:
			app.Get(route.Path, route.Handler)
		case http.MethodPost:
			app.Post(route.Path, route.Handler)
		case http.MethodPut:
			app.Put(route.Path, route.Handler)
		case http.MethodDelete:
			app.Delete(route.Path, route.Handler)
		default:
			panic("invalid HTTP method " + route.Method)
		}
	}
}

// response returns a JSON response
func response(c *fiber.Ctx, status int, message string, data map[string]interface{}) error {
	// Capitalize first letter
	if len(message) > 1 {
		message = strings.ToUpper(message[0:1]) + message[1:]
	}

	return c.Status(status).JSON(fiber.Map{
		"success": (200 <= status) && (status < 300),
		"message": message,
		"data":    data,
	})
}

// internalServerError logs and returns a 503 Internal Server Error
func internalServerError(c *fiber.Ctx, err error) error {
	fmt.Printf("503 Internal Server Error ---------------------- %s ----------------------\n", err)
	sentry.CaptureException(err)
	return response(c, http.StatusInternalServerError, "Internal Server Error", nil)
}

// Document generates a markdown table of API routes
func Document() string {
	table := `| Route | Method | Description |
| :---- | :----- | :---------- |
`
	for _, route := range routes {
		table += fmt.Sprintf("| %s | %s | %s |\n", route.Path, route.Method, route.Description)
	}

	return table
}

// checkUserAuthorizationByID checks if a user is authorized for a zone given a zone ID
func checkUserAuthorizationByID(c *fiber.Ctx, zoneId string) (*db.User, bool, error) {
	// Find user
	user, err := findUser(c)
	if err != nil {
		return nil, false, internalServerError(c, err)
	}
	if user == nil {
		return nil, false, response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	// Check enabled group
	if !util.StrSliceContains(user.Groups, db.GroupEnabled) {
		return user, false, response(c, http.StatusForbidden, errUserDisabled, nil)
	}

	// Allow admins access to all zones
	if util.StrSliceContains(user.Groups, db.GroupAdmin) {
		return user, true, nil
	}

	// Check if user is authorized for zone
	if err := db.ZoneUserAuthorized(Database, zoneId, user.ID); err != nil {
		return user, false, response(c, http.StatusForbidden, "Forbidden", nil)
	}

	return user, true, nil
}

// meta handles a GET request to get API metadata
func meta(c *fiber.Ctx) error {
	return response(c, http.StatusOK, "Metadata retrieved successfully", buildMetadata)
}
