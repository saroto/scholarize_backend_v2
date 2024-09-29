package administrator

import (
	"fmt"
	"net/http"
	"root/auth"
	"root/constant"
	"root/database"
	"root/generator"
	"root/mail"
	"root/model"
	"root/permission"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Handle Get Admin List
func HandleGetAdminList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	offset := (page - 1) * count
	searchTerm := strings.ToLower(c.DefaultQuery("search", ""))
	adminRoleName := constant.AdminRole

	query := database.Db.Model(&model.ScholarizeUser{}).
		Joins("JOIN userrole ON scholarize_user.user_id = userrole.user_id").
		Joins("JOIN role ON userrole.role_id = role.role_id").
		Where("role.role_name = ?", adminRoleName)

	// Get total admin count
	var totalAdmin int64
	query.Count(&totalAdmin)

	// Apply case-insensitive filtering if a search term is provided
	if searchTerm != "" {
		query = query.Where("LOWER(user_name) LIKE ? OR LOWER(user_email) LIKE ?", "%"+searchTerm+"%", "%"+searchTerm+"%")
	}

	// Order by user name
	query = query.Order("CASE WHEN user_status = true THEN 1 ELSE 2 END, user_name ASC")

	var users []model.ScholarizeUser
	err := query.Limit(count).Offset(offset).Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	adminUser := make([]constant.UserList, len(users))
	for i, user := range users {
		role, _ := permission.GetAdminPanelUserRoleData(user.UserID)
		adminUser[i] = constant.UserList{
			UserId:         user.UserID,
			UserName:       user.UserName,
			UserEmail:      user.UserEmail,
			UserProfileImg: user.UserProfileImg,
			UserStatus:     user.UserStatus,
			UserRole:       role,
		}
	}

	totalResult := int64(len(adminUser))

	c.JSON(http.StatusOK, gin.H{
		"users":        adminUser,
		"total_result": totalResult,
		"total_admin":  totalAdmin,
	})
}

// SMTP : Added Toggle
// Add a new admin user (Frontend checks if user exist in db before calling this function)
func HandleAddAdmin(c *gin.Context) {
	// Get admin id from request body
	adminString := c.PostForm("user_id")

	// Convert the admin id to an integer
	adminId, err := strconv.Atoi(adminString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user id"})
		return
	}

	// Check if user exist
	user := permission.GetUserById(adminId)
	if user.UserID == 0 {
		c.JSON(401, gin.H{"error": "User not found"})
		return
	}

	// Check if the user is an admin
	if permission.IsUserAdmin(adminId) {
		c.JSON(401, gin.H{"error": "User is already an admin"})
		return
	}

	if permission.IsUserSuperAdmin(adminId) {
		c.JSON(401, gin.H{"error": "User is a super admin"})
		return
	}

	// Assign the admin and add the role to the database
	assignAdmin(adminId)

	// Add admin to reset password table
	addAdminToResetPasswordTable(adminId)

	// Generate a password for the admin
	password, _ := generator.GeneratePlainPassword(8)
	hashedPassword, _ := generator.HashPassword(password)

	// insert the password into the database
	result := database.Db.Model(&model.ScholarizeUser{}).Where("user_id = ?", adminId).Update("user_password", hashedPassword)
	if result.Error != nil {
		c.JSON(500, gin.H{"error": "Error updating password"})
		return
	}

	// SMTP
	// Check if SMTP is enabled
	addAdminSMTP := viper.GetBool("mailsmtp.toggle.adminpanel.add_admin")
	fmt.Printf("Add Admin SMTP: %v\n", addAdminSMTP)
	if addAdminSMTP {
		// Create mail objects
		emailData := mail.EmailTemplateData{
			PreviewHeader: "Admin Credentials",
			EmailPurpose:  "You have been promoted to Admin on Scholarize! Login to Scholarize Admin Panel with a new generated password: " + password,
			ActionURL:     viper.GetString("client.adminpanel"),
			Action:        "Login",
			EmailEnding:   "If you believe this is a mistake, please contact the system administrator.",
		}

		// Customize the email template
		emailBody, err := mail.CustomizeHTML(emailData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error customizing email"})
			return
		}

		// Send email to user
		errSending := mail.SendEmail(user.UserEmail, "Scholarize - Admin Credentials", emailBody)
		if errSending != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending email"})
			return
		}
	}

	c.JSON(200, gin.H{
		"message":  "Admin added successfully",
		"password": password,
	})
}

// Assign Admin Role to Existing User
func assignAdmin(userId int) {
	// Get Admin Role ID
	roleID, err := permission.GetRoleId(constant.AdminRole)
	if err != nil {
		return
	}
	permission.AssignUserRole(userId, roleID)
	fmt.Printf("User ID %d assigned role %s\n", userId, constant.AdminRole)
}

