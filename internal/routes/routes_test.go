package routes

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type apiResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

// testReq makes a test HTTP request
func testReq(app *fiber.App, method string, path string, jsonContent string) (*http.Response, *apiResponse, error) {
	req, err := http.NewRequest(method, path, strings.NewReader(jsonContent))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Length", strconv.Itoa(len(jsonContent)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		return nil, nil, err
	}

	var apiResp apiResponse
	err = json.NewDecoder(resp.Body).Decode(&apiResp)
	if err != nil {
		return nil, nil, err
	}

	return resp, &apiResp, nil
}
