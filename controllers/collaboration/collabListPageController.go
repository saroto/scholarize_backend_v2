package collaboration

import (
	"fmt"
	"math/rand"
	"net/http"
	"root/constant"
	"root/database"
	"root/model"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CollabMemberPermissionList struct {
	CollabID    int             `json:"collab_id"`
	Permissions map[string]bool `json:"permissions"`
}

func HandleListCollabs(c *gin.Context) {
	// Get user ID from Context
	userID := c.MustGet("userID").(int)

	// Get the collab list
	collabList, err := GetCollabListByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching collab list"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collab_list": collabList,
	})
}

func GetCollabListByUserID(userID int) (constant.CollabLists, error) {
	var collabs []model.Collab
	result := constant.CollabLists{
		ActiveCollabs:   make([]constant.CollabDetails, 0),
		ArchivedCollabs: make([]constant.CollabDetails, 0),
	}

	// Query for collabs where the user is the owner or a member
	err := database.Db.Table("collab").
		Joins("LEFT JOIN collab_member ON collab.collab_id = collab_member.collab_id AND collab_member.user_id = ? AND collab_member.joined = true", userID).
		Where("collab.owner_id = ? OR (collab_member.user_id = ? AND collab_member.joined = true)", userID, userID).
		Group("collab.collab_id").
		Order("collab.collab_name ASC").
		Find(&collabs).Error
	if err != nil {
		return result, err
	}

	// Fetch additional details for each collab
	for _, collab := range collabs {
		var members []constant.SimpleUser
		var owner constant.SimpleUser
		var actions []string

		// Get owner details
		err := database.Db.Model(&model.ScholarizeUser{}).Where("user_id = ?", collab.OwnerID).Scan(&owner).Error
		if err != nil {
			return result, err
		}

		// Get member details
		err = database.Db.Table("collab_member").
			Select("scholarize_user.user_id, scholarize_user.user_name, scholarize_user.user_email, scholarize_user.user_profile_img").
			Joins("join scholarize_user on scholarize_user.user_id = collab_member.user_id").
			Where("collab_member.collab_id = ? and collab_member.joined = ?", collab.CollabID, true).
			Scan(&members).Error

		if err != nil {
			return result, err
		}

		// Determine actions based on user role
		if collab.CollabArchiveStatus {
			actions = []string{"View Info"}
		} else {
			if userID == collab.OwnerID {
				actions = []string{"View Info", "Edit Collab", "Edit Permission", "Archive Collab", "Delete Collab"}
			} else {
				actions = []string{"View Info", "Leave Collab"}
			}
		}

		// Construct the collab details object
		collabDetails := constant.CollabDetails{
			CollabID:            collab.CollabID,
			CollabName:          collab.CollabName,
			CollabArchiveStatus: collab.CollabArchiveStatus,
			CollabColor:         collab.CollabColor,
			Owner:               owner,
			Members:             members,
			Actions:             actions,
			TotalMembers:        len(members) + 1,
		}

		// Append to the appropriate list based on the archive status
		if collab.CollabArchiveStatus {
			result.ArchivedCollabs = append(result.ArchivedCollabs, collabDetails)
		} else {
			result.ActiveCollabs = append(result.ActiveCollabs, collabDetails)
		}
	}

	return result, nil
}

// Handle client search for available members to invite
func HandleGetAvailableMembersForNewCollab(c *gin.Context) {
	// Get user ID from Context
	userID := c.MustGet("userID").(int)

	// Get the search query and pagination limit
	query := c.DefaultQuery("search", "")
	limit := c.DefaultQuery("limit", "5")

	// Convert limit to integer
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 5 || limitInt > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}

	// Get the available members
	availableMembers, err := getAvailableUsersForNewCollab(userID, query, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching available members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available_members": availableMembers,
	})
}

