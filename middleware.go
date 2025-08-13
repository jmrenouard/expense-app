package main

import (
    "database/sql"
    "net/http"
    "strings"

    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

const ContextUserIDKey = "userID"

// AuthMiddleware checks for an Authorization header containing a valid JWT and
// adds the user ID to the request context. It also supports static API tokens
// via the X-API-Key header (not implemented fully; reserved for future use).
func AuthMiddleware(db *sql.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Check X-API-Key for static token support (placeholder)
        apiKey := c.GetHeader("X-API-Key")
        if apiKey != "" {
            // In a complete implementation, we would verify apiKey against a database table.
            // For now, we reject with unauthorized.
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API keys not supported"})
            return
        }
        // Parse JWT from Authorization header
        auth := c.GetHeader("Authorization")
        if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid Authorization header"})
            return
        }
        tokenString := strings.TrimSpace(auth[len("Bearer "):])
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrTokenMalformed
            }
            return []byte(jwtSecret), nil
        })
        if err != nil || !token.Valid {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
            return
        }
        claims, ok := token.Claims.(jwt.MapClaims)
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
            return
        }
        sub, ok := claims["sub"].(float64)
        if !ok {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
            return
        }
        // Set user ID in context
        c.Set(ContextUserIDKey, int64(sub))
        c.Next()
    }
}

// RequirePermission ensures that the authenticated user possesses a specific permission.
// When a user lacks the permission, a 403 Forbidden response is returned.
func RequirePermission(db *sql.DB, permission string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userIDIfc, exists := c.Get(ContextUserIDKey)
        if !exists {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
            return
        }
        userID, _ := userIDIfc.(int64)
        has, err := UserHasPermission(db, userID, permission)
        if err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to check permissions"})
            return
        }
        if !has {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
            return
        }
        c.Next()
    }
}
