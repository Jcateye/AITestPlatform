package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks for the admin session cookie.
// For MVP, it checks for a predefined mock token.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie(sessionCookieName)
		if err != nil {
			// Cookie not found
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Missing session token"})
			c.Abort()
			return
		}

		if cookie == mockSessionToken {
			// Token is valid
			c.Next()
			return
		}

		// Token is invalid
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: Invalid session token"})
		c.Abort()
	}
}