func getAvailableUsersForNewCollab(ownerId int, query string, limit int) ([]constant.SimpleUser, error) {
	var users []constant.SimpleUser
	err := database.Db.Table("scholarize_user").
		Select("user_id, user_name, user_email, user_profile_img").
		Where("user_id != ? AND (user_name ILIKE ? OR user_email ILIKE ?)", ownerId, "%"+query+"%", "%"+query+"%").
		Limit(limit).
		Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// Handle create the Collab
func HandleCreateCollab(c *gin.Context) {
	// Get user ID from Context
	userID := c.MustGet("userID").(int)

	// Get the collab name from the request
	collabName := c.PostForm("collab_name")
	if collabName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab name is required"})
		return
	}

	// Trim collab name
	collabName = strings.TrimSpace(collabName)

	if checkExistCollabName(collabName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab name already exists"})
		return
	}

	// Generate random color
	color := generateRandomColor()

	// Create the collab
	newCollab, err := createCollab(userID, collabName, color)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating collab"})
		return
	}

	// Create invitation if there are any members
	// Get members details if there are any
	memberIDs := c.PostFormArray("member_ids")

	// filter out the owner from the member list
	for i, memberID := range memberIDs {
		memberIDInt, _ := strconv.Atoi(memberID)
		if memberIDInt == userID {
			memberIDs = append(memberIDs[:i], memberIDs[i+1:]...)
			break
		}
	}

	if len(memberIDs) > 0 {
		CreateInviteTokenForMembers(newCollab.CollabID, memberIDs)
	}

	// Set default permission for all members
	err = setDefaultMemberPermission(newCollab.CollabID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error setting default permission"})
		return
	}

	// Return details of the created collab
	c.JSON(http.StatusOK, gin.H{
		"collab": newCollab,
	})
}

func checkExistCollabName(collabName string) bool {
	var collab model.Collab
	database.Db.Where("LOWER(collab_name) = ?", strings.ToLower(collabName)).First(&collab)
	if collab.CollabID != 0 {
		return true
	}
	return false
}

func createCollab(userID int, collabName, color string) (model.Collab, error) {
	// Create the collab
	newCollab := model.Collab{
		CollabName:          collabName,
		OwnerID:             userID,
		CollabArchiveStatus: false,
		CollabColor:         color,
	}

	err := database.Db.Create(&newCollab).Error
	if err != nil {
		return model.Collab{}, err
	}

	return newCollab, nil
}

func generateRandomColor() string {
	// Check if the color is unique
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	color := r.Intn(0xffffff)
	colorHex := fmt.Sprintf("#%06x", color)

	// Check if the color is unique
	var collab model.Collab
	database.Db.Where("collab_color = ?", colorHex).First(&collab)
	if collab.CollabID != 0 {
		return generateRandomColor()
	}

	return colorHex
}

func setDefaultMemberPermission(collabID int) error {
	// Get all Collab Permissions
	var collabPermissions []model.CollabPermission
	err := database.Db.Find(&collabPermissions).Error
	if err != nil {
		return err
	}
	// Set default permission for all members
	for _, collabPermission := range collabPermissions {
		newCollabMemberPermission := model.CollabMemberPermission{
			CollabID:           collabID,
			CollabPermissionID: collabPermission.CollabPermissionID,
		}
		err := database.Db.Create(&newCollabMemberPermission).Error
		if err != nil {
			return err
		}
	}

	return nil
}

// GET individual details of the Collabs
func GetIndividualCollabDetails(collabID int) (constant.IndividualCollabDetails, error) {
	// Get collab details
	var collab model.Collab
	err := database.Db.Where("collab_id = ?", collabID).First(&collab).Error
	if err != nil {
		return constant.IndividualCollabDetails{}, err
	}

	// Get owner details
	var owner constant.SimpleUser
	err = database.Db.Model(&model.ScholarizeUser{}).Where("user_id = ?", collab.OwnerID).Scan(&owner).Error
	if err != nil {
		return constant.IndividualCollabDetails{}, err
	}

	// Get member details
	var members []constant.SimpleUser
	err = database.Db.Table("collab_member").
		Select("scholarize_user.user_id, scholarize_user.user_name, scholarize_user.user_email, scholarize_user.user_profile_img").
		Joins("join scholarize_user on scholarize_user.user_id = collab_member.user_id").
		Where("collab_member.collab_id = ? AND collab_member.joined = ?", collabID, true).
		Scan(&members).Error
	if err != nil {
		return constant.IndividualCollabDetails{}, err
	}

	// Get the pending invites
	var pendingInvites []constant.SimpleUser
	err = database.Db.Table("collab_member").
		Select("scholarize_user.user_id, scholarize_user.user_name, scholarize_user.user_email, scholarize_user.user_profile_img").
		Joins("join scholarize_user on scholarize_user.user_id = collab_member.user_id").
		Where("collab_member.collab_id = ? AND collab_member.joined = ?", collabID, false).
		Scan(&pendingInvites).Error
	if err != nil {
		return constant.IndividualCollabDetails{}, err
	}

	// Construct the collab details object
	individualCollabDetails := constant.IndividualCollabDetails{
		CollabID:            collab.CollabID,
		CollabName:          collab.CollabName,
		CollabArchiveStatus: collab.CollabArchiveStatus,
		Owner:               owner,
		Members:             members,
		PendingInvites:      pendingInvites,
	}

	return individualCollabDetails, nil
}

