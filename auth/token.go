package auth

import (
	"fmt"
	"root/database"
	"root/model"
	"root/permission"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// GenerateToken generates a new JWT token
func GenerateApiToken(userEmail string) (string, error) {
	secretJwt := viper.GetString("secret_jwt.key")
	// Get UserID from userEmail
	userID, err := permission.GetUserId(userEmail)
	if err != nil {
		return "", err
	}

	userRole := permission.GetFrontPanelUserRoleName(userID)

	// Create the Claims
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = userID
	claims["role"] = userRole
	claims["exp"] = time.Now().Add(time.Hour * 24 * 7).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretJwt))
	if err != nil {
		return "", err
	}

	// Create a new Token instance
	newToken := model.Token{
		UserID:      int(userID),
		ApiToken:    tokenString,
		TokenExpire: time.Now().Add(time.Hour * 24 * 7),
	}

	// Insert the new token record into the database
	result := database.Db.Create(&newToken)
	if result.Error != nil {
		return "", result.Error
	}

	return tokenString, nil
}

// UpdateToken updates an existing JWT token
func UpdateApiToken(userEmail string) (string, error) {
	// Get UserID from userEmail
	userID, err := permission.GetUserId(userEmail)
	if err != nil {
		return "", err
	}

	userRole := permission.GetFrontPanelUserRoleName(userID)

	// Create token string
	var tokenString string

	// Check if the token is not found
	var tokenModel model.Token
	if result := database.Db.Where("user_id = ?", userID).First(&tokenModel); result.Error != nil {
		fmt.Printf("Error when trying to login: %s\n", result.Error.Error())
		if result.Error == gorm.ErrRecordNotFound {
			// Generate a new token
			fmt.Printf("Generating a new token for user %s\n", userEmail)
			resultToken, err := GenerateApiToken(userEmail)
			if err != nil {
				return "", err
			}
			return resultToken, nil
		}
		return "", result.Error
	}

	// Check if token valid
	if tokenModel.ApiToken != "cleared" {
		// Check if the token is not expired
		if tokenModel.TokenExpire.Format("2006-01-02 15:04:05") > time.Now().Format("2006-01-02 15:04:05") {
			// Parse token to see user role
			claims, err := ParseApiToken(tokenModel.ApiToken)
			if err != nil {
				return "", err
			}

			// Extract the user role
			role := claims["role"].(string)
			fmt.Printf("Role when trying to login: %s\n", role)

			// Check if the user role is the same
			if role == userRole {
				fmt.Printf("Role when trying to login: %s is the same as %s\n", role, userRole)
				tokenString = tokenModel.ApiToken
			} else {
				fmt.Printf("Role when trying to login: %s is not the same as %s\n", role, userRole)
				resultToken, err := createNewToken(userID, userRole)
				if err != nil {
					return "", err
				}
				// Update the token record in the database
				newToken := model.Token{
					UserID:      int(userID),
					ApiToken:    resultToken,
					TokenExpire: time.Now().Add(time.Hour * 24 * 7),
				}

				// Update the token record in the database
				result := database.Db.Where("user_id = ?", userID).Updates(&newToken)
				if result.Error != nil {
					return "", result.Error
				}

				tokenString = newToken.ApiToken
			}
		} else {
			// If the token is expired
			resultToken, err := createNewToken(userID, userRole)
			if err != nil {
				return "", err
			}
			// Update the token record in the database
			newToken := model.Token{
				UserID:      int(userID),
				ApiToken:    resultToken,
				TokenExpire: time.Now().Add(time.Hour * 24 * 7),
			}

			// Update the token record in the database
			result := database.Db.Where("user_id = ?", userID).Updates(&newToken)
			if result.Error != nil {
				return "", result.Error
			}

			tokenString = newToken.ApiToken
		}
	} else {
		resultToken, err := createNewToken(userID, userRole)
		if err != nil {
			return "", err
		}
		// Update the token record in the database
		newToken := model.Token{
			UserID:      int(userID),
			ApiToken:    resultToken,
			TokenExpire: time.Now().Add(time.Hour * 24 * 7),
		}

		// Update the token record in the database
		result := database.Db.Where("user_id = ?", userID).Updates(&newToken)
		if result.Error != nil {
			return "", result.Error
		}

		tokenString = newToken.ApiToken
	}

	return tokenString, nil
}

func createNewToken(userId int, userRole string) (string, error) {
	secretJwt := viper.GetString("secret_jwt.key")

	// Create the Claims
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = userId
	claims["role"] = userRole
	claims["exp"] = time.Now().Add(time.Hour * 24 * 7).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretJwt))
	if err != nil {
		return "", err
	}

	return tokenString, err
}

func GenerateAdminApiToken(userEmail string) (string, error) {
	secretJwt := viper.GetString("secret_jwt.key")
	// Get UserID from userEmail
	userID, err := permission.GetUserId(userEmail)
	if err != nil {
		return "", err
	}

	userRole := permission.GetAdminPanelUserRoleName(userID)

	// Create the Claims
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = userID
	claims["role"] = userRole
	claims["boarded"] = false
	claims["exp"] = time.Now().Add(time.Hour * 24 * 7).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretJwt))
	if err != nil {
		return "", err
	}

	// Create a new Token instance
	newToken := model.Token{
		UserID:      int(userID),
		ApiToken:    tokenString,
		TokenExpire: time.Now().Add(time.Hour * 24 * 7),
	}

	// Insert the new token record into the database
	result := database.Db.Create(&newToken)
	if result.Error != nil {
		return "", result.Error
	}

	return tokenString, nil
}

