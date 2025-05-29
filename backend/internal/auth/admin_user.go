package auth

// AdminUser holds the credentials for an admin user.
// For MVP, these are loaded directly from environment variables.
type AdminUser struct {
	Username string
	Password string // Plain text for MVP as per revised instructions
}