// Add admin into reset password table
func addAdminToResetPasswordTable(userId int) {
	// Add admin to reset password table
	resetPassword := model.AdminResetPassword{
		UserID: userId,
	}
	database.Db.Create(&resetPassword)
}

// SMTP : Added Toggle
// Hande Transfer Super Admin
func HandleTransferSuperAdmin(c *gin.Context) {
	// Get user id from request body
	userIdString := c.PostForm("user_id")
	superAdminPass := c.PostForm("verification_password")

	// Get the super admin id
	superAdminIdStr, _ := c.Get("userID")
	superAdminId := superAdminIdStr.(int)

	// Convert the user id to an integer
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user id"})
		return
	}

	// Check if the source user is super admin
	if !permission.IsUserSuperAdmin(superAdminId) {
		c.JSON(401, gin.H{"error": "Source user is not a super admin"})
		return
	}

	// Check if the target user is an admin
	if !permission.IsUserAdmin(userId) {
		c.JSON(401, gin.H{"error": "Target user is not an admin"})
		return
	}

	// Get the super admin data
	superAdmin := permission.GetUserById(superAdminId)

	// Check if the super admin password is correct
	if !generator.VerifyPassword(superAdmin.UserPassword, superAdminPass) {
		c.JSON(401, gin.H{"error": "Invalid verification password"})
		return
	}

	// Begin the transfer
	_, errorStat := permission.TransferSuperAdmin(superAdminId, userId)
	if errorStat != nil {
		c.JSON(500, gin.H{"error": "Error transferring super admin"})
		return
	}

	// Clear the super admin token and admin token
	auth.ClearApiTokenOfUser(userId)
	auth.ClearApiTokenOfUser(superAdminId)

	// New super admin data
	newSuperAdmin := permission.GetUserById(userId)

	// SMTP
	// Check if SMTP is enabled
	transferSuperAdminSMTP := viper.GetBool("mailsmtp.toggle.adminpanel.transfer_superadmin")
	if transferSuperAdminSMTP {
		// Send email to the new super admin
		emailBody := mail.EmailTemplateData{
			PreviewHeader: "Super Admin Credentials",
			EmailPurpose:  "You have been promoted to Super Admin on Scholarize! Login to Scholarize Admin Panel with your existing credentials.",
			ActionURL:     viper.GetString("client.adminpanel"),
			Action:        "Login",
			EmailEnding:   "If you believe this is a mistake, please contact the system administrator.",
		}

		// Customize the email template
		emailBodyData, err := mail.CustomizeHTML(emailBody)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error customizing email"})
			return
		}

		// Send email to user
		errSending := mail.SendEmail(newSuperAdmin.UserEmail, "Scholarize - Super Admin Credentials", emailBodyData)
		if errSending != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error sending email"})
			return
		}
	}

	c.JSON(200, gin.H{"message": "Super admin transferred successfully"})
}

// Handle Remove Admin
func HandleRemoveAdmin(c *gin.Context) {
	// Get user id from request body
	userIdString := c.PostForm("user_id")
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user id"})
		return
	}
	// Get the user data
	user := permission.GetUserById(userId)

	// Check if the admin tries to delete himself
	adminIdStr, _ := c.Get("userID")
	adminId := adminIdStr.(int)
	if userId == adminId {
		c.JSON(401, gin.H{"error": "Cannot remove yourself"})
		return
	}

	// Verify email of the user
	emailVerification := c.PostForm("email_verification")
	if emailVerification == "" {
		c.JSON(400, gin.H{"error": "Email verification is required"})
		return
	}

	if emailVerification != user.UserEmail {
		c.JSON(400, gin.H{"error": "Invalid admin email"})
		return
	}

	// Check if the user is an admin
	if !permission.IsUserAdmin(userId) {
		c.JSON(401, gin.H{"error": "User is not an admin"})
		return
	}

	// Check if the user is a super admin
	if permission.IsUserSuperAdmin(userId) {
		c.JSON(401, gin.H{"error": "Cannot remove super admin user"})
		return
	}

	// If everything is okay, remove the user from the admin role
	adminRoleId, _ := permission.GetRoleId(constant.AdminRole)
	permission.RemoveUserRole(userId, adminRoleId)

	// Clear the user token
	auth.ClearApiTokenOfUser(userId)

	// Remove the user from the reset password table
	database.Db.Where("user_id = ?", userId).Delete(&model.AdminResetPassword{})

	// Get user name
	userEmail := user.UserEmail

	c.JSON(200, gin.H{"message": "User " + userEmail + " removed successfully"})
}
