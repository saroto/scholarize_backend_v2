package middleware

import (
	"fmt"
	"net/http"
	"root/database"
	"root/model"
	"root/permission"
	"strconv"

	"github.com/gin-gonic/gin"
)

// RoleHasPermission checks if the user's role has the required permission
func RoleHasPermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		role, ok := userRole.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		roleId, _ := permission.GetRoleId(role)

		// Check if the user's role has the required permission
		if !permission.RoleHasPermission(roleId, permission.GetPermissionId(requiredPermission)) {
			fmt.Printf("User with role %s does not have permission: %s\n", userRole, requiredPermission)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}

		fmt.Printf("User with role %s has permission: %s\n", userRole, requiredPermission)

		// User has the required permission; proceed to the next middleware or handler
		c.Next()
	}
}

// Collab member has specific permission through middleware
func CollabMemberHasPermissionMiddleware(requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the context
		userID, _ := c.Get("userID")

		// Get the collaboration ID from param
		collabIDStr := c.Param("collab_id")
		collabID, err := strconv.Atoi(collabIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid collaboration ID"})
			return
		}

		// Get the collab
		var collab model.Collab
		err = database.Db.Table("collab").
			Where("collab_id = ?", collabID).
			First(&collab).Error
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error fetching collab"})
			return
		}

		// Check if the user is owner
		if collab.OwnerID == userID {
			fmt.Printf("User %d is the owner of the collaboration %d\n", userID, collabID)
			fmt.Printf("Therefore, he has permission: %s\n", requiredPerm)
			c.Next()
			return
		}

		// Check if the required permission is in the collab member permissions
		if !collabMemberHasPermission(collabID, requiredPerm) {
			fmt.Printf("User %d in collab %d does not have permission: %s\n", userID, collabID, requiredPerm)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
			return
		}

		fmt.Printf("User %d in collab %d has permission: %s\n", userID, collabID, requiredPerm)

		// Proceed to next middleware or handler
		c.Next()
	}
}

func collabMemberHasPermission(collabID int, requiredPerm string) bool {
	// Get the requiredperm id
	var requiredPermId int
	err := database.Db.Model(&model.CollabPermission{}).
		Select("collab_permission_id").
		Where("collab_permission_name = ?", requiredPerm).
		Scan(&requiredPermId).Error
	if err != nil {
		return false
	}

	// Check if the collab has the required permission
	var collabMemPerm model.CollabMemberPermission
	err = database.Db.Model(&model.CollabMemberPermission{}).
		Where("collab_id = ? AND collab_permission_id = ?", collabID, requiredPermId).
		First(&collabMemPerm).Error
	if err != nil {
		return false
	}

	return collabMemPerm.CollabMemberPermID != 0
}
