package auth

import (
	"log"

	"github.com/google/uuid"
)

// creates a new UUID string
func NewUUID() string {
	uuid, err := uuid.NewUUID()
	if err != nil {
		log.Fatalf("[ERROR] failed to generate UUID: \n%v\n", err)
	}
	return uuid.String()
}
