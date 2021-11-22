package auth

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func containsDuplicates(s []string) bool {
	sMap := make(map[string]bool)
	for _, item := range s {
		if _, exist := sMap[item]; exist {
			return true
		}
		sMap[item] = true
	}
	return false
}

func TestAuthRandomString(t *testing.T) {
	rand.Seed(0)
	var randomStrings []string
	for i := 0; i < 128; i++ {
		randomString, err := RandomString(i)
		assert.Nil(t, err)
		randomStrings = append(randomStrings, randomString)

		assert.Len(t, randomString, i)
		assert.False(t, containsDuplicates(randomStrings))
	}
}

func TestAuthHashAndValidHash(t *testing.T) {
	for _, plaintext := range []string{
		"foo",
		"password123",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
	} {
		// Hash
		hashed, err := Hash(plaintext)
		assert.Nil(t, err)

		// Validate
		assert.True(t, ValidHash(hashed, plaintext))
	}
}