func UpdateAdminApiToken(userEmail string) (string, error) {
	// Get UserID from userEmail
	userID, err := permission.GetUserId(userEmail)
	if err != nil {
		return "", err
	}

	// Get the admin reset password record for boarding status
	var adminResetPassword model.AdminResetPassword
	if result := database.Db.Where("user_id = ?", userID).First(&adminResetPassword); result.Error != nil {
		return "", result.Error
	}

	userRole := permission.GetAdminPanelUserRoleName(userID)

	// No need to check
	// var tokenModel model.Token
	// if result := database.Db.Where("user_id = ?", userID).First(&tokenModel); result.Error != nil {
	// 	return "", result.Error
	// }

	// if tokenModel.ApiToken != "cleared" {
	// 	if tokenModel.TokenExpire.Format("2006-01-02 15:04:05") > time.Now().Format("2006-01-02 15:04:05") {
	// 		return tokenModel.ApiToken, nil
	// 	}
	// }

	// Create the Claims
	secretJwt := viper.GetString("secret_jwt.key")
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = userID
	claims["role"] = userRole
	claims["boarded"] = adminResetPassword.IsBoarded
	claims["exp"] = time.Now().Add(time.Hour * 24 * 7).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretJwt))
	if err != nil {
		return "", err
	}

	// Create a new Token instance
	newToken := model.Token{
		UserID:      int(userID),
		ApiToken:    tokenString,
		TokenExpire: time.Now().Add(time.Hour * 24 * 7),
	}

	// Update the token record in the database
	result := database.Db.Where("user_id = ?", userID).Updates(&newToken)
	if result.Error != nil {
		return "", result.Error
	}

	return tokenString, nil
}

// ParseToken parses a JWT token
func ParseApiToken(tokenString string) (jwt.MapClaims, error) {
	secretJwt := viper.GetString("secret_jwt.key")
	// Parse the token
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

// Clears deletes a JWT token
func ClearApiToken(tokenString string) error {
	// Parse the token
	claims, err := ParseApiToken(tokenString)
	if err != nil {
		return err
	}
	// Extract the user ID
	userID := claims["user_id"].(float64)

	// Clear the token
	newToken := model.Token{
		ApiToken:    "cleared",
		TokenExpire: time.Now(),
	}

	// Update the token in the database
	result := database.Db.Model(&model.Token{}).Where("user_id = ?", userID).Updates(newToken)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Check token with database
func ValidateApiTokenWithDatabase(tokenString string) bool {
	claims, err := ParseApiToken(tokenString)
	if err != nil {
		return false
	}

	// Extract the user ID
	userId := claims["user_id"].(float64)

	// Check if the token exists in the database
	var token model.Token
	if result := database.Db.Where("user_id = ? AND api_token = ?", userId, tokenString).First(&token); result.Error != nil {
		return false
	}
	return true
}

// Generate Reset Password Token
func GenerateResetPasswordToken(tx *gorm.DB, userId int) (string, error) {
	secretResetToken := viper.GetString("secret_reset_token.key")
	// Create the Claims
	claims := jwt.MapClaims{}
	claims["authorized"] = true
	claims["user_id"] = userId
	claims["exp"] = time.Now().Add(time.Minute * 10).Unix()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secretResetToken))
	if err != nil {
		return "", err
	}

	// Create a new Token instance
	newToken := model.AdminResetPassword{
		ResetToken:       tokenString,
		ResetTokenExpiry: time.Now().Add(time.Minute * 10),
	}

	// Update the token record in the database
	result := tx.Model(&model.AdminResetPassword{}).Where("user_id = ?", userId).Assign(newToken).FirstOrCreate(&newToken)
	if result.Error != nil {
		return "", result.Error
	}

	return tokenString, nil
}

func ParseResetPasswordToken(tokenString string) (jwt.MapClaims, error) {
	secretResetToken := viper.GetString("secret_reset_token.key")
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretResetToken), nil
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

// Clear Reset Password Token
func ClearResetPasswordToken(userId int) error {
	adminResetPassword := model.AdminResetPassword{
		ResetToken:       "cleared",
		ResetTokenExpiry: time.Now(),
	}
	result := database.Db.Model(&model.AdminResetPassword{}).Where(`user_id = ?`, userId).Updates(adminResetPassword)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// Validate Reset Password Token
func ValidateResetPasswordToken(tokenString string) bool {
	claims, err := ParseResetPasswordToken(tokenString)
	if err != nil {
		return false
	}

	// Extract the user ID
	userId := claims["user_id"].(float64)

	// Check if the token exists in the database
	var adminResetPassword model.AdminResetPassword
	if result := database.Db.Where("user_id = ? AND reset_token = ?", userId, tokenString).First(&adminResetPassword); result.Error != nil {
		return false
	}

	if adminResetPassword.ResetTokenExpiry.Format("2006-01-02 15:04:05") < time.Now().Format("2006-01-02 15:04:05") {
		return false
	}

	return true
}

// Clear API Token of a user
func ClearApiTokenOfUser(userId int) error {
	newToken := model.Token{
		ApiToken:    "cleared",
		TokenExpire: time.Now(),
	}

	// Update the token in the database
	result := database.Db.Model(&model.Token{}).Where("user_id = ?", userId).Updates(newToken)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
