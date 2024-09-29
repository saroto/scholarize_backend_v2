package auth

import (
	"errors"
	"fmt"
	"net/http"
	"root/constant"
	"root/database"
	"root/generator"
	"root/mail"
	"root/model"
	"root/permission"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const userIDQuery = "user_id = ?"

// User Login
func HandleFrontPanelLogin(c *gin.Context) {
	userEmail := c.PostForm("email")
	userName := c.PostForm("name")
	userProfileURL := c.PostForm("profile_url")
	accessToken := c.PostForm("access_token")

	// Check if request body is missing field
	if userEmail == "" || userName == "" || userProfileURL == "" || accessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Must have all fields"})
		return
	}

	// Validate the user's email domain
	if !ValidateUserGmailProvider(userEmail) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email is not authorized: " + userEmail})
		return
	}

	// Validate the user's access token with Google
	_, err := VerifyGoogleToken(accessToken, userEmail, userName)
	if err != nil {
		if err.Error() == "invalid token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token"})
			return
		} else if err.Error() == "token info does not match user credentials" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token info does not match user credentials"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify Google token"})
			return
		}
	}

	// Check if the user is registered
	// first get UserRow, if not exist it return 0 as user id
	user := permission.GetUserByEmail(userEmail)

	var token string
	var newUser *model.ScholarizeUser
	if user.UserID == 0 {
		// Register the user if not exist
		newUser = registerUser(userEmail, userName, userProfileURL)
		token, _ = GenerateApiToken(userEmail)
	} else {
		// Check user status, return unauthorized if false
		if !user.UserStatus {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User " + userEmail + " is not active"})
			return
		}
		newUser = user

		// Check if user name has changed
		if user.UserName != userName {
			result := database.Db.Model(&model.ScholarizeUser{}).Where(userIDQuery, user.UserID).Update("user_name", userName)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating user name"})
				return
			}
			fmt.Println("User name updated for user ID: ", user.UserID)
		}

		// Check if user profile image has changed
		if user.UserProfileImg != userProfileURL {
			result := database.Db.Model(&model.ScholarizeUser{}).Where(userIDQuery, user.UserID).Update("user_profile_img", userProfileURL)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating profile image"})
				return
			}
			fmt.Println("Profile image updated for user ID: ", user.UserID)
		}

		// Update the token if status is active
		token, err = UpdateApiToken(userEmail)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating token"})
			return
		}
	}

	// Get permission of user role
	userRole, _ := permission.GetFrontUserRoleData(newUser.UserID)
	rolePermissions, _ := permission.GetFrontPanelRolePermissions(userRole.RoleID)

	// Get department if user role is HOD
	if userRole.RoleName == constant.HODRole {
		departmentData, errr := permission.GetHODdepartment(newUser.UserID)
		if errr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting department"})
			return
		}

		// Return JSON response with
		c.JSON(http.StatusOK, gin.H{
			"token":            token,
			"role":             userRole,
			"role_permissions": rolePermissions,
			"department":       departmentData,
		})
		return
	}

	// Return JSON response with
	c.JSON(http.StatusOK, gin.H{
		"token":            token,
		"role":             userRole,
		"role_permissions": rolePermissions,
	})
}

