package administrator

import (
	"net/http"
	"root/constant"
	"root/database"
	"root/model"

	"github.com/gin-gonic/gin"
)

type RoleWithPermissions struct {
	RoleID      int             `json:"role_id"`
	RoleName    string          `json:"role_name"`
	RoleColor   string          `json:"role_color"`
	Permissions map[string]bool `json:"permissions"`
}

type RoleWithPermissionsUpdate struct {
	RoleID      int             `json:"role_id"`
	Permissions map[string]bool `json:"permissions"`
}

type RolesPermissionsUpdate struct {
	RolePermissions []RoleWithPermissionsUpdate `json:"role_permissions"`
}

func GetAllRolePermissionsList() ([]RoleWithPermissions, error) {
	var roles []model.Role
	var permissions []model.Permission
	var rolePermissions []model.RolePermission

	// Fetch all permissions
	err := database.Db.Find(&permissions).Error
	if err != nil {
		return nil, err
	}

	// Create a map for quick permission lookup
	permissionMap := make(map[int]string)
	for _, p := range permissions {
		permissionMap[p.PermissionID] = p.PermissionName
	}

	// Fetch all roles excluding 'Admin' and 'Super Admin'
	err = database.Db.Not("role_name IN (?)", []string{constant.AdminRole, constant.SuperAdminRole}).Find(&roles).Error
	if err != nil {
		return nil, err
	}

	// Fetch all role permissions
	err = database.Db.Find(&rolePermissions).Error
	if err != nil {
		return nil, err
	}

	// Create a map for quick lookup of role permissions
	rolePermMap := make(map[int]map[int]bool)
	for _, rp := range rolePermissions {
		if rolePermMap[rp.RoleID] == nil {
			rolePermMap[rp.RoleID] = make(map[int]bool)
		}
		rolePermMap[rp.RoleID][rp.PermissionID] = true
	}

	// Construct the final response
	roleWithPermissionsList := make([]RoleWithPermissions, 0)
	for _, role := range roles {
		// Initialize permissions for each role with 'false'
		rolePerms := make(map[string]bool)
		for _, name := range permissionMap {
			rolePerms[name] = false
		}
		// Set permission to 'true' if role has it
		for permID, hasPermission := range rolePermMap[role.RoleID] {
			if hasPermission {
				rolePerms[permissionMap[permID]] = true
			}
		}
		roleWithPermissionsList = append(roleWithPermissionsList, RoleWithPermissions{
			RoleID:      role.RoleID,
			RoleName:    role.RoleName,
			RoleColor:   role.RoleColor,
			Permissions: rolePerms,
		})
	}

	return roleWithPermissionsList, nil
}

func HandleGetRolePermissionList(c *gin.Context) {
	rolePermissionsMap, err := GetAllRolePermissionsList()
	if err != nil {
		return
	}

	c.JSON(200, gin.H{
		"role_permissions": rolePermissionsMap,
	})
}

func HandleUpdateRolePermissions(c *gin.Context) {
	var updatedRoles RolesPermissionsUpdate
	if err := c.BindJSON(&updatedRoles); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not begin transaction"})
		return
	}

	for _, roleUpdate := range updatedRoles.RolePermissions {
		currentPerms, err := getCurrentPermissions(roleUpdate.RoleID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch current permissions"})
			return
		}

		// Detect changes
		toAdd, toRemove := detectPermissionChanges(currentPerms, roleUpdate.Permissions)

		// Delete permissions to remove
		for permName := range toRemove {
			var permission model.Permission
			if err := tx.Where("permission_name = ?", permName).First(&permission).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Permission not found"})
				return
			}
			if err := tx.Where("role_id = ? AND permission_id = ?", roleUpdate.RoleID, permission.PermissionID).Delete(&model.RolePermission{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission"})
				return
			}
		}

		// Add permissions to add
		for permName := range toAdd {
			var permission model.Permission
			if err := tx.Where("permission_name = ?", permName).First(&permission).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Permission not found"})
				return
			}
			newRolePerm := model.RolePermission{
				RoleID:       roleUpdate.RoleID,
				PermissionID: permission.PermissionID,
			}
			if err := tx.Create(&newRolePerm).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add permission"})
				return
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role permissions updated successfully"})
}

func detectPermissionChanges(current, updated map[string]bool) (toAdd, toRemove map[string]bool) {
	toAdd, toRemove = make(map[string]bool), make(map[string]bool)
	for perm, hasPerm := range updated {
		if hasPerm && !current[perm] {
			toAdd[perm] = true
		}
	}
	for perm, hasPerm := range current {
		if hasPerm && !updated[perm] {
			toRemove[perm] = true
		}
	}
	return toAdd, toRemove
}

func getCurrentPermissions(roleID int) (map[string]bool, error) {
	permissions := make(map[string]bool)

	// Set all permission to false
	var allPermissions []model.Permission
	if err := database.Db.Find(&allPermissions).Error; err != nil {
		return nil, err
	}
	for _, perm := range allPermissions {
		permissions[perm.PermissionName] = false
	}

	// Update the map of permissions
	var rolePerms []model.RolePermission
	if err := database.Db.Where("role_id = ?", roleID).Find(&rolePerms).Error; err != nil {
		return nil, err
	}
	for _, rolePerm := range rolePerms {
		var perm model.Permission
		if err := database.Db.Where("permission_id = ?", rolePerm.PermissionID).First(&perm).Error; err != nil {
			continue
		}
		permissions[perm.PermissionName] = true
	}

	return permissions, nil
}
