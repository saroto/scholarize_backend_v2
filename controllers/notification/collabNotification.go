package notification

import (
	"net/url"
	"root/database"
	"root/model"
	"strconv"
	"time"
)

func showFirst20Characters(str string) string {
	if len(str) > 20 {
		return str[:20] + "..."
	}
	return str
}

// Insert a new notification for a collaboration request
func InsertCollabInviteNotification(collabID int, userID int, token string, collabName string, ownerName string) error {
	// Show only first 20 characters of collab name
	collabName = showFirst20Characters(collabName)
	notification := model.Notification{
		NotificationAt:  time.Now(),
		NotificationMsg: "You have been invited to " + collabName + " by " + ownerName + " at " + time.Now().Format("Jan-2-2006 03:04 PM") + ".",
		IsCollabInvite:  true,
		Link:            "",
		UserIDs:         []int64{int64(userID)},
		InviteToken:     token,
		UserReads:       []int64{},
	}
	err := database.Db.Create(&notification).Error
	if err != nil {
		return err
	}
	return nil
}

// TASK NOTIFICATIONS
// New notification for create task
func TaskNotification(typeName string, creatorName string, userIDs []int, taskName string, collabID int, collabName string) error {
	// Convert userIDs to int64
	var userIDsInt64 []int64
	for _, userID := range userIDs {
		userIDsInt64 = append(userIDsInt64, int64(userID))
	}

	encodedCollabName := url.QueryEscape(collabName)

	// Only show first 20 characters of task name
	taskName = showFirst20Characters(taskName)
	collabName = showFirst20Characters(collabName)

	notification := model.Notification{
		NotificationAt:  time.Now(),
		NotificationMsg: creatorName + " " + typeName + " a task '" + taskName + "' in " + collabName + " at " + time.Now().Format("Jan-2-2006 03:04 PM") + ".",
		IsCollabInvite:  false,
		Link:            "/dashboard/collaboration/group/" + encodedCollabName + "/tasks?groupId=" + strconv.Itoa(collabID),
		UserIDs:         userIDsInt64,
		InviteToken:     "",
		UserReads:       []int64{},
	}
	err := database.Db.Create(&notification).Error
	if err != nil {
		return err
	}
	return nil
}

// Assign task notification
func AssignTaskNotification(creatorName string, userIDs []int, taskName string, collabID int, collabName string) error {
	return TaskNotification("assigned you to", creatorName, userIDs, taskName, collabID, collabName)
}

// Update task notification
func RenameTaskNotification(creatorName string, userIDs []int, taskName string, collabID int, collabName string) error {
	return TaskNotification("updated", creatorName, userIDs, taskName, collabID, collabName)
}

// Create a task notification
func CreateTaskNotification(creatorName string, userIDs []int, taskName string, collabID int, collabName string) error {
	return TaskNotification("created", creatorName, userIDs, taskName, collabID, collabName)
}

// Change task status notification
func UpdateTaskStatusNotification(creatorName string, userIDs []int, taskName string, collabID int, collabName string) error {
	return TaskNotification("updated status of", creatorName, userIDs, taskName, collabID, collabName)
}

// Delete task notification
func DeleteTaskNotification(creatorName string, userIDs []int, taskName string, collabID int, collabName string) error {
	return TaskNotification("deleted", creatorName, userIDs, taskName, collabID, collabName)
}
