package auth

import (
	"log"
	"os"
)

var adminUsername string
var adminPassword string // Plain text for MVP

// LoadAdminCredentials loads the admin username and password from environment variables.
// It logs a warning if they are not set.
func LoadAdminCredentials() {
	adminUsername = os.Getenv("ADMIN_USERNAME")
	adminPassword = os.Getenv("ADMIN_PASSWORD")

	if adminUsername == "" {
		log.Println("WARNING: ADMIN_USERNAME environment variable not set.")
	}
	if adminPassword == "" {
		log.Println("WARNING: ADMIN_PASSWORD environment variable not set.")
	}
}
