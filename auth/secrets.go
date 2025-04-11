package auth

import (
	"os"
)

// retrieve users JWT secret used for signing tokens
func GetSecret() []byte {
	secret := os.Getenv("GOMCP_SECRET")
	return []byte(secret)
}
