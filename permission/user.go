package permission

import (
	"fmt"
	"root/constant"
	"root/database"
	"root/model"
)

// Check user exists by Email
func GetUserByEmail(email string) *model.ScholarizeUser {
	var user model.ScholarizeUser
	database.Db.Where("user_email = ?", email).First(&user)
	if database.Db.Error != nil {
		return nil
	}
	return &user
}

func GetUserById(id int) *model.ScholarizeUser {
	var user model.ScholarizeUser
	database.Db.Where("user_id = ?", id).First(&user)
	if database.Db.Error != nil {
		return nil
	}
	return &user
}

// Get user id by email
func GetUserId(email string) (int, error) {
	user := GetUserByEmail(email)
	if user == nil {
		return 0, fmt.Errorf("user not found")
	}
	return user.UserID, nil
}

// Get User Role Data
func GetUserRoleData(userID int) ([]model.Role, error) {
	var userRoles []model.UserRole
	var roles []model.Role

	// Find all user roles associated with the given user ID
	err := database.Db.Where("user_id = ?", userID).Find(&userRoles).Error
	if err != nil {
		return nil, err
	}

	// For each user role, find the associated role and add it to the roles slice
	for _, userRole := range userRoles {
		var role model.Role
		err := database.Db.Where("role_id = ?", userRole.RoleID).First(&role).Error
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, nil
}

func IsUserAdmin(userId int) bool {
	roleId, err := GetRoleId(constant.AdminRole)
	if err != nil {
		// Handle the error here
		return false
	}
	return HasRole(userId, roleId)
}

func IsUserSuperAdmin(userId int) bool {
	roleId, err := GetRoleId(constant.SuperAdminRole)
	if err != nil {
		// Handle the error here
		return false
	}
	return HasRole(userId, roleId)
}
