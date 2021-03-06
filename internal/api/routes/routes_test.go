package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/common/db"
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
		var respBytes []byte
		respBytes, _ = io.ReadAll(resp.Body)
		return resp, nil, fmt.Errorf("unexpected status code %d %s", resp.StatusCode, string(respBytes))
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
	Register(app, map[string]interface{}{"version": "dev"})
	httpResp, _, err := testReq(app, http.MethodGet, "/non-existent-path", "", map[string]string{})
	assert.NotNil(t, err)
	assert.Equal(t, http.StatusNotFound, httpResp.StatusCode)
}

func TestDocument(t *testing.T) {
	doc := Document()
	assert.Greater(t, len(doc), 100)
}

func TestRoutesInvalidJSON(t *testing.T) {
	Database = nil

	app := fiber.New()
	Register(app, map[string]interface{}{"version": "dev"})

	content := "invalid json"
	for _, route := range routes {
		if route.InvalidJSONTest {
			httpResp, _, err := testReq(app, route.Method, route.Path, content, map[string]string{})
			assert.NotNilf(t, err, route.Path)
			assert.Equalf(t, http.StatusUnprocessableEntity, httpResp.StatusCode, route.Method+" "+route.Path)
		}
	}
}
