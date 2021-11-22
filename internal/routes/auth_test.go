package routes

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

func TestRoutesAuthInvalidUserPass(t *testing.T) {
	Database = nil

	app := fiber.New()
	Register(app)
	err := validation.Register()
	assert.Nil(t, err)

	content := `{"email":"invalidemail", "password":"x"}` // Invalid email and password too short
	for _, route := range []string{"/user/signup", "/user/login"} {
		httpResp, _, err := testReq(app, http.MethodPost, route, content, map[string]string{})
		assert.NotNilf(t, err, route)
		assert.Equalf(t, http.StatusBadRequest, httpResp.StatusCode, route)
	}
}

func TestRoutesUserSignupLoginDelete(t *testing.T) {
	var err error
	Database, err = db.TestSetup()
	assert.Nil(t, err)

	app := fiber.New()
	Register(app)

	err = validation.Register()
	assert.Nil(t, err)

	// Sign up user1@example.com
	content := `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err := testReq(app, http.MethodPost, "/user/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Sign up user1@example.com again to check for conflict validation
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, _, err = testReq(app, http.MethodPost, "/user/signup", content, map[string]string{})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusConflict, httpResp.StatusCode)

	// Enable user1@example.com
	u, err := db.UserFindByEmail(Database, "user1@example.com")
	assert.Nil(t, err)
	err = db.UserGroupAdd(Database, u.ID, db.GroupEnabled)
	assert.Nil(t, err)

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	userToken := apiResp.Data["token"].(string)
	assert.Equal(t, 64, len(userToken)) // 64 is the user token length

	// Check user1@example.com's info
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/user/info", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	assert.False(t, apiResp.Data["admin"].(bool))

	// Change user1@example.com's password
	content = `{"password":"example-users-NEW-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/password", content, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Log in user1@example.com with the new password
	content = `{"email":"user1@example.com", "password":"example-users-NEW-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	userToken = apiResp.Data["token"].(string)
	assert.Equal(t, 64, len(userToken)) // 64 is the user token length

	// Delete user1@example.com
	content = `{"email":"user1@example.com"}`
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/user/delete", content, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Log in user1@example.com to make sure it's been deleted
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, _, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusUnauthorized, httpResp.StatusCode)
}
