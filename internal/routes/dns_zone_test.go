package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/db"
)

func TestRoutesZoneAddListDelete(t *testing.T) {
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
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/dns/zones", `{"zone":"example.com"}`, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// Delete example.com again
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/dns/zones", `{"zone":"example.com"}`, map[string]string{"Authorization": "Token " + userToken})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusNotFound, httpResp.StatusCode)

	// List zones for user
	httpResp, apiResp, err = testReq(app, http.MethodGet, "/dns/zones", "", map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	fmt.Println(apiResp.Data)
	respJSON, err = json.Marshal(apiResp.Data["zones"])
	assert.Nil(t, err)
	zones = []db.Zone{}
	err = json.Unmarshal(respJSON, &zones)
	assert.Nil(t, err)
	assert.Equal(t, 0, len(zones))
}
