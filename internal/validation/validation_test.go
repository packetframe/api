package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/packetframe/api/internal/db"
)

func TestValidate(t *testing.T) {
	u := &db.User{Email: "invalidemail"}
	errors := Validate(u)
	assert.Equal(t, 2, len(errors))

	u = &db.User{Email: "user@example.com", Password: "0123456789"}
	errors = Validate(u)
	assert.Equal(t, 0, len(errors))
}
