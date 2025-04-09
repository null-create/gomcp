package auth

import (
	"log"
	"os"
)

func GetCwd() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working directory: %v", err)
	}
	return dir
}
