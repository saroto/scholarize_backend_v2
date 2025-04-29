package scheduling

import (
	"fmt"
	"net/http"
	"root/constant"
	"root/database"
	"root/mail"
	"root/model"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

// Helper functions
func getCollabOfUserByID(userID int) ([]model.Collab, error) {
	// Get Collab of User
	var collabs []model.Collab
	// Query for collabs where the user is the owner or a member and collab is not archived
	err := database.Db.Table("collab").
		Joins("LEFT JOIN collab_member ON collab.collab_id = collab_member.collab_id AND collab_member.user_id = ? AND collab_member.joined = true", userID).
		Where("collab.owner_id = ? OR (collab_member.user_id = ? AND collab_member.joined = true)", userID, userID).
		Where("collab.collab_archive_status = false").
		Group("collab.collab_id").
		Order("collab.collab_name ASC").
		Find(&collabs).Error
	if err != nil {
		return nil, err
	}

	return collabs, nil
}

// PREORAL Adjustments
// Get collab ids of the user that is the owner
func GetCollabOfUserOwner(userID int) []int {
	// Get Collab of User
	var collabs []model.Collab
	// Query for collabs where the user is the ownerand collab is not archived
	err := database.Db.Table("collab").
		Where("collab.owner_id = ?", userID).
		Where("collab.collab_archive_status = false").
		Group("collab.collab_id").
		Find(&collabs).Error
	if err != nil {
		return nil
	}

	var collabIds []int
	for _, collab := range collabs {
		collabIds = append(collabIds, collab.CollabID)
	}

	return collabIds
}

// PREORAL Adjustments
// Get the schedules of the collab owner
func GetCollabSchedulesOfOwner(collabIDs []int) ([]model.Schedule, error) {
	// Get the schedules of the collab owner
	var schedules []model.Schedule
	err := database.Db.Joins("JOIN schedulecollab ON schedulecollab.schedule_id = schedule.schedule_id").
		Where("schedulecollab.collab_id IN (?)", collabIDs).
		Find(&schedules).Error
	if err != nil {
		return nil, err
	}

	return schedules, nil
}

// PREORAL Adjustments
// Get the schedules of the collab owner with today's date of the new schedule
func GetCollabSchedulesOfOwnerToday(collabIDs []int, startTime time.Time, endTime time.Time) ([]model.Schedule, error) {
	// Get the schedules of the collab owner
	var schedules []model.Schedule
	err := database.Db.Joins("JOIN schedulecollab ON schedulecollab.schedule_id = schedule.schedule_id").
		Where("schedulecollab.collab_id IN (?)", collabIDs).
		Where("schedule.schedule_time_start::date = ?::date", startTime).
		Find(&schedules).Error
	if err != nil {
		return nil, err
	}

	return schedules, nil
}

// PREORAL Adjustments
// Check if the new schedule has overlapping time with the owner's schedule
func CheckOwnerOverlappingTime(schedules []model.Schedule, startTime time.Time, endTime time.Time) bool {
	for _, schedule := range schedules {
		if startTime.Before(schedule.ScheduleTimeEnd) && endTime.After(schedule.ScheduleTimeStart) {
			return true
		}
	}
	return false
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

func collabMemberHasSchedulePermission(userID, collabID int, requiredPerm string) bool {
	// Get the collab
	var collab model.Collab
	err := database.Db.Table("collab").
		Where("collab_id = ?", collabID).
		First(&collab).Error
	if err != nil {
		return false
	}

	// Check if the user is owner
	if collab.OwnerID == userID {
		fmt.Printf("User %d is the owner of the collaboration %d\n", userID, collabID)
		fmt.Printf("Therefore, he has permission: %s\n", requiredPerm)
		return true
	}
	
	// Check if the required permission is in the collab member permissions
	if !collabMemberHasPermission(collabID, requiredPerm) {
		fmt.Printf("User %d in collab %d does not have permission: %s\n", userID, collabID, requiredPerm)
		return false
	}

	fmt.Printf("User %d in collab %d has permission: %s\n", userID, collabID, requiredPerm)
	return true
}

// CLEANED
// Handle Get all schedules of a user
func HandleGetUserSchedules(c *gin.Context) {
	userID := c.MustGet("userID").(int)

	// Get Collab of User
	collabs, err := getCollabOfUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve collabs"})
		return
	}

	var collabSchedule []constant.CollabSchedule
	for _, collab := range collabs {
		var schedules []model.Schedule
		if err := database.Db.Joins("JOIN schedulecollab ON schedulecollab.schedule_id = schedule.schedule_id").
			Where("schedulecollab.collab_id = ?", collab.CollabID).
			Find(&schedules).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve schedules"})
			return
		}

		var scheduleDetails []constant.ScheduleDetail
		for _, schedule := range schedules {
			// Get user info
			var user model.ScholarizeUser
			if err := database.Db.Table("scholarize_user").
				Where("user_id = ?", schedule.UserID).
				First(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user info"})
				return
			}

			scheduleDetails = append(scheduleDetails, constant.ScheduleDetail{
				ScheduleID:        schedule.ScheduleID,
				ScheduleTitle:     schedule.ScheduleTitle,
				ScheduleTimeStart: schedule.ScheduleTimeStart,
				ScheduleTimeEnd:   schedule.ScheduleTimeEnd,
				RepeatInterval:    schedule.RepeatInterval,
				RepeatGroup:       schedule.RepeatGroup,
				UserID:            schedule.UserID,
				UserName:          user.UserName,
			})
		}

		collabSchedule = append(collabSchedule, constant.CollabSchedule{
			CollabID:            collab.CollabID,
			CollabName:          collab.CollabName,
			CollabArchiveStatus: collab.CollabArchiveStatus,
			CollabColor:         collab.CollabColor,
			ScheduleDetails:     scheduleDetails,
		})
	}
	c.JSON(http.StatusOK, gin.H{
		"collab_schedules": collabSchedule,
	})
}

