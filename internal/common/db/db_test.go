package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbConnect(t *testing.T) {
	_, err := TestSetup()
	assert.Nil(t, err)
}
