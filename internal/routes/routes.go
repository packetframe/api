package routes

import (
	"fmt"
	"github.com/miekg/dns"
	"github.com/packetframe/api/internal/db"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// Database stores the global database for the API
var Database *gorm.DB

// routes stores a map of route to handler
var routes = []*route{
	{Path: "/auth/login", Method: "POST", Handler: AuthLogin, Description: "Log a user in"},
	{Path: "/auth/signup", Method: "POST", Handler: AuthSignup, Description: "Create a new user account"},
	{Path: "/dns/zones", Method: "GET", Handler: ZoneList, Description: "List all DNS zones authorized for a user"},
	{Path: "/dns/zones", Method: "POST", Handler: ZoneAdd, Description: "Add a new DNS zone"},
	{Path: "/dns/zones", Method: "DELETE", Handler: ZoneDelete, Description: "Delete a DNS zone"},
	{Path: "/dns/zones/user", Method: "PUT", Handler: ZoneUserAdd, Description: "Add a user to a DNS zone"},
	{Path: "/dns/zones/user", Method: "DELETE", Handler: ZoneUserDelete, Description: "Remove a user from a DNS zone"},
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
		case "GET":
			app.Get(route.Path, route.Handler)
		case "POST":
			app.Post(route.Path, route.Handler)
		case "PUT":
			app.Put(route.Path, route.Handler)
		case "DELETE":
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

// checkUserAuthorization checks if a user is authorized for a zone
func checkUserAuthorization(c *fiber.Ctx, zone string) error {
	// Find user
	user, err := findUser(c)
	if err != nil {
		return internalServerError(c, err)
	}
	if user == nil {
		return response(c, http.StatusUnauthorized, "Authentication credentials must be provided", nil)
	}

	// Find zone
	zDb, err := db.ZoneFind(Database, dns.Fqdn(zone))
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
		return response(c, http.StatusForbidden, "Forbidden", nil)
	}
	return nil
}
