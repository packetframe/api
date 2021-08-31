package routes

import (
	"encoding/json"
	"fmt"
	"github.com/packetframe/api/internal/db"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

type apiResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// testReq makes a test HTTP request
func testReq(app *fiber.App, method string, path string, jsonContent string, headers map[string]string) (*http.Response, *apiResponse, error) {
	req, err := http.NewRequest(method, path, strings.NewReader(jsonContent))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Length", strconv.Itoa(len(jsonContent)))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := app.Test(req)
	if err != nil {
		return nil, nil, err
	}
	if (resp.StatusCode > 200) || (resp.StatusCode < 200) {
		return resp, nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var apiResp apiResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, nil, err
	}

	return resp, &apiResp, nil
}

func TestRoutes404(t *testing.T) {
	var err error
	Database, err = db.TestSetup()
	assert.Nil(t, err)
	app := fiber.New()
	Register(app)
	httpResp, _, err := testReq(app, http.MethodGet, "/non-existent-path", "", map[string]string{})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusNotFound, httpResp.StatusCode)
}
