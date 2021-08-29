package routes

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestRoutesAuthLoginInvalidJSON(t *testing.T) {
	database = nil

	app := fiber.New()
	Register(app)

	content := "invalid json"
	req, err := http.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(content))
	assert.Nil(t, err)
	req.Header.Set("Content-Length", strconv.Itoa(len(content)))
	req.Header.Set("Content-Type", "application/json")
	assert.Nil(t, err)

	resp, err := app.Test(req)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
}

func TestRoutesAuthLoginInvalidUserPass(t *testing.T) {
	database = nil

	app := fiber.New()
	Register(app)

	content := `{"username":"invalidemail", "password":"x"}` // Invalid email and password too short
	req, err := http.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(content))
	assert.Nil(t, err)
	req.Header.Set("Content-Length", strconv.Itoa(len(content)))
	req.Header.Set("Content-Type", "application/json")
	assert.Nil(t, err)

	resp, err := app.Test(req)
	assert.Nil(t, err)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
