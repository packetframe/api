package routes

import (
	"net/http"
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
		httpResp, apiResp, err := testReq(app, http.MethodPost, route.Path, content, map[string]string{})
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
		httpResp, apiResp, err := testReq(app, http.MethodPost, route, content, map[string]string{})
		assert.Nil(t, err)
		assert.Equal(t, http.StatusBadRequest, httpResp.StatusCode)
		assert.False(t, apiResp.Success)
	}
}

func TestRoutesAuthSignupLogin(t *testing.T) {
	var err error
	database, err = db.TestSetup()
	assert.Nil(t, err)

	app := fiber.New()
	Register(app)

	// Sign up user1@example.com
	content := `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err := testReq(app, http.MethodPost, "/auth/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Sign up user1@example.com again to check for conflict validation
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/auth/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusConflict, httpResp.StatusCode)
	assert.False(t, apiResp.Success)

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/auth/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	assert.Equal(t, 64, len(apiResp.Data["token"].(string))) // 64 is the user token length
}