// PREORAL Adjustments
// CLEANED
// TO TEST SMTP
// Handle Create Schedule For Multiple Selected Collabs that the user is part of
func HandleCreateScheduleForSelectedCollabs(c *gin.Context) {
	// Get user ID
	userID := c.MustGet("userID").(int)
	notify := c.Query("notify")

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token or is invalid"})
		return
	}

	var newSchedules constant.NewScheduleForCollabs
	if err := c.ShouldBindJSON(&newSchedules); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if missing fields
	if newSchedules.CollabIDs == nil ||
		newSchedules.ScheduleTitle == "" ||
		newSchedules.ScheduleTimeStart.IsZero() ||
		newSchedules.ScheduleTimeEnd.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing fields"})
		return
	}

	// Loop through the collab IDs
	for _, collabID := range newSchedules.CollabIDs {
		// Get the owner id
		var collab model.Collab
		if err := database.Db.Table("collab").
			Where("collab_id = ?", collabID).
			First(&collab).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve collab"})
			return
		}

		// Check if the owner schedule has overlapping time
		collabOftheOwner := GetCollabOfUserOwner(collab.OwnerID)

		schedules, err := GetCollabSchedulesOfOwnerToday(collabOftheOwner, newSchedules.ScheduleTimeStart, newSchedules.ScheduleTimeEnd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve schedules of the collab owner"})
			return
		}

		if CheckOwnerOverlappingTime(schedules, newSchedules.ScheduleTimeStart, newSchedules.ScheduleTimeEnd) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Schedule overlaps! Please select another time"})
			return
		}
	}

	// Start transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	for _, collabID := range newSchedules.CollabIDs {
		// Get the collab
		var collab model.Collab
		if err := tx.Table("collab").
			Where("collab_id = ?", collabID).
			First(&collab).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching collab"})
			return
		}

		// Check if user has permission to create schedule for collab
		if !collabMemberHasSchedulePermission(userID, collabID, constant.AddScheduleEvent) {
			tx.Rollback()
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "User does not have permission to create schedule for collab " + collab.CollabName,
			})
			return
		}

		newSchedule := model.Schedule{
			ScheduleTitle:     newSchedules.ScheduleTitle,
			ScheduleTimeStart: newSchedules.ScheduleTimeStart,
			ScheduleTimeEnd:   newSchedules.ScheduleTimeEnd,
			UserID:            userID,
			RepeatInterval:    newSchedules.RepeatInterval,
		}

		// Create schedule in database
		if err := tx.Create(&newSchedule).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule for collab ID " + strconv.Itoa(collabID)})
			continue
		}

		// Creating the association between the schedule and the collab
		scheduleCollab := model.ScheduleCollab{
			ScheduleID: newSchedule.ScheduleID,
			CollabID:   collabID,
		}

		if err := tx.Create(&scheduleCollab).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate schedule with collab", "details": err.Error()})
			continue
		}

		// Handle repeat schedules if needed
		if newSchedule.RepeatInterval > 0 {
			endDate := newSchedule.ScheduleTimeEnd.AddDate(0, 0, 7*newSchedule.RepeatInterval)

			// Check if the repeat dates has overlap
			for date := newSchedule.ScheduleTimeStart.AddDate(0, 0, 7); date.Before(endDate); date = date.AddDate(0, 0, 7) {
				overlap := checkIfScheduleOverlap(collab, constant.NewSchedule{
					ScheduleTimeStart: date,
					ScheduleTimeEnd:   date.Add(newSchedule.ScheduleTimeEnd.Sub(newSchedule.ScheduleTimeStart)),
				})
				if overlap {
					tx.Rollback()
					c.JSON(http.StatusBadRequest, gin.H{"error": "Schedule overlaps with the repeat dates! Please select another time"})
					return
				}
			}

			newSchedule.RepeatGroup = strconv.Itoa(newSchedule.ScheduleID)
			if err := tx.Save(&newSchedule).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update repeat group"})
				continue
			}

			for date := newSchedule.ScheduleTimeStart.AddDate(0, 0, 7); date.Before(endDate); date = date.AddDate(0, 0, 7) {
				repeatingSchedule := model.Schedule{
					ScheduleTitle:     newSchedule.ScheduleTitle,
					ScheduleTimeStart: date,
					ScheduleTimeEnd:   date.Add(newSchedule.ScheduleTimeEnd.Sub(newSchedule.ScheduleTimeStart)),
					UserID:            newSchedule.UserID,
					RepeatInterval:    newSchedule.RepeatInterval,
					RepeatGroup:       newSchedule.RepeatGroup,
				}

				if err := tx.Create(&repeatingSchedule).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create repeating schedule"})
					continue
				}

				repeatingScheduleCollab := model.ScheduleCollab{
					ScheduleID: repeatingSchedule.ScheduleID,
					CollabID:   collabID,
				}

				if err := tx.Create(&repeatingScheduleCollab).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate repeating schedule with collab", "details": err.Error()})
					continue
				}
			}
		}

		// Send notification to all members of the collab
		// SMTP
		// Get user info for email
		createScheduleSMTP := viper.GetBool("mailsmtp.toggle.userpanel.add_schedule")
		if createScheduleSMTP && notify == "yes" {
			// Get collab member and owner ids
			var members []constant.SimpleUser
			tx.Table("collab_member").
				Select("scholarize_user.user_id, scholarize_user.user_name, scholarize_user.user_email, scholarize_user.user_profile_img").
				Joins("join scholarize_user on scholarize_user.user_id = collab_member.user_id").
				Where("collab_member.collab_id = ? and collab_member.joined = ?", collab.CollabID, true).
				Where("collab_member.user_id != ?", userID).
				Scan(&members)
			if tx.Error != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collab members"})
			}

			// Get owner info
			if collab.OwnerID != userID {
				var owner constant.SimpleUser
				tx.Table("scholarize_user").
					Select("user_id, user_name, user_email, user_profile_img").
					Where("user_id = ?", collab.OwnerID).
					Scan(&owner)
				if tx.Error != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get collab owner"})
				}
				members = append(members, owner)
			}

			// Get user info
			var user model.ScholarizeUser
			if err := tx.Table("scholarize_user").
				Where("user_id = ?", userID).
				First(&user).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user info"})
				return
			}

			// Send email to all members
			for _, member := range members {
				if member.UserEmail == "" {
					continue
				}

				// Send email
				emailBody := mail.EmailTemplateData{
					PreviewHeader: "Schedule Created for " + collab.CollabName + " by " + user.UserName,
					EmailPurpose:  "Schedule title '" + newSchedule.ScheduleTitle + "' is set for " + newSchedule.ScheduleTimeStart.Format("15:04:05") + " to " + newSchedule.ScheduleTimeEnd.Format("15:04:05 2006-01-02") + " in collaboration group " + collab.CollabName + " by " + user.UserName,
					ActionURL:     viper.GetString("client.userpanel") + "/dashboard/schedule",
					Action:        "View schedule",
					EmailEnding:   "If you believe this is a mistake, please ignore this message.",
				}

				// Customize the email template
				emailBodyData, err := mail.CustomizeHTML(emailBody)
				if err != nil {
					fmt.Print("Error customizing email template: ", err)
					continue
				}

				// Send email to user
				errSending := mail.SendEmail(member.UserEmail, "Scholarize - A new schedule has been created!", emailBodyData)
				if errSending != nil {
					fmt.Print("Error customizing email template: ", err)
					continue
				}

				// Log
				fmt.Printf("Email sent to %s in collab %s\n", member.UserEmail, collab.CollabName)
			}
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Schedules created successfully for selected collabs",
		"details": newSchedules,
	})
}

