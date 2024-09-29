package middleware

import (
	"fmt"
	"net/http"
	"root/database"
	"root/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

func IsPOSTCollabOwnerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the user ID from the context
		userID, _ := c.Get("userID")

		// Get the collaboration ID from param
		collabIDStr := c.PostForm("collab_id")
		collabID, err := strconv.Atoi(collabIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid collaboration ID"})
			return
		}

		fmt.Printf("User %d is trying to access the collaboration %d\n", userID, collabID)

		// Check if the user is the owner of the collaboration
		if !isCollabOwner(userID.(int), collabID) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Proceed to next middleware or handler
		fmt.Printf("User %d is the owner of the collaboration %d\n", userID, collabID)
		c.Next()
	}
}

func IsCollabOwnerOrMemberMiddleware() gin.HandlerFunc {
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

		fmt.Printf("User %d is trying to access the collaboration %d\n", userID, collabID)

		// Check if the user is the owner or member of the collaboration
		if !isCollabOwner(userID.(int), collabID) && !isCollabMember(userID.(int), collabID) {
			fmt.Printf("User %d is not the owner or member of the collaboration %d\n", userID, collabID)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Proceed to next middleware or handler
		fmt.Printf("User %d is the owner or member of the collaboration %d\n", userID, collabID)
		c.Next()
	}
}

func isCollabOwner(userID int, collabID int) bool {
	var collab model.Collab
	database.Db.Where("owner_id = ? AND collab_id = ?", userID, collabID).First(&collab)
	return collab.CollabID != 0
}

func isCollabMember(userID int, collabID int) bool {
	var collabMember model.CollabMember
	database.Db.Where("user_id = ? AND collab_id = ? AND joined = true", userID, collabID).First(&collabMember)
	return collabMember.CollabID != 0
}

func IsUserCollabOwnerOrMember(userID int, collabID int) bool {
	return isCollabOwner(userID, collabID) || isCollabMember(userID, collabID)
}

func CollabArchiveStatusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the collaboration ID from param
		collabIDStr := c.Param("collab_id")
		collabID, err := strconv.Atoi(collabIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid collaboration ID"})
			return
		}

		fmt.Printf("Collaboration %d is trying to be accessed\n", collabID)

		// Check if the collaboration is archived
		var collab model.Collab
		database.Db.Where("collab_id = ?", collabID).First(&collab)
		if collab.CollabID == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Collab does not exist"})
			return
		}
		if collab.CollabArchiveStatus {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Collab is archived"})
			return
		}

		// Proceed to next middleware or handler
		fmt.Printf("Collaboration %d is not archived\n", collabID)
		c.Next()
	}
}
