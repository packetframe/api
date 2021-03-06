package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/api/validation"
	"github.com/packetframe/api/internal/common/db"
)

func TestRoutesZoneAddListDelete(t *testing.T) {
	err := validation.Register()
	assert.Nil(t, err)

	Database, err = db.TestSetup()
	assert.Nil(t, err)

	app := fiber.New()
	Register(app, map[string]interface{}{"version": "dev"})

	// Populate suffixes slice. This normally happens in a go routine, but this is required for testing
	Suffixes, err = db.SuffixList()
	assert.Nil(t, err)

	// Sign up user1@example.com
	content := `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err := testReq(app, http.MethodPost, "/user/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Enable user1@example.com
	u, err := db.UserFindByEmail(Database, "user1@example.com")
	assert.Nil(t, err)
	err = db.UserGroupAdd(Database, u.ID, db.GroupEnabled)
	assert.Nil(t, err)

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
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

	// Add com (known public suffix)
	httpResp, _, err = testReq(app, http.MethodPost, "/dns/zones", `{"zone":"pages.dev"}`, map[string]string{"Authorization": "Token " + userToken})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusBadRequest, httpResp.StatusCode)

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
	content = fmt.Sprintf(`{"id":"%s"}`, zones[0].ID)
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/dns/zones", content, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// Delete example.com again
	// NOTE: If you try to delete a zone that doesn't exist, you'll get a 403 Forbidden
	content = fmt.Sprintf(`{"id":"%s"}`, zones[0].ID)
	httpResp, _, err = testReq(app, http.MethodDelete, "/dns/zones", content, map[string]string{"Authorization": "Token " + userToken})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusForbidden, httpResp.StatusCode)

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
	err := validation.Register()
	assert.Nil(t, err)

	Database, err = db.TestSetup()
	assert.Nil(t, err)

	app := fiber.New()
	Register(app, map[string]interface{}{"version": "dev"})

	// Populate suffixes slice. This normally happens in a go routine, but this is required for testing
	Suffixes, err = db.SuffixList()
	assert.Nil(t, err)

	// Sign up user1@example.com
	content := `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err := testReq(app, http.MethodPost, "/user/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Enable user1@example.com
	u, err := db.UserFindByEmail(Database, "user1@example.com")
	assert.Nil(t, err)
	err = db.UserGroupAdd(Database, u.ID, db.GroupEnabled)
	assert.Nil(t, err)

	// Sign up user2@example.com
	content = `{"email":"user2@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/signup", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, httpResp.StatusCode)
	assert.True(t, apiResp.Success)

	// Enable user2@example.com
	u, err = db.UserFindByEmail(Database, "user2@example.com")
	assert.Nil(t, err)
	err = db.UserGroupAdd(Database, u.ID, db.GroupEnabled)
	assert.Nil(t, err)

	// Log in user1@example.com
	content = `{"email":"user1@example.com", "password":"example-users-password'"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/user/login", content, map[string]string{})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)
	userToken := apiResp.Data["token"].(string)

	// Add the zone
	content = `{"zone":"example.com"}`
	httpResp, apiResp, err = testReq(app, http.MethodPost, "/dns/zones", content, map[string]string{"Authorization": "Token " + userToken})
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
	assert.Equal(t, 1, len(zones[0].Users))

	// Add user2 to the zone
	content = fmt.Sprintf(`{"zone":"%s", "user": "user2@example.com"}`, zones[0].ID)
	httpResp, apiResp, err = testReq(app, http.MethodPut, "/dns/zones/user", content, map[string]string{"Authorization": "Token " + userToken})
	assert.Nil(t, err)
	assert.Equalf(t, http.StatusOK, httpResp.StatusCode, apiResp.Message)
	assert.Truef(t, apiResp.Success, apiResp.Message)

	// List zones for user to assert that user2@example.com was added
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
	assert.Equal(t, 2, len(zones[0].Users))
	assert.Contains(t, zones[0].UserEmails, "user1@example.com")
	assert.Contains(t, zones[0].UserEmails, "user2@example.com")

	// Remove user2 from zone
	content = fmt.Sprintf(`{"zone":"%s", "user": "user2@example.com"}`, zones[0].ID)
	httpResp, apiResp, err = testReq(app, http.MethodDelete, "/dns/zones/user", content, map[string]string{"Authorization": "Token " + userToken})
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
