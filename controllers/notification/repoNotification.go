package notification

import (
	"root/database"
	"root/model"
	"time"
)

// Approval notification
func InsertApprovalNotification(researchTitle string, userID int) error {
	// Show only first 20 characters of collab name
	researchTitle = showFirst20Characters(researchTitle)
	notification := model.Notification{
		NotificationAt:  time.Now(),
		NotificationMsg: "Your paper " + researchTitle + " has been approved by HOD at " + time.Now().Format("Jan-2-2006 03:04 PM") + ".",
		IsCollabInvite:  false,
		Link:            "/dashboard/repository/my-publications",
		UserIDs:         []int64{int64(userID)},
		InviteToken:     "",
		UserReads:       []int64{},
	}
	err := database.Db.Create(&notification).Error
	if err != nil {
		return err
	}
	return nil
}

// Reject notification
func InsertRejectNotification(researchTitle string, userID int) error {
	// Show only first 20 characters of collab name
	researchTitle = showFirst20Characters(researchTitle)
	notification := model.Notification{
		NotificationAt:  time.Now(),
		NotificationMsg: "Your paper " + researchTitle + " has been rejected by HOD at " + time.Now().Format("Jan-2-2006 03:04 PM") + ".",
		IsCollabInvite:  false,
		Link:            "/dashboard/repository/my-publications",
		UserIDs:         []int64{int64(userID)},
		InviteToken:     "",
		UserReads:       []int64{},
	}
	err := database.Db.Create(&notification).Error
	if err != nil {
		return err
	}
	return nil
}

// New paper notification
func InsertNewPaperNotification(researchTitle string, userID int) error {
	// Show only first 20 characters of collab name
	researchTitle = showFirst20Characters(researchTitle)
	notification := model.Notification{
		NotificationAt:  time.Now(),
		NotificationMsg: "A new paper " + researchTitle + " has been submitted for review at " + time.Now().Format("Jan-2-2006 03:04 PM") + ".",
		IsCollabInvite:  false,
		Link:            "/dashboard/repository/hod-submission",
		UserIDs:         []int64{int64(userID)},
		InviteToken:     "",
		UserReads:       []int64{},
	}
	err := database.Db.Create(&notification).Error
	if err != nil {
		return err
	}
	return nil
}

// Resubmit paper notification
func InsertResubmitPaperNotification(researchTitle string, userID int) error {
	// Show only first 20 characters of collab name
	researchTitle = showFirst20Characters(researchTitle)
	notification := model.Notification{
		NotificationAt:  time.Now(),
		NotificationMsg: "The paper " + researchTitle + " has been re-submitted for review at " + time.Now().Format("Jan-2-2006 03:04 PM") + ".",
		IsCollabInvite:  false,
		Link:            "/dashboard/repository/hod-submission",
		UserIDs:         []int64{int64(userID)},
		InviteToken:     "",
		UserReads:       []int64{},
	}
	err := database.Db.Create(&notification).Error
	if err != nil {
		return err
	}
	return nil
}
