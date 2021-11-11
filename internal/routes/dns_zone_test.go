package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/db"
	"github.com/packetframe/api/internal/validation"
)

func TestRoutesZoneAddListDelete(t *testing.T) {
	err := validation.Register()
	assert.Nil(t, err)

	Database, err = db.TestSetup()
	assert.Nil(t, err)

	app := fiber.New()
	Register(app)

	// Sign up user1@example.com
	content := `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err := testReq(app, http.MethodPost, "/auth/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/auth/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	userToken := apiResp.Data["token"].(string)

	// Add an invalid domain (ex^mple.com)
	httpResp, _, err = testReq(app, http.MethodPost, "/dns/zones", `{"zone":"ex^mple.com"}`, map[string]string{"Authorization": "Token " + userToken})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, httpResp.StatusCode)

	// Add example.com
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/dns/zones", `{"zone":"example.com"}`, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// List zones for user
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/dns/zones", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	respJSON, err := json.Marshal(apiResp.Data["zones"])
	assert.Nil(t, err)
	var zones []db.Zone
	err = json.Unmarshal(respJSON, &zones)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(zones))
	assert.Equal(t, "example.com.", zones[0].Zone)

	// Delete example.com
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/dns/zones", fmt.Sprintf(`{"id":"%s"}`, zones[0].ID), map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// Delete example.com again
	httpResp, _, err = testReq(app, http.MethodDelete, "/dns/zones", fmt.Sprintf(`{"id":"%s"}`, zones[0].ID), map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// List zones for user
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/dns/zones", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	respJSON, err = json.Marshal(apiResp.Data["zones"])
	assert.Nil(t, err)
	zones = []db.Zone{}
	err = json.Unmarshal(respJSON, &zones)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(zones))
}

func TestRoutesZoneUserAddDelete(t *testing.T) {
	var err error
	Database, err = db.TestSetup()
	assert.Nil(t, err)

	app := fiber.New()
	Register(app)

	// Sign up user1@example.com
	content := `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err := testReq(app, http.MethodPost, "/auth/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Sign up user2@example.com
	content = `{"email":"user2@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/auth/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/auth/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	userToken := apiResp.Data["token"].(string)

	// Add the zone
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/dns/zones", `{"zone":"example.com"}`, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// Add user2 to the zone
	httpResp, apiResp, err = testReq(app, http.MethodPut, "/dns/zones/user", `{"zone":"example.com", "users": ["user2@example.com"]}`, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// List zones for user
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/dns/zones", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	respJSON, err := json.Marshal(apiResp.Data["zones"])
	assert.Nil(t, err)
	var zones []db.Zone
	err = json.Unmarshal(respJSON, &zones)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(zones))
	assert.Equal(t, 2, len(zones[0].Users))

	// Remove user2 from zone
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/dns/zones/user", `{"zone":"example.com", "users": ["user2@example.com"]}`, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// List zones for user
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/dns/zones", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	respJSON, err = json.Marshal(apiResp.Data["zones"])
	assert.Nil(t, err)
	zones = []db.Zone{}
	err = json.Unmarshal(respJSON, &zones)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(zones))
	assert.Equal(t, 1, len(zones[0].Users))
}