// PREORAL Adjustments
// Additional functions with Filtering
func HandleGetUserSchedulesFilter(c *gin.Context) {
	userID := c.MustGet("userID").(int)
	collabIds := c.QueryArray("collab_ids")

	// if no collab_ids are provided, return none
	if len(collabIds) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"collab_schedules": []constant.CollabSchedule{},
		})
		return
	}

	var collabs []model.Collab
	err := database.Db.Table("collab").
		Select("DISTINCT collab.*").
		Where("collab.collab_id IN (?)", collabIds).
		Joins("LEFT JOIN collab_member ON collab.collab_id = collab_member.collab_id AND collab_member.user_id = ? AND collab_member.joined = true", userID).
		Where("collab.owner_id = ? OR (collab_member.user_id = ? AND collab_member.joined = true)", userID, userID).
		Find(&collabs).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve collabs"})
		return
	}

	/// if not found
	if len(collabs) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"collab_schedules": []constant.CollabSchedule{},
		})
		return
	}

	// Add owner name to each collab
	for i, collab := range collabs {
		var owner model.ScholarizeUser
		if err := database.Db.Table("scholarize_user").
			Where("user_id = ?", collab.OwnerID).
			First(&owner).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve owner info"})
			return
		}
		if userID == collab.OwnerID {
			collabs[i].CollabName = collabs[i].CollabName + " (You are the owner)"
		} else {
			collabs[i].CollabName = collabs[i].CollabName + " (Owner: " + owner.UserName + ")"
		}
	}

	var collabIDs []int

	// Get the schedules
	var collabSchedule []constant.CollabSchedule
	for _, collab := range collabs {
		var schedules []model.Schedule
		if err := database.Db.Joins("JOIN schedulecollab ON schedulecollab.schedule_id = schedule.schedule_id").
			Where("schedulecollab.collab_id = ?", collab.CollabID).
			Find(&schedules).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve schedules"})
			return
		}

		var scheduleDetails []constant.ScheduleDetail
		for _, schedule := range schedules {
			// Get user info
			var user model.ScholarizeUser
			if err := database.Db.Table("scholarize_user").
				Where("user_id = ?", schedule.UserID).
				First(&user).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user info"})
				return
			}

			scheduleDetails = append(scheduleDetails, constant.ScheduleDetail{
				ScheduleID:        schedule.ScheduleID,
				ScheduleTitle:     schedule.ScheduleTitle,
				ScheduleTimeStart: schedule.ScheduleTimeStart,
				ScheduleTimeEnd:   schedule.ScheduleTimeEnd,
				RepeatInterval:    schedule.RepeatInterval,
				RepeatGroup:       schedule.RepeatGroup,
				UserID:            schedule.UserID,
				UserName:          user.UserName,
			})
		}

		collabSchedule = append(collabSchedule, constant.CollabSchedule{
			CollabID:            collab.CollabID,
			CollabName:          collab.CollabName,
			CollabArchiveStatus: collab.CollabArchiveStatus,
			CollabColor:         collab.CollabColor,
			ScheduleDetails:     scheduleDetails,
		})

		// Get the owner collabs
		if collab.OwnerID != userID {
			collabOftheOwner := GetCollabOfUserOwner(collab.OwnerID)
			collabIDs = append(collabIDs, collabOftheOwner...)
		}
	}

	// Get the schedules of the collab owner
	schedules, err := GetCollabSchedulesOfOwner(collabIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve schedules of the collab owner"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"collab_schedules": collabSchedule,
		"owner_schedules":  schedules,
	})
}
