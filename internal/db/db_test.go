package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDbConnect(t *testing.T) {
	_, err := Connect("host=localhost user=api password=api dbname=api port=5432 sslmode=disable")
	assert.Nil(t, err)
}
