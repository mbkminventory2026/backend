package passwordutil

import (
	"crypto/rand"
	"fmt"
)

const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%^&*"

func GenerateTemporaryPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate temporary password: %w", err)
	}

	result := make([]byte, length)
	for i, b := range bytes {
		result[i] = alphabet[int(b)%len(alphabet)]
	}

	return string(result), nil
}
