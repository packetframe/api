package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"math/big"

	"golang.org/x/crypto/argon2"
)

var (
	JWTSecret = "TEST_JWT_SECRET" // TODO
)

// RandomString returns a securely generated random string of specified length
func RandomString(length int) (string, error) {
	const letters = "0123456789abcdef"
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}

// argon2IDKey computes an argon2 hash by given input and salt
func argon2IDKey(input []byte, salt []byte) []byte {
	return argon2.IDKey(input, salt, 1, 64*1024, 4, 32)
}

// Hash generates a password hash from plaintext string
func Hash(plaintext string) ([]byte, error) {
	// Generate a random 16-byte salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	hash := argon2IDKey([]byte(plaintext), salt)

	return append(salt, hash...), nil
}

// ValidHash validates a hash and provided plaintext password
func ValidHash(payload []byte, plaintext string) bool {
	salt := payload[:16]
	hash := payload[16:]

	providedHash := argon2IDKey([]byte(plaintext), salt)
	return subtle.ConstantTimeCompare(hash, providedHash) == 1
}
