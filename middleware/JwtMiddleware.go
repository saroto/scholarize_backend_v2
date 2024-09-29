package middleware

import (
	"fmt"
	"net/http"
	"root/database"
	"root/model"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

var invalidTokenMsg = "Token invalid! Please login again!"

// JwtMiddleware checks the token in the request and extracts User info for Gin
func JwtMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := extractToken(c)
		if tokenString == "" {
			fmt.Println("Token not found")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": invalidTokenMsg})
			return
		}

		// Parse the token
		claims, err := parseApiToken(tokenString)
		if err != nil {
			fmt.Println("Token parsing error")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": invalidTokenMsg})
			return
		}

		// Extract user info from claims
		userID, ok := claims["user_id"].(float64)
		if !ok {
			fmt.Println("User ID not found")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": invalidTokenMsg})
			return
		}

		// Validate the token with the database
		if !validateApiTokenWithDatabase(int(userID), tokenString) {
			fmt.Println("Token not found in database")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": invalidTokenMsg})
			return
		}

		userRole, ok := claims["role"].(string)
		if !ok {
			fmt.Println("User role not found")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": invalidTokenMsg})
			return
		}

		// Set user info in Gin context
		c.Set("userID", int(userID))
		c.Set("userRole", userRole)

		// Proceed to next middleware or handler
		c.Next()
	}
}

func extractToken(c *gin.Context) string {
	bearerToken := c.GetHeader("Authorization")
	if len(bearerToken) > 7 && strings.ToUpper(bearerToken[0:7]) == "BEARER " {
		return bearerToken[7:]
	}
	return ""
}

func validateApiTokenWithDatabase(userId int, tokenString string) bool {

	// Check if the token exists in the database
	var token model.Token
	result := database.Db.Where("user_id = ? AND api_token = ?", userId, tokenString).First(&token)
	return result.Error == nil
}

func parseApiToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	secretJwt := viper.GetString("secret_jwt.key")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretJwt), nil
	})
	if err != nil {
		return nil, err
	}
	// Extract the claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, err
	}
	return claims, nil
}