// Admin Login
func HandleAdminPanelLogin(c *gin.Context) {
	// Get the user email and password from the response
	userReqEmail := c.PostForm("email")
	userReqPassword := c.PostForm("password")

	// Check if request body is missing field
	if userReqEmail == "" || userReqPassword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Must have all fields"})
		return
	}

	// Check if the user is registered
	user := permission.GetUserByEmail(userReqEmail)
	if user.UserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User " + userReqEmail + " not found"})
		return
	}
	fmt.Println("Record " + userReqEmail + " found in the database with ID: " + fmt.Sprint(user.UserID))

	// Check if user is an admin or a super admin
	if !(permission.IsUserAdmin(user.UserID) || permission.IsUserSuperAdmin(user.UserID)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User " + userReqEmail + " is not an admin"})
		return
	}

	// Check user status, return unauthorized if false
	if !user.UserStatus {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User " + userReqEmail + " is not active"})
		return
	}
	fmt.Println("Status checked for " + userReqEmail)

	// Check user password
	if !generator.VerifyPassword(user.UserPassword, userReqPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
		return
	}
	fmt.Println("Password verified for " + userReqEmail)

	// Generate a new token
	var apiToken string
	var token model.Token

	result := database.Db.Where(userIDQuery, user.UserID).First(&token)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			apiToken, _ = GenerateAdminApiToken(userReqEmail)
		}
	} else {
		apiToken, _ = UpdateAdminApiToken(userReqEmail)
	}

	// Get status of boarding
	var boardingStatus bool
	var adminResetPassword model.AdminResetPassword
	result = database.Db.Where(userIDQuery, user.UserID).First(&adminResetPassword)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			boardingStatus = false
		}
	} else {
		boardingStatus = adminResetPassword.IsBoarded
	}

	// Get role name of the admin user
	roleName := permission.GetAdminPanelUserRoleName(user.UserID)

	// Get profile picture of the user
	userProfileImg := user.UserProfileImg
	if userProfileImg == "" {
		userProfileImg = viper.GetString("default_profile_img.admin")
	}

	// Return JSON response with
	c.JSON(http.StatusOK, gin.H{
		"token":            apiToken,
		"boarded":          boardingStatus,
		"role":             roleName,
		"user_name":        user.UserName,
		"user_profile_img": userProfileImg,
	})
}

// Boarding an admin
func HandleUpdateAdminPasswordOnBoarding(c *gin.Context) {
	// Get user data from token
	tokenString := ExtractToken(c)
	claims, err := ParseApiToken(tokenString)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized token"})
		return
	}

	var adminResetPassword model.AdminResetPassword

	// Check boarding status from token
	if claims["boarded"] == true {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Already boarded"})
		return
	}

	// Get the user ID from the token
	adminId := int(claims["user_id"].(float64))

	// Get user id and request password from the request body
	newPass := c.PostForm("new_password")

	fmt.Println("Boarding admin ", adminId)
	// Update the boarding status
	result := database.Db.Where(userIDQuery, adminId).First(&adminResetPassword)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized record not found"})
			return
		}
	} else {
		// Check boarding status again from database
		if adminResetPassword.IsBoarded {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Already boarded"})
			return
		}

		// Update the boarding status
		result = database.Db.Model(&model.AdminResetPassword{}).Where(userIDQuery, adminId).Update("is_boarded", true)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating boarding status"})
			return
		}
	}

	// Update password
	if !UpdatePassword(adminId, newPass) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating password"})
		return
	}

	// Respond to the client
	c.JSON(http.StatusOK, gin.H{"message": "Boarding successful. Password updated. Let user logout and login again"})
}

// SMTP : Added Toggle
// Send reset password link
func HandleSendResetPasswordLink(c *gin.Context) {
	// Extract email from request
	requestEmail := c.PostForm("email")

	var user model.ScholarizeUser
	if err := database.Db.Where("user_email = ?", requestEmail).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Email not found"})
		return
	}

	if !(permission.IsUserAdmin(user.UserID) || permission.IsUserSuperAdmin(user.UserID)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User " + requestEmail + " is not an admin"})
		return
	}

	var adminResetPassword model.AdminResetPassword
	if err := database.Db.Where(userIDQuery, user.UserID).First(&adminResetPassword).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
	}

	timeNow := time.Now().Format("2006-01-02 15:04:05")
	resetTime := adminResetPassword.ResetTokenExpiry.Format("2006-01-02 15:04:05")

	if resetTime > timeNow {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Please wait befoqre requesting another reset",
			"expiry": resetTime,
			"now":    timeNow,
		})
		return
	}

	// Start transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error starting transaction"})
		return
	}

	// Generate reset token using jwt
	resetToken, _ := GenerateResetPasswordToken(tx, user.UserID)
	if resetToken == "" {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating reset token"})
		return
	}

	// SMTP
	resetAdminPassSMTP := viper.GetBool("mailsmtp.toggle.adminpanel.reset_password")
	if resetAdminPassSMTP {
		// Create mail objects
		emailData := mail.EmailTemplateData{
			PreviewHeader: "Reset Admin Password",
			EmailPurpose:  "You have requested an admin password reset. Click the button below to reset your password.",
			ActionURL:     viper.GetString("client.adminpanel") + "/resetpassword/" + resetToken,
			Action:        "Reset Password",
			EmailEnding:   "If you did not request this, please ignore this email. This password reset is only valid for the next 10 minutes.",
		}

		// Customize the email template
		emailBody, err := mail.CustomizeHTML(emailData)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error customizing email"})
			return
		}

		// Send email to user
		errSending := mail.SendEmail(requestEmail, "Scholarize - Reset Admin Password", emailBody)
		if errSending != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending email"})
			return
		}
	}

	// Commit the transaction
	errc := tx.Commit()
	if errc.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "If your email is registered, you will receive a password reset link", "reset_token": resetToken})
}

