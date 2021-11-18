package routes

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

func TestRoutesAdminUserList(t *testing.T) {
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

	// Sign up user2@example.com
	content = `{"email":"user2@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Sign up user3@example.com
	content = `{"email":"user3@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	userToken := apiResp.Data["token"].(string)
	assert.Equal(t, 64, len(userToken)) // 64 is the user token length

	// Get user1@example.com's ID
	user1, err := db.UserFindByEmail(Database, "user1@example.com")
	assert.Nil(t, err)

	// Make user1@example.com admin
	err = db.UserGroupAdd(Database, user1.ID, db.GroupAdmin)
	assert.Nil(t, err)

	// Check user1@example.com's info
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/user/info", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	assert.True(t, apiResp.Data["admin"].(bool))

	// Check admin user list
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/admin/user/list", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	respJSON, err := json.Marshal(apiResp.Data["users"])
	assert.Nil(t, err)
	var users []db.User
	err = json.Unmarshal(respJSON, &users)
	assert.Nil(t, err)
	assert.Equal(t, 3, len(users))
}