// GET Avaialble Members when Edit collab
func HandleGetAvailableMembersForCollab(c *gin.Context) {
	collabId := c.Query("collab_id")

	// Convert collabId to integer
	collabIdInt, err := strconv.Atoi(collabId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collaboration ID"})
		return
	}

	// check if collab exits
	exist := collabExist(collabIdInt)
	if !exist {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collaboration does not exist"})
		return
	}

	// check if collab is archived
	archived := collabIsArchived(collabIdInt)
	if archived {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collaboration is archived and cannot be updated"})
		return
	}

	// Is collab owner
	userID, _ := c.Get("userID")
	if !isOwnerCollab(userID.(int), collabIdInt) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get the search query and pagination limit
	query := c.DefaultQuery("search", "")
	limit := c.DefaultQuery("limit", "5")

	// Convert limit to integer
	limitInt, err := strconv.Atoi(limit)
	if err != nil || limitInt < 5 || limitInt > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit"})
		return
	}

	// Get the available members
	availableMembers, err := getAvailableUsersForCollab(collabIdInt, query, limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching available members"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available_members": availableMembers,
	})
}

func getAvailableUsersForCollab(collabId int, query string, limit int) ([]constant.SimpleUser, error) {
	var excludedUserIDs []int

	// Fetch owner ID and member IDs using a single query to avoid multiple database hits
	err := database.Db.Model(&model.Collab{}).Where("collab_id = ?", collabId).Pluck("owner_id", &excludedUserIDs).Error
	if err != nil {
		return nil, err
	}

	var memberIDs []int
	err = database.Db.Model(&model.CollabMember{}).Where("collab_id = ?", collabId).Pluck("user_id", &memberIDs).Error
	if err != nil {
		return nil, err
	}

	// Combine owner and member IDs
	excludedUserIDs = append(excludedUserIDs, memberIDs...)

	// Construct the final query to fetch non-members
	var users []constant.SimpleUser
	queryStr := "%" + query + "%"
	err = database.Db.Table("scholarize_user").
		Select("user_id, user_name, user_email, user_profile_img").
		Not("user_id", excludedUserIDs).
		Where("user_name ILIKE ? OR user_email ILIKE ?", queryStr, queryStr).
		Limit(limit).
		Scan(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

// GET the collab details for the update form
func HandleGetUpdateFormCollab(c *gin.Context) {
	// Get collab ID from the request
	collabID, err := strconv.Atoi(c.Query("collab_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
		return
	}

	// Check if collab is not archived
	var collab model.Collab
	database.Db.Where("collab_id = ? AND collab_archive_status = ?", collabID, false).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived and cannot be updated"})
		return
	}

	// Is collab owner
	userID, _ := c.Get("userID")
	if !isOwnerCollab(userID.(int), collabID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get the collab details
	collabDetails, err := GetIndividualCollabDetails(collabID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching collab details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collab_details": collabDetails,
	})
}

// Update the collab details POST request
func HandleUpdateCollab(c *gin.Context) {
	// Get collab ID from the request
	collabID, err := strconv.Atoi(c.PostForm("collab_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
		return
	}

	// Get the collab name from the request
	collabName := c.PostForm("collab_name")
	if collabName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab name is required"})
		return
	}

	// Trim collab name
	collabName = strings.TrimSpace(collabName)

	var collab model.Collab
	// Check if the collab exists
	database.Db.Where("collab_id = ?", collabID).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab does not exist"})
		return
	}

	// Check if the collab is not archive
	if collab.CollabArchiveStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived and cannot be updated"})
		return
	}

	// Check if the collab name is unique from the others
	var collabCheck model.Collab
	database.Db.Where("collab_id != ? AND LOWER(collab_name) = LOWER(?)", collabID, collabName).First(&collabCheck)
	if collabCheck.CollabID != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab name already exists"})
		return
	}

	// Recheck each member in the request
	memberIDs := c.PostFormArray("member_ids")
	updatedMemberIDs := make([]int, 0, len(memberIDs))
	for _, memberID := range memberIDs {
		memberIDInt, err := strconv.Atoi(memberID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
			return
		}
		updatedMemberIDs = append(updatedMemberIDs, memberIDInt)
	}

	// Update collab details and manage members
	err = updateCollab(collabID, collabName, updatedMemberIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update collaboration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Collaboration updated successfully"})
}

