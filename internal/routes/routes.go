package routes

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// database stores the global database for the API
var database *gorm.DB

// routes stores a map of route to handler
var routes = []*route{
	{Path: "/auth/login", Method: "POST", Handler: AuthLogin},
	{Path: "/auth/signup", Method: "POST", Handler: AuthSignup},
}

type route struct {
	Path    string
	Method  string
	Handler func(c *fiber.Ctx) error
}

// Register registers route handlers
func Register(app *fiber.App) {
	for _, route := range routes {
		switch route.Method {
		case "GET":
			app.Get(route.Path, route.Handler)
		case "POST":
			app.Post(route.Path, route.Handler)
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
	return response(c, http.StatusInternalServerError, "Internal Server Error", nil)
}
