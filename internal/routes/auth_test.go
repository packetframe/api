package routes

import (
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/db"
)

func TestRoutesInvalidJSON(t *testing.T) {
	database = nil

	app := fiber.New()
	Register(app)

	content := "invalid json"
	for _, route := range routes {
		httpResp, apiResp, err := testReq(app, http.MethodPost, route.Path, content)
		assert.Nilf(t, err, route.Path)
		assert.Equalf(t, http.StatusUnprocessableEntity, httpResp.StatusCode, route.Path)
		assert.Falsef(t, apiResp.Success, route.Path)
	}
}

func TestRoutesAuthInvalidUserPass(t *testing.T) {
	database = nil

	app := fiber.New()
	Register(app)

	content := `{"email":"invalidemail", "password":"x"}` // Invalid email and password too short
	for _, route := range []string{"/auth/signup", "/auth/login"} {
		httpResp, apiResp, err := testReq(app, http.MethodPost, route, content)
		assert.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, httpResp.StatusCode)
		assert.False(t, apiResp.Success)
	}
}

func TestRoutesAuthSignupLogin(t *testing.T) {
	var err error
	database, err = db.Connect(os.Getenv("PACKETFRAME_API_TEST_DB"))
	assert.Nil(t, err)

	err = database.Exec("DELETE FROM users").Error
	assert.Nil(t, err)

	app := fiber.New()
	Register(app)

	content := `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err := testReq(app, http.MethodPost, "/auth/signup", content)
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
}