// Reset password page
func HandleAccessResetPasswordPage(c *gin.Context) {
	// Get user data from token on route
	tokenString := c.Param("token")
	if ValidateResetPasswordToken(tokenString) {
		c.JSON(http.StatusOK, gin.H{"message": "Token valid"})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalid"})
	}
}

// Update password of admin
func HandleUpdateAdminPasswordOnReset(c *gin.Context) {
	// Extract the token from the request
	resetToken := c.PostForm("reset_token")
	newPassword := c.PostForm("new_password")

	if !ValidateResetPasswordToken(resetToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalid"})
		return
	}

	// Get the user ID from the token
	claims, err := ParseResetPasswordToken(resetToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get the user ID from the token
	adminId := int(claims["user_id"].(float64))

	// Update password
	if !UpdatePassword(adminId, newPassword) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating password"})
		return
	}

	// Clear the reset token
	ClearResetPasswordToken(adminId)

	// Clear Api Token of the user
	ClearApiTokenOfUser(adminId)

	// Update the boarding status
	var adminResetPassword model.AdminResetPassword
	result := database.Db.Where(userIDQuery, adminId).First(&adminResetPassword)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
	} else {
		// Update the boarding status if not already boarded
		if !adminResetPassword.IsBoarded {
			result = database.Db.Model(&model.AdminResetPassword{}).Where(userIDQuery, adminId).Update("is_boarded", true)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating boarding status"})
				return
			}
		}
	}

	// Respond to the client
	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset. Try login in with your new password", "logout": true})
}

// Logout user
func HandleLogout(c *gin.Context) {
	// Extract the token from the request, typically from the Authorization header
	tokenString := ExtractToken(c)
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No token provided"})
		return
	}

	// Invalidate the token in the database
	err := ClearApiToken(tokenString)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error clearing token"})
		return
	}

	// Respond to the client
	c.JSON(http.StatusOK, gin.H{"logout": true, "message": "Logout successful."})
}

// Check user email before allowing user to login
func ValidateUserGmailProvider(email string) bool {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	domain := parts[1]

	authorizedDomains := viper.GetStringMap("authorized_gmail")
	if authorized, ok := authorizedDomains[domain]; ok {
		// If the domain not exist in the authorized domains, return false
		if authorized == true {
			fmt.Printf("Email %s is authorized\n", email)
			return true
		}
	}

	fmt.Printf("Email %s is not authorized\n", email)
	return false
}

// Extract the token from the request
func ExtractToken(c *gin.Context) string {
	bearerToken := c.GetHeader("Authorization")
	if len(bearerToken) > 7 && strings.ToUpper(bearerToken[0:7]) == "BEARER " {
		return bearerToken[7:]
	}
	return ""
}

// Update the password of admin
func UpdatePassword(userId int, newPassword string) bool {
	hashedPassword, err := generator.HashPassword(newPassword)
	if err != nil {
		return false
	}

	result := database.Db.Model(&model.ScholarizeUser{}).Where(userIDQuery, userId).Update("user_password", hashedPassword)
	if result.Error != nil {
		return false
	}
	fmt.Println("Password updated for user ID: ", userId)
	return true
}

// User registeration, applies to admin and front panel
func registerUser(userEmail string, userName string, userProfileImg string) *model.ScholarizeUser {
	// Register the user
	newUser := model.ScholarizeUser{
		UserEmail:      userEmail,
		UserName:       userName,
		UserProfileImg: userProfileImg,
	}
	database.Db.Create(&newUser)
	fmt.Printf("User %s registered successfully\n", userEmail)

	// Assign User Role to the user
	roleID, err := permission.GetRoleId(constant.UserRole)
	if err != nil {
		return nil
	}
	permission.AssignUserRole(newUser.UserID, roleID)
	fmt.Printf("User %s assigned role %s\n", userEmail, constant.UserRole)

	return &newUser
}
