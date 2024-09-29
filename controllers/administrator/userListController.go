package administrator

import (
	"net/http"
	"root/auth"
	"root/constant"
	"root/database"
	"root/model"
	"root/permission"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Change Front User Role This is a duplicate function, not needed
func HandleChangeFrontUserRole(c *gin.Context) {
	// Get user id and role name from request body
	userIdString := c.PostForm("user_id")
	roleIdString := c.PostForm("role_id")

	// Convert the user id to an integer
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user id"})
		return
	}

	// Convert the role id to an integer
	roleId, err := strconv.Atoi(roleIdString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid role id"})
		return
	}

	// Assign the role to the user
	status := permission.ChangeFrontUserRole(userId, roleId)

	// Clear API Token of target user
	auth.ClearApiTokenOfUser(userId)

	c.JSON(200, status)
}

// Handle Update Front User Information
func HandleUpdateFrontUserInfo(c *gin.Context) {
	// Get the user id from the request body
	userIdString := c.PostForm("user_id")
	userId, err := strconv.Atoi(userIdString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user id"})
		return
	}

	// Get the role id from the request body
	roleIdString := c.PostForm("role_id")
	roleIdInt, err := strconv.Atoi(roleIdString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid role id"})
		return
	}

	// Get the user status from the request body
	userStatusString := c.PostForm("user_status")

	// Convert the user status to a boolean
	userStatus, err := strconv.ParseBool(userStatusString)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid user status"})
		return
	}

	// Fetch the current user data from the database
	var currentUser model.ScholarizeUser
	if err := database.Db.Where("user_id = ?", userId).First(&currentUser).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User not found"})
		return
	}

	// Get the current user role
	userRoleData, err := permission.GetFrontUserRoleData(userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error fetching user role"})
		return
	}

	// Check for changes in role and status
	roleChanged := userRoleData.RoleID != roleIdInt
	statusChanged := currentUser.UserStatus != userStatus

	// Update role if changed
	if roleChanged {
		result := permission.ChangeFrontUserRole(userId, roleIdInt)

		// Check if there is an error message in the result
		if resultMap, ok := result.(map[string]string); ok {
			if errMsg, exists := resultMap["error"]; exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
				return
			}

			// Clear API Token of target user
			auth.ClearApiTokenOfUser(userId)
		}
	}

	// Update status if changed
	if statusChanged {
		if err := database.Db.Model(&model.ScholarizeUser{}).Where("user_id = ?", userId).Update("user_status", userStatus).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
			return
		}

		// Clear API Token of target user
		auth.ClearApiTokenOfUser(userId)
	}

	if !roleChanged && !statusChanged {
		c.JSON(http.StatusOK, gin.H{"message": "No changes made to user information"})
		return
	}

	userRoleData, _ = permission.GetFrontUserRoleData(userId)
	c.JSON(http.StatusOK, gin.H{"message": "User information updated successfully", "new_user_info": constant.UserList{
		UserId:         currentUser.UserID,
		UserName:       currentUser.UserName,
		UserEmail:      currentUser.UserEmail,
		UserProfileImg: currentUser.UserProfileImg,
		UserStatus:     userStatus,
		UserRole:       userRoleData,
	}})
}

func HandleGetFrontUserList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	searchTerm := strings.ToLower(c.DefaultQuery("search", ""))
	roleFilter := c.Query("roles") // e.g., "User,Advisor,HOD"
	offset := (page - 1) * count

	var users []model.ScholarizeUser

	var totalUser int64
	database.Db.Model(&users).Count(&totalUser)

	query := database.Db.Joins("JOIN userrole ON scholarize_user.user_id = userrole.user_id").
		Joins("JOIN role ON userrole.role_id = role.role_id")

	// Apply case-insensitive search if provided
	if searchTerm != "" {
		searchTermPattern := "%" + searchTerm + "%"
		query = query.Where("LOWER(user_name) LIKE ? OR LOWER(user_email) LIKE ?", searchTermPattern, searchTermPattern)
	}

	// Order by user id
	query = query.Order("CASE WHEN user_status = true THEN 1 ELSE 2 END, user_name ASC")

	// Apply role filter if provided
	if roleFilter != "" {
		roles := strings.Split(roleFilter, ",")
		query = query.Where("role.role_name IN (?)", roles)
	}

	// Exclude admin roles
	excludedRoles := []string{constant.AdminRole, constant.SuperAdminRole}
	query = query.Where("role.role_name NOT IN (?)", excludedRoles)

	query.Limit(count).Offset(offset).Find(&users)

	var userInfo []constant.UserList
	for _, user := range users {
		userRoleData, _ := permission.GetFrontUserRoleData(user.UserID)
		userInfo = append(userInfo, constant.UserList{
			UserId:         user.UserID,
			UserName:       user.UserName,
			UserEmail:      user.UserEmail,
			UserProfileImg: user.UserProfileImg,
			UserStatus:     user.UserStatus,
			UserRole:       userRoleData,
		})
	}

	// Get roles that are not admin roles
	var roleList []model.Role
	err := database.Db.Where("role_name NOT IN (?)", excludedRoles).Find(&roleList).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roles"})
		return
	}

	totalResult := int64(len(userInfo))

	c.JSON(http.StatusOK, gin.H{
		"roles":        roleList,
		"users":        userInfo,
		"total_result": totalResult,
		"total_user":   totalUser,
	})
}
