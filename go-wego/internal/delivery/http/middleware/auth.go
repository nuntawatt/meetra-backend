package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	internalAuth "github.com/go-wego/wego/internal/auth"
	"github.com/go-wego/wego/pkg/response"
)

// Auth returns a Gin middleware that validates the Bearer JWT access token.
// On success, it stores "user_id" and "role" in the context for downstream handlers.
func Auth(jwtSvc *internalAuth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			response.Unauthorized(c, "invalid authorization format")
			c.Abort()
			return
		}

		claims, err := jwtSvc.ParseAccessToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		// Inject claims into context for handler access
		c.Set("user_id", claims.UserID.String())
		c.Set("role", string(claims.Role))

		c.Next()
	}
}

// RequireRole returns a Gin middleware that allows only specific roles.
// Must be used after the Auth middleware.
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *gin.Context) {
		role := c.GetString("role")
		if _, ok := allowed[role]; !ok {
			response.Forbidden(c, "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}