func updateCollab(collabID int, collabName string, updatedMemberIDs []int) error {
	// Start a transaction
	tx := database.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update collab name
	if err := tx.Model(&model.Collab{}).Where("collab_id = ?", collabID).Update("collab_name", collabName).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Get current members
	var currentMembers []model.CollabMember
	if err := tx.Where("collab_id = ? AND joined = ?", collabID, true).Find(&currentMembers).Error; err != nil {
		tx.Rollback()
		return err
	}
	currentMemberIDs := make(map[int]bool)
	for _, m := range currentMembers {
		currentMemberIDs[m.UserID] = true
	}

	// Process new members and existing members
	var newMemberIDs []int
	for _, memberID := range updatedMemberIDs {
		if _, exists := currentMemberIDs[memberID]; !exists {
			newMemberIDs = append(newMemberIDs, memberID)

			// Check if the member is already in the collab
			var member model.CollabMember
			tx.Where("collab_id = ? AND user_id = ?", collabID, memberID).First(&member)
			if member.CollabID == 0 {
				newMember := model.CollabMember{CollabID: collabID, UserID: memberID, Joined: false}
				if err := tx.Create(&newMember).Error; err != nil {
					tx.Rollback()
					return err
				}
			} else {
				continue
			}
		}
		delete(currentMemberIDs, memberID)
	}

	if err := CreateOrUpdateInvitesForMembers(tx, collabID, newMemberIDs); err != nil {
		tx.Rollback()
		return err
	}

	// Remove members who are no longer in the updated list
	for memberID := range currentMemberIDs {
		if err := tx.Where("collab_id = ? AND user_id = ?", collabID, memberID).Delete(&model.CollabMember{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

// TODO: Remove pending members
func HandleRemovePendingMember(c *gin.Context) {
	// Get collab ID from the request
	collabID, err := strconv.Atoi(c.PostForm("collab_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
		return
	}

	// Check if the collab is not deleted
	var collab model.Collab
	database.Db.Where("collab_id = ?", collabID).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab does not exist"})
		return
	}
	if collab.CollabArchiveStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived and pending members cannot be removed"})
		return
	}

	// Check if user is the owner of the collab
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
		return
	}

	if collab.OwnerID != userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not the owner of the collab"})
		return
	}

	// Get the pending member ID from the request
	pendingMemberID, err := strconv.Atoi(c.PostForm("pending_member_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pending member ID"})
		return
	}

	// Get if the pending member exists in the collab
	var pendingMember model.CollabMember
	database.Db.Where("collab_id = ? AND user_id = ? AND joined = ?", collabID, pendingMemberID, false).First(&pendingMember)
	if pendingMember.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pending member does not exist in the collab"})
		return
	}

	// Start a transaction
	tx := database.Db.Begin()

	// Delete the pending member
	if err := tx.Where("collab_id = ? AND user_id = ? AND joined = ?", collabID, pendingMemberID, false).Delete(&model.CollabMember{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing pending member"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing pending member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Pending member removed successfully",
		"member_id": pendingMemberID,
		"collab_id": collabID,
	})
}

