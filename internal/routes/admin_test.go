package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

func TestRoutesAdminNonAdminUser(t *testing.T) {
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

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	user1Token := apiResp.Data["token"].(string)
	assert.Equal(t, 64, len(user1Token)) // 64 is the user token length

	// Log in user2@example.com
	content = `{"email":"user2@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)
	user2Token := apiResp.Data["token"].(string)
	assert.Equal(t, 64, len(user2Token)) // 64 is the user token length

	// Get user1@example.com's ID
	user1, err := db.UserFindByEmail(Database, "user1@example.com")
	assert.Nil(t, err)

	// Make user1@example.com admin
	err = db.UserGroupAdd(Database, user1.ID, db.GroupAdmin)
	assert.Nil(t, err)

	assert.Greater(t, len(routes), 5)
	for _, route := range routes {
		if strings.Contains(route.Path, "admin") {
			t.Logf("Checking %s", route.Path)

			// Make sure this doesn't throw a 401 for an admins
			httpResp, apiResp, err := testReq(app, route.Method, route.Path, "", map[string]string{"Authorization": "Token " + user1Token})
			assert.Nil(t, err)
			assert.NotEqual(t, http.StatusUnauthorized, httpResp.StatusCode)
			assert.True(t, apiResp.Success)

			// Make sure this throws a 401 for non admins
			httpResp, _, err = testReq(app, route.Method, route.Path, "", map[string]string{"Authorization": "Token " + user2Token})
			assert.NotNil(t, err)
			assert.Equal(t, http.StatusUnauthorized, httpResp.StatusCode)
		}
	}
}

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

func TestRoutesAdminGroupAddRemove(t *testing.T) {
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
	assert.Equal(t, 1, len(users))
	assert.Contains(t, users[0].Groups, db.GroupEnabled)
	assert.Contains(t, users[0].Groups, db.GroupAdmin)

	// Add example group to user
	exampleGroupName := "test.EXAMPLE_GROUP"
	content = fmt.Sprintf(`{"user": "%s", "group": "%s"}`, users[0].ID, exampleGroupName)
	httpResp, apiResp, err = testReq(app, http.MethodPut, "/admin/user/groups", content, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// Check admin user list
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/admin/user/list", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	respJSON, err = json.Marshal(apiResp.Data["users"])
	assert.Nil(t, err)
	users = []db.User{}
	err = json.Unmarshal(respJSON, &users)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	assert.Contains(t, users[0].Groups, db.GroupEnabled)
	assert.Contains(t, users[0].Groups, db.GroupAdmin)
	assert.Contains(t, users[0].Groups, exampleGroupName)

	// Add example group to user
	content = fmt.Sprintf(`{"user": "%s", "group": "%s"}`, users[0].ID, exampleGroupName)
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/admin/user/groups", content, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// Check admin user list
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/admin/user/list", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	respJSON, err = json.Marshal(apiResp.Data["users"])
	assert.Nil(t, err)
	users = []db.User{}
	err = json.Unmarshal(respJSON, &users)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	assert.Contains(t, users[0].Groups, db.GroupEnabled)
	assert.Contains(t, users[0].Groups, db.GroupAdmin)
	assert.NotContains(t, users[0].Groups, exampleGroupName)
}
