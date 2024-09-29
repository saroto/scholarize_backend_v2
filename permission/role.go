package permission

import (
	"encoding/json"
	"fmt"
	"root/constant"
	"root/database"
	"root/model"
)

// Get all roles
func GetAllRoles() ([]model.Role, error) {
	var roles []model.Role
	err := database.Db.Find(&roles).Error
	return roles, err
}

// Get all roles with their permissions
func GetAllRolePermissions() ([]model.RolePermission, error) {
	var rolePermissions []model.RolePermission
	err := database.Db.Find(&rolePermissions).Error
	return rolePermissions, err
}

// Get role name by role id
func GetRoleName(roleId int) (string, error) {
	var role model.Role
	result := database.Db.Where("role_id = ?", roleId).First(&role)
	if result.Error != nil {
		return "", result.Error
	}
	return role.RoleName, nil
}

// Get role id by role name
func GetRoleId(roleName string) (int, error) {
	var role model.Role
	result := database.Db.Where("role_name = ?", roleName).First(&role)
	if result.Error != nil {
		return 0, result.Error
	}
	return role.RoleID, nil
}

// Get all roles of a user
func GetUserRoles(userId int) []model.UserRole {
	var userRoles []model.UserRole
	database.Db.Where("user_id = ?", userId).Find(&userRoles)
	return userRoles
}

// Check if a user has a specific role id
func HasRole(userId int, roleId int) bool {
	userRoles := GetUserRoles(userId)
	for _, userRole := range userRoles {
		if userRole.RoleID == roleId {
			return true
		}
	}
	return false
}

// Add a role to a user
func AssignUserRole(userId int, roleId int) {
	userRole := model.UserRole{
		UserID: userId,
		RoleID: roleId,
	}
	database.Db.Create(&userRole)
	fmt.Printf("User ID %d assigned role %d\n", userId, roleId)
}

// Remove a role from a user
func RemoveUserRole(userId int, roleId int) {
	database.Db.Where("user_id = ? AND role_id = ?", userId, roleId).Delete(&model.UserRole{})
	fmt.Printf("User ID %d removed role %d\n", userId, roleId)
}

// Get Front User role data
func GetFrontUserRoleData(userId int) (model.Role, error) {
	roles, _ := GetUserRoleData(userId)
	adminRole, _ := GetRoleId(constant.AdminRole)
	superAdminRole, _ := GetRoleId(constant.SuperAdminRole)
	var userRole model.Role
	for _, role := range roles {
		if !(role.RoleID == adminRole || role.RoleID == superAdminRole) {
			userRole = role
			break
		}
	}
	return userRole, nil
}

// Get Front user Role ID
func GetFrontUserRoleID(userId int) (int, error) {
	roles := GetUserRoles(userId)
	adminRole, _ := GetRoleId(constant.AdminRole)
	superAdminRole, _ := GetRoleId(constant.SuperAdminRole)
	for _, role := range roles {
		{
			if !(role.RoleID == adminRole || role.RoleID == superAdminRole) {
				return role.RoleID, nil
			}
		}
	}
	return 0, nil
}

// Get Front Panel User Role Name
func GetFrontPanelUserRoleName(userID int) string {

	roles, _ := GetUserRoleData(userID)

	var roleName string

	for _, role := range roles {
		if !(role.RoleName == constant.AdminRole || role.RoleName == constant.SuperAdminRole) {
			roleName = role.RoleName
			break
		}
	}
	return roleName
}

// Get Admin role data
func GetAdminPanelUserRoleData(userId int) (model.Role, error) {
	roles, _ := GetUserRoleData(userId)
	adminRole, _ := GetRoleId(constant.AdminRole)
	superAdminRole, _ := GetRoleId(constant.SuperAdminRole)
	var userRole model.Role
	for _, role := range roles {
		if role.RoleID == adminRole || role.RoleID == superAdminRole {
			userRole = role
			break
		}
	}
	return userRole, nil
}

// Get Admin or Super Admin user Role
func GetAdminPanelUserRoleID(userId int) (int, error) {
	roles := GetUserRoles(userId)
	adminRole, _ := GetRoleId(constant.AdminRole)
	superAdminRole, _ := GetRoleId(constant.SuperAdminRole)
	for _, role := range roles {
		{
			if role.RoleID == adminRole || role.RoleID == superAdminRole {
				return role.RoleID, nil
			}
		}
	}
	return 0, nil
}

// Get Back Panel User Role Name
func GetAdminPanelUserRoleName(userID int) string {
	roles, _ := GetUserRoleData(userID)

	var roleName string

	for _, role := range roles {
		if role.RoleName == constant.AdminRole || role.RoleName == constant.SuperAdminRole {
			roleName = role.RoleName
			break
		}
	}

	if roleName == "" {
		jsonData, err := json.Marshal(map[string]string{"error": "No admin role found"})
		if err != nil {
			return ""
		}
		return string(jsonData)
	}
	return roleName
}

// Change Front User Role
func ChangeFrontUserRole(userID int, newRoleID int) interface{} {
	currentRoleId, _ := GetFrontUserRoleID(userID)
	if HasRole(userID, newRoleID) {
		fmt.Printf("User ID %d already has role %d\n", userID, newRoleID)
		return map[string]string{"error": "User already has the role"}
	}

	hodRole, _ := GetRoleId(constant.HODRole)
	if currentRoleId == hodRole {
		// Check if the HOD user is exist in the DepartmentHead table
		var departmentHead model.DepartmentHead
		database.Db.Where("user_id = ?", userID).First(&departmentHead)
		if departmentHead.DepartmentHeadID != 0 {
			return map[string]string{"error": "The user is already assigned to a department"}
		}
		fmt.Printf("No Department is assigned to this HOD %d", userID)
	}

	// Remove the current role
	RemoveUserRole(userID, currentRoleId)
	// Assign the new role
	AssignUserRole(userID, newRoleID)

	return map[string]string{"message": "Role changed successfully. Force user to logout!"}
}

// Transfer super admin
func TransferSuperAdmin(sourceUserId int, targetUserID int) (bool, error) {
	adminRoleId, _ := GetRoleId("Admin")
	superAdminRoleId, _ := GetRoleId("Super Admin")

	// Demote source user to Admin
	RemoveUserRole(sourceUserId, superAdminRoleId)
	AssignUserRole(sourceUserId, adminRoleId)

	// Promote target user to Super Admin
	RemoveUserRole(targetUserID, adminRoleId)
	AssignUserRole(targetUserID, superAdminRoleId)

	return true, nil
}

// Get HOD Department of user
func GetHODdepartment(userId int) (model.Department, error) {

	// department head record
	var depHead model.DepartmentHead
	database.Db.Where("user_id = ?", userId).First(&depHead)

	// department record
	var department model.Department
	database.Db.Where("department_id = ?", depHead.DepartmentID).First(&department)
	if department.DepartmentID == 0 {
		return model.Department{}, nil
	}

	// return department data
	return department, nil
}