// Delete the collab POST request
func HandleDeleteCollab(c *gin.Context) {
	// Get collab ID from the request
	collabID, err := strconv.Atoi(c.PostForm("collab_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
		return
	}

	// Check if the collab is not archive
	var collab model.Collab
	database.Db.Where("collab_id = ? AND collab_archive_status = ?", collabID, false).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived and cannot be deleted"})
		return
	}

	// Delete the collab
	err = deleteCollab(collabID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting collab"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Collab deleted successfully"})
}

func deleteCollab(collabID int) error {
	// Start a transaction
	tx := database.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete collab permissions
	if err := tx.Where("collab_id = ?", collabID).Delete(&model.CollabMemberPermission{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete collab members
	if err := tx.Where("collab_id = ?", collabID).Delete(&model.CollabMember{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete collab tasks
	if err := deleteAllTasksAndSubtasksOfCollab(tx, collabID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete collab files
	if err := deleteAllCollabFiles(tx, collabID); err != nil {
		tx.Rollback()
		return err
	}

	// Delete collab
	if err := tx.Where("collab_id = ?", collabID).Delete(&model.Collab{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
	}

	return nil
}

// Archive the collab POST request
func HandleArchiveCollab(c *gin.Context) {
	// Get collab ID from the request
	collabID, err := strconv.Atoi(c.PostForm("collab_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
		return
	}

	// Check if the collab is not archive
	var collab model.Collab
	database.Db.Where("collab_id = ? AND collab_archive_status = ?", collabID, false).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is already archived"})
		return
	}

	// Archive the collab
	err = archiveCollab(collabID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error archiving collab"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Collab archived successfully"})
}

func archiveCollab(collabID int) error {
	err := database.Db.Model(&model.Collab{}).Where("collab_id = ?", collabID).Update("collab_archive_status", true).Error
	if err != nil {
		return err
	}
	return nil
}

// GET permissions for collab members
func HandleGetPermissionForCollabMembers(c *gin.Context) {
	// Get collab ID from the query
	collabIDStr := c.DefaultQuery("collab_id", "a")

	collabID, err := strconv.Atoi(collabIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
		return
	}

	// Check if the collab is not deleted
	var collab model.Collab
	database.Db.Where("collab_id = ?", collabID).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab does not exist"})
		return
	}
	if collab.CollabArchiveStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived and permissions cannot be retrieved"})
		return
	}

	// Check if user not owner of the collab
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
		return
	}

	if collab.OwnerID != userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not the owner of the collab"})
		return
	}

	// Get the permissions
	permissions, err := getCollabMemberPermissions(collabID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching permissions"})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

func getCollabMemberPermissions(collabID int) (CollabMemberPermissionList, error) {
	// First, get all permissions available and set them to false in the map
	var permissions []model.CollabPermission
	err := database.Db.Find(&permissions).Error
	if err != nil {
		return CollabMemberPermissionList{}, err
	}

	permissionMap := make(map[string]bool)
	for _, perm := range permissions {
		permissionMap[perm.CollabPermissionName] = false
	}

	// Now, fetch permissions that are assigned to members in this collab
	var memberPermissions []model.CollabMemberPermission
	err = database.Db.Where("collab_id = ?", collabID).Find(&memberPermissions).Error
	if err != nil {
		return CollabMemberPermissionList{}, err
	}

	// Create a map of permission IDs to quickly check against the member permissions
	permissionIDMap := make(map[int]bool)
	for _, mp := range memberPermissions {
		permissionIDMap[mp.CollabPermissionID] = true
	}

	// Use this map to set true to permissions the member has
	for permID := range permissionIDMap {
		var permission model.CollabPermission
		if err := database.Db.Where("collab_permission_id = ?", permID).First(&permission).Error; err != nil {
			return CollabMemberPermissionList{}, err
		}
		permissionMap[permission.CollabPermissionName] = true
	}

	return CollabMemberPermissionList{
		CollabID:    collabID,
		Permissions: permissionMap,
	}, nil
}

// POST update permissions for collab members
func HandleUpdatePermissionForCollabMembers(c *gin.Context) {
	var req CollabMemberPermissionList
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check if the collab is not deleted
	var collab model.Collab
	database.Db.Where("collab_id = ?", req.CollabID).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab does not exist"})
		return
	}
	if collab.CollabArchiveStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived and permissions cannot be updated"})
		return
	}

	// Check if user not owner of the collab
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
		return
	}

	if collab.OwnerID != userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not the owner of the collab"})
		return
	}

	// Update permissions
	newPerms, err := updateCollabMemberPermissions(req.CollabID, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Permissions updated successfully",
		"permissions": newPerms,
	})
}

