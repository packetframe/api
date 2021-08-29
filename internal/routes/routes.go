package routes

import (
	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
)

// database stores the global database for the API
var database *gorm.DB

// Register registers route handlers
func Register(app *fiber.App) {
	app.Post("/auth/login", AuthLogin)
}
