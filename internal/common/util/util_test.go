package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStrSliceContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	assert.True(t, StrSliceContains(slice, "a"))
	assert.False(t, StrSliceContains(slice, "d"))
}
