package notification

import (
	"errors"
	"net/http"
	"root/database"
	"root/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserNotification struct {
	Notification model.Notification
	IsRead       bool
}

// Get All notification
func GetAllNotifications(userID int) ([]UserNotification, error) {
	var notifications []model.Notification
	err := database.Db.Where("? = ANY(user_ids)", strconv.Itoa(userID)).
		Order("notification_at DESC").
		Limit(20).
		Find(&notifications).Error
	if err != nil {
		return nil, err
	}

	// Create a slice to store notifications and read status
	var userNotifications []UserNotification

	// Check if the user has read the notification
	for _, notification := range notifications {
		isRead := false
		for _, userRead := range notification.UserReads {
			if int(userRead) == userID {
				isRead = true
				break
			}
		}
		userNotifications = append(userNotifications, UserNotification{Notification: notification, IsRead: isRead})
	}

	// Return the notifications
	return userNotifications, nil
}

// Update the notification as read by the user who received the notification
func MarkNotificationAsRead(notificationID int, userID int) error {
	var notification model.Notification
	err := database.Db.Where("notification_id = ?", notificationID).First(&notification).Error
	if err != nil {
		return err
	}

	// Create a map for efficient lookup
	userReadsMap := make(map[int64]bool)
	for _, userRead := range notification.UserReads {
		userReadsMap[userRead] = true
	}

	// Check if the user has already read the notification
	if userReadsMap[int64(userID)] {
		return errors.New("user has already read the notification")
	}

	// Update user_reads field
	userReads := append(notification.UserReads, int64(userID))
	err = database.Db.Model(&notification).Update("user_reads", userReads).Error
	if err != nil {
		return err
	}
	return nil
}

// Handle Get All Notifications
func HandleGetAllNotifications(c *gin.Context) {
	userID := c.MustGet("userID").(int)

	notifications, err := GetAllNotifications(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(notifications) == 0 {
		notifications = []UserNotification{}
	}

	// Check if all notifications are read
	allRead := true
	for _, notification := range notifications {
		if !notification.IsRead {
			allRead = false
			break
		}
	}

	c.JSON(http.StatusOK, gin.H{"notifications": notifications, "allRead": allRead})
}

// Handle mark notification as read
func HandleMarkNotificationAsRead(c *gin.Context) {
	notificationID, err := strconv.Atoi(c.Param("notificationID"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	userID := c.MustGet("userID").(int)

	err = MarkNotificationAsRead(notificationID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Notification marked as read"})
}
