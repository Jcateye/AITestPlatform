package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// LoginPayload defines the expected JSON structure for login requests.
type LoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Mock session token for MVP. In a real app, use JWT or a secure session store.
const mockSessionToken = "SUPER_SECRET_MVP_TOKEN"
const sessionCookieName = "admin_session_token"

// LoginHandler handles admin login requests.
// It checks credentials against environment-configured values.
// On success, it sets a simple session cookie (for MVP).
func LoginHandler(c *gin.Context) {
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// LoadAdminCredentials() should have been called at application startup.
	// Ensure adminUsername and adminPassword are not empty (loaded from env).
	if adminUsername == "" || adminPassword == "" {
		// This indicates a server configuration issue.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Admin credentials not configured on server"})
		return
	}

	if payload.Username == adminUsername && payload.Password == adminPassword {
		// Set a simple cookie for MVP. Secure to true if using HTTPS.
		// HttpOnly should always be true to prevent XSS.
		// MaxAge is in seconds (e.g., 1 hour).
		// Path set to "/" to be valid for all paths.
		c.SetCookie(sessionCookieName, mockSessionToken, 3600, "/", "", false, true) // Secure=false for local dev without HTTPS
		c.JSON(http.StatusOK, gin.H{
			"message": "Login successful",
			"token":   mockSessionToken, // Also returning as token for flexibility
		})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	}
}

// LogoutHandler placeholder - In a real app, this would invalidate the session/token.
// For MVP with a simple cookie, it can clear the cookie.
func LogoutHandler(c *gin.Context) {
	// Clear the cookie by setting its MaxAge to -1.
	c.SetCookie(sessionCookieName, "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logout successful"})
}
