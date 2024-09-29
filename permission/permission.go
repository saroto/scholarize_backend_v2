package permission

import (
	"root/constant"
	"root/database"
	"root/model"
)

// Get all permissions
func GetAllPermissions() (map[int][]model.Permission, error) {
	var permissions []model.Permission
	err := database.Db.Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	categorizedPermissions := make(map[int][]model.Permission)
	for _, p := range permissions {
		categorizedPermissions[p.PermissionCategoryID] = append(categorizedPermissions[p.PermissionCategoryID], p)
	}
	return categorizedPermissions, nil
}

// Get all permissions of a role for Frontend
func GetFrontPanelRolePermissions(roleId int) ([]constant.UserPermissionList, error) {
	// Get all permissions for the role
	var permissions []model.Permission
	result := database.Db.
		Select("permission.permission_id, permission.permission_name, permission.permission_category_id").
		Joins("JOIN rolepermission ON permission.permission_id = rolepermission.permission_id").
		Joins("JOIN permission_category ON permission.permission_category_id = permission_category.permission_category_id").
		Where("rolepermission.role_id = ?", roleId).
		Find(&permissions)
	if result.Error != nil {
		return nil, result.Error
	}

	// Assuming you have a function to get category names: getCategoryName(categoryID string) string
	permissionMap := make(map[string][]string)
	for _, perm := range permissions {
		cateName := getCategoryName(perm.PermissionCategoryID)
		permissionMap[cateName] = append(permissionMap[cateName], perm.PermissionName)
	}

	var userPermissionList []constant.UserPermissionList
	for cateName, perms := range permissionMap {
		userPermissionList = append(userPermissionList, constant.UserPermissionList{
			PermissionCate:  cateName,
			UserPermissions: perms,
		})
	}

	return userPermissionList, nil
}

// Get permissions of a role
func GetRolePermissions(roleId int) ([]model.Permission, error) {
	var permissions []model.Permission
	result := database.Db.
		Select("permission.permission_id, permission.permission_name, permission.permission_category_id").
		Joins("JOIN rolepermission ON permission.permission_id = rolepermission.permission_id").
		Joins("JOIN permission_category ON permission.permission_category_id = permission_category.permission_category_id").
		Where("rolepermission.role_id = ?", roleId).
		Find(&permissions)
	if result.Error != nil {
		return nil, result.Error
	}

	return permissions, nil
}

// Add permission to a role
func AssignRolePermission(roleId int, permissionId int) {
	rolePermission := model.RolePermission{
		RoleID:       roleId,
		PermissionID: permissionId,
	}
	database.Db.Create(&rolePermission)
}

// Remove permission from a role
func RemoveRolePermission(roleId int, permissionId int) {
	database.Db.Where("role_id = ? AND permission_id = ?", roleId, permissionId).Delete(&model.RolePermission{})
}

// Check if a role has a specific permission
func RoleHasPermission(roleId int, permissionId int) bool {
	rolePermissions, _ := GetRolePermissions(roleId)
	for _, rolePermission := range rolePermissions {
		if rolePermission.PermissionID == permissionId {
			return true
		}
	}
	return false
}

// Get permission id by permission name
func GetPermissionId(permissionName string) int {
	var permission model.Permission
	result := database.Db.Where("permission_name = ?", permissionName).First(&permission)
	if result.Error != nil {
		return 0
	}
	return permission.PermissionID
}

// Get permission category name by permission category id
func getCategoryName(permissionCateId int) string {
	var permissionCategory model.PermissionCategory
	result := database.Db.Where("permission_category_id = ?", permissionCateId).First(&permissionCategory)
	if result.Error != nil {
		return ""
	}
	return permissionCategory.PermissionCategoryName
}
