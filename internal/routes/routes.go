package routes

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/miekg/dns"
	"gorm.io/gorm"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/util"
)

// Database stores the global database for the API
var Database *gorm.DB

// routes stores a map of route to handler
var routes = []*route{
	{Path: "/auth/login", Method: http.MethodPost, Handler: AuthLogin, Description: "Log a user in"},
	{Path: "/auth/signup", Method: http.MethodPost, Handler: AuthSignup, Description: "Create a new user account"},
	{Path: "/auth/delete", Method: http.MethodDelete, Handler: UserDelete, Description: "Delete a user account"},
	{Path: "/dns/zones", Method: http.MethodGet, Handler: ZoneList, Description: "List all DNS zones authorized for a user"},
	{Path: "/dns/zones", Method: http.MethodPost, Handler: ZoneAdd, Description: "Add a new DNS zone"},
	{Path: "/dns/zones", Method: http.MethodDelete, Handler: ZoneDelete, Description: "Delete a DNS zone"},
	{Path: "/dns/zones/user", Method: http.MethodPut, Handler: ZoneUserAdd, Description: "Add a user to a DNS zone"},
	{Path: "/dns/zones/user", Method: http.MethodDelete, Handler: ZoneUserDelete, Description: "Remove a user from a DNS zone"},
	{Path: "/dns/records/:id", Method: http.MethodGet, Handler: RecordList, Description: "List DNS records for a zone"},
	{Path: "/dns/records", Method: http.MethodPost, Handler: RecordAdd, Description: "Add a DNS record to a zone"},
	{Path: "/dns/records", Method: http.MethodDelete, Handler: RecordDelete, Description: "Delete a DNS record from a zone"},
	{Path: "/dns/records", Method: http.MethodPut, Handler: RecordUpdate, Description: "Update a DNS record"},
}

type route struct {
	Description string
	Path        string
	Method      string
	Handler     func(c *fiber.Ctx) error
}

// Register registers route handlers
func Register(app *fiber.App) {
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
	return c.Status(status).JSON(fiber.Map{
		"success": (200 <= status) && (status < 300),
		"message": message,
		"data":    data,
	})
}

// internalServerError logs and returns a 503 Internal Server Error
func internalServerError(c *fiber.Ctx, err error) error {
	// TODO: Sentry log err
	fmt.Printf("503 Internal Server Error ---------------------- %s ----------------------\n", err)
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

// checkUserAuthorization checks if a user is authorized for a zone by FQDN
func checkUserAuthorization(c *fiber.Ctx, zoneFqdn string) error {
	// Find zone
	zDb, err := db.ZoneFind(Database, dns.Fqdn(zoneFqdn))
	if err != nil {
		return internalServerError(c, err)
	}
	if zDb == nil {
		return response(c, http.StatusNotFound, "Zone doesn't exist", nil)
	}

	return checkUserAuthorizationByID(c, zDb.ID)
}

// checkUserAuthorizationByID checks if a user is authorized for a zone given a zone ID
func checkUserAuthorizationByID(c *fiber.Ctx, zoneId string) error {
	// Find user
	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	// Check enabled group
	if !util.StrSliceContains(user.Groups, db.GroupEnabled) {
		return response(c, http.StatusForbidden, errUserDisabled, nil)
	}

	// Check if user is authorized for zone
	authorized, err := db.ZoneUserAuthorized(Database, zoneId, user.ID)
	if err != nil {
		return internalServerError(c, err)
	}
	if !authorized {
		return response(c, http.StatusForbidden, "Forbidden", nil)
	}
	return nil
}