func updateCollabMemberPermissions(collabID int, permissions map[string]bool) (CollabMemberPermissionList, error) {
	// Start transaction
	tx := database.Db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// A map to hold the final state of permissions to return
	finalPermissions := make(map[string]bool)

	// Iterate over the permission map
	for permName, hasPerm := range permissions {
		var perm model.CollabPermission
		// Fetch permission ID based on name (assuming permName is unique)
		if err := tx.Where("collab_permission_name = ?", permName).First(&perm).Error; err != nil {
			tx.Rollback()
			return CollabMemberPermissionList{}, err
		}

		// Check if there's an existing record in CollabMemberPermission
		var memberPerm model.CollabMemberPermission
		err := tx.Where("collab_id = ? AND collab_permission_id = ?", collabID, perm.CollabPermissionID).First(&memberPerm).Error
		if err == nil { // Entry exists
			if !hasPerm {
				// Delete permission if it should no longer exist
				if err := tx.Delete(&memberPerm).Error; err != nil {
					tx.Rollback()
					return CollabMemberPermissionList{}, err
				}
			} else {
				// Ensure permission is considered active
				finalPermissions[permName] = true
			}
		} else if hasPerm { // No entry found and permission should be granted
			// Add permission if it should exist
			newMemberPerm := model.CollabMemberPermission{
				CollabID:           collabID,
				CollabPermissionID: perm.CollabPermissionID,
			}
			if err := tx.Create(&newMemberPerm).Error; err != nil {
				tx.Rollback()
				return CollabMemberPermissionList{}, err
			}
			finalPermissions[permName] = true
		}
		// If the permission is false and does not exist, it remains false
		if !hasPerm {
			finalPermissions[permName] = false
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return CollabMemberPermissionList{}, err
	}

	// Return the updated list of permissions
	return CollabMemberPermissionList{
		CollabID:    collabID,
		Permissions: finalPermissions,
	}, nil
}

// Leave collab group
func HandleLeaveCollab(c *gin.Context) {
	// Get collab ID from the request
	collabID, err := strconv.Atoi(c.PostForm("collab_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
		return
	}

	// Check if the collab is not deleted
	var collab model.Collab
	database.Db.Where("collab_id = ?", collabID).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab does not exist"})
		return
	}
	if collab.CollabArchiveStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived and cannot be left"})
		return
	}

	// Check if user is a member of collab
	userID, ok := c.Get("userID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID not found"})
		return
	}

	if collab.OwnerID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Owner cannot leave the collab"})
		return
	}

	var member model.CollabMember
	database.Db.Where("collab_id = ? AND user_id = ?", collabID, userID).First(&member)
	if member.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not a member of the collab"})
		return
	}
	if !member.Joined {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You haven't joined the collab yet"})
		return
	}

	// Leave the collab
	err = leaveCollab(collabID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error leaving collab"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "You have left the collab"})
}

func leaveCollab(collabID int, userID interface{}) error {
	err := database.Db.Where("collab_id = ? AND user_id = ?", collabID, userID).Delete(&model.CollabMember{}).Error
	if err != nil {
		return err
	}
	fmt.Printf("User %d has left the collab %d\n", userID, collabID)
	return nil
}

// check if collab exsit
func collabExist(collabID int) bool {
	var collab model.Collab
	database.Db.Where("collab_id = ?", collabID).First(&collab)
	return collab.CollabID != 0
}

// check if collab is archived
func collabIsArchived(collabID int) bool {
	var collab model.Collab
	database.Db.Where("collab_id = ? AND collab_archive_status = ?", collabID, true).First(&collab)
	return collab.CollabID != 0
}

// Check if collab owner
func isOwnerCollab(userID int, collabID int) bool {
	var collab model.Collab
	database.Db.Where("owner_id = ? AND collab_id = ?", userID, collabID).First(&collab)
	return collab.CollabID != 0
}
