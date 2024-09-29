package collaboration

import (
	"fmt"
	"net/http"
	"root/constant"
	"root/controllers/notification"
	"root/database"
	"root/middleware"
	"root/model"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Insert notification
func InsertNotificationForTask(collabId int, notiType string, taskName string, creatorID int) error {
	// Get collab name
	var collab model.Collab
	err := database.Db.Where("collab_id = ?", collabId).First(&collab).Error
	if err != nil {
		return err
	}
	collabName := collab.CollabName

	// Get all collab members including owner
	memberIDs, err := GetAllCollabMembersIDs(collabId)
	if err != nil {
		return err
	}

	// Remove creator ID from member IDs
	for idx, memberID := range memberIDs {
		if memberID == creatorID {
			memberIDs = append(memberIDs[:idx], memberIDs[idx+1:]...)
			break
		}
	}

	// Get creator name
	var creator model.ScholarizeUser
	err = database.Db.Where("user_id = ?", creatorID).First(&creator).Error
	if err != nil {
		return err
	}

	// Insert notification
	var notiErr error

	switch notiType {
	case "create":
		notiErr = notification.CreateTaskNotification(creator.UserName, memberIDs, taskName, collabId, collabName)
	case "rename":
		notiErr = notification.RenameTaskNotification(creator.UserName, memberIDs, taskName, collabId, collabName)
	case "delete":
		notiErr = notification.DeleteTaskNotification(creator.UserName, memberIDs, taskName, collabId, collabName)
	case "updateStatus":
		notiErr = notification.UpdateTaskStatusNotification(creator.UserName, memberIDs, taskName, collabId, collabName)
	}

	if notiErr != nil {
		return notiErr
	}

	return nil
}

func InsertNotificationForAssignee(collabId int, taskName string, assigneeIDs []int, creatorID int) error {
	// Get collab name
	var collab model.Collab
	err := database.Db.Where("collab_id = ?", collabId).First(&collab).Error
	if err != nil {
		return err
	}
	collabName := collab.CollabName

	// Get creator name
	var creator model.ScholarizeUser
	err = database.Db.Where("user_id = ?", creatorID).First(&creator).Error
	if err != nil {
		return err
	}

	notiErr := notification.AssignTaskNotification(creator.UserName, assigneeIDs, taskName, collabId, collabName)
	if notiErr != nil {
		return notiErr
	}

	return nil
}

// Handle Get Collab Details
func HandleGetCollabDetails(c *gin.Context) {
	// Get collab id from param
	collabId, err := strconv.Atoi(c.Param("collab_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid collab id"})
		return
	}

	collabDetail, err := GetIndividualCollabDetails(collabId)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error fetching collab details"})
		return
	}

	c.JSON(200, gin.H{
		"collab": collabDetail,
	})
}

// Get all collab members including owner
func HandleGetAllCollabMembers(c *gin.Context) {
	// Get collab ID from the param
	collabIDStr := c.Param("collab_id")
	collabID, err := strconv.Atoi(collabIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid collab ID"})
	}

	// Get collab details
	var collab model.Collab
	err = database.Db.Where("collab_id = ?", collabID).First(&collab).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch collab details"})
		return
	}

	// Get owner details
	var owner constant.SimpleUser
	err = database.Db.Model(&model.ScholarizeUser{}).Where("user_id = ?", collab.OwnerID).Scan(&owner).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch owner details"})
		return
	}

	// Get member details
	var members []constant.SimpleUser
	err = database.Db.Table("collab_member").
		Select("scholarize_user.user_id, scholarize_user.user_name, scholarize_user.user_email, scholarize_user.user_profile_img").
		Joins("join scholarize_user on scholarize_user.user_id = collab_member.user_id").
		Where("collab_member.collab_id = ? AND collab_member.joined = ?", collabID, true).
		Scan(&members).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch member details"})
		return
	}

	// Join owner and members
	allMembers := append(members, owner)

	c.JSON(http.StatusOK, gin.H{
		"collab":  collab,
		"members": allMembers,
	})
}

// Get all collab members including owner ID
func GetAllCollabMembersIDs(collabID int) ([]int, error) {
	// Get owner ID
	var ownerID int
	err := database.Db.Table("collab").
		Select("owner_id").
		Where("collab_id = ?", collabID).
		Pluck("owner_id", &ownerID).Error
	if err != nil {
		return nil, err
	}

	// Get member IDs
	var memberIDs []int
	err = database.Db.Table("collab_member").
		Select("user_id").
		Where("collab_id = ? AND joined = ?", collabID, true).
		Pluck("user_id", &memberIDs).Error
	if err != nil {
		return nil, err
	}

	// Append owner ID to member IDs
	memberIDs = append(memberIDs, ownerID)

	return memberIDs, nil
}

// Get all task information for Board View and List view
func HandleGetAllTasks(c *gin.Context) {
	// Get collab id from param
	collabId, err := strconv.Atoi(c.Param("collab_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid collab id"})
		return
	}

	// Fetch tasks for the collab
	taskLists, err := getTaskByCollab(collabId)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error fetching tasks"})
		return
	}

	// Get collab archive status for future use
	var collab model.Collab
	err = database.Db.Table("collab").
		Where("collab_id = ?", collabId).
		First(&collab).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error fetching collab"})
		return
	}

	c.JSON(200, gin.H{
		"collab_name":           collab.CollabName,
		"tasks":                 taskLists,
		"collab_archive_status": collab.CollabArchiveStatus})
}

func getTaskByCollab(collabId int) ([]constant.TaskList, error) {
	// First, fetch all task statuses to ensure all are represented in the output
	var statuses []struct {
		TaskStatusID    int    `gorm:"column:task_status_id"`
		TaskStatusName  string `gorm:"column:task_status_name"`
		TaskStatusColor string `gorm:"column:task_status_color"`
	}
	err := database.Db.Table("task_status").Scan(&statuses).Error
	if err != nil {
		return nil, err
	}

	// Map to hold TaskList with grouped tasks by status, initialized with all statuses
	taskMap := make(map[int]*constant.TaskList)
	for _, status := range statuses {
		taskMap[status.TaskStatusID] = &constant.TaskList{
			TaskStatusID:    status.TaskStatusID,
			TaskStatusName:  status.TaskStatusName,
			TaskStatusColor: status.TaskStatusColor,
			TaskDetails:     []constant.TaskDetails{},
		}
	}

	// Fetch tasks with their status information only for the specified collab ID
	var tasks []struct {
		TaskID          int       `gorm:"column:task_id"`
		TaskTitle       string    `gorm:"column:task_title"`
		TaskPriority    bool      `gorm:"column:task_priority"`
		TaskStatusID    int       `gorm:"column:task_status_id"`
		StatusUpdatedAt time.Time `gorm:"column:status_updated_at"`
	}
	err = database.Db.Table("task").
		Select("task.task_id, task.task_title, task.task_priority, task.status_updated_at, statustask.task_status_id").
		Joins("JOIN statustask ON statustask.task_id = task.task_id").
		Joins("JOIN collab ON collab.collab_id = task.collab_id").
		Where("collab.collab_id = ?", collabId).
		Order("task_status_id, status_updated_at ASC").
		Scan(&tasks).Error
	if err != nil {
		return nil, err
	}

	// Populate tasks into the corresponding status in the map
	for _, task := range tasks {
		if tl, exists := taskMap[task.TaskStatusID]; exists {
			tl.TaskDetails = append(tl.TaskDetails, constant.TaskDetails{
				TaskID:       task.TaskID,
				TaskTitle:    task.TaskTitle,
				TaskPriority: task.TaskPriority,
			})
		}
	}

	// Fetch assignees for each task
	for _, taskList := range taskMap {
		for idx, detail := range taskList.TaskDetails {
			var assignees []constant.SimpleUser
			err = database.Db.Table("taskassignee").
				Select("scholarize_user.user_id, scholarize_user.user_name, scholarize_user.user_profile_img, scholarize_user.user_email").
				Joins("JOIN scholarize_user ON scholarize_user.user_id = taskassignee.user_id").
				Where("taskassignee.task_id = ?", detail.TaskID).
				Scan(&assignees).Error
			if err != nil {
				return nil, err
			}
			taskList.TaskDetails[idx].TaskAssignee = assignees
		}
	}

	// Convert map to slice for the response
	var taskLists []constant.TaskList
	for _, tl := range taskMap {
		taskLists = append(taskLists, *tl)
	}

	// Sort the task lists by status ID
	sort.Slice(taskLists, func(i, j int) bool {
		return taskLists[i].TaskStatusID < taskLists[j].TaskStatusID
	})

	return taskLists, nil
}

// ADDED NOTIFICATION
// Add new Task to the collab
func HandleAddNewTask(c *gin.Context) {
	// Get collab id from param
	collabId, err := strconv.Atoi(c.Param("collab_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid collab id"})
		return
	}

	// Get task status id from post form
	taskStatusId, errr := strconv.Atoi(c.PostForm("task_status_id"))
	if errr != nil {
		c.JSON(400, gin.H{"error": "Invalid task status"})
		return
	}

	// Get task title from post form
	taskTitle := c.PostForm("task_title")
	if taskTitle == "" {
		c.JSON(400, gin.H{"error": "Task title cannot be empty"})
		return
	}

	// Create new task record
	taskRecord := model.Task{
		TaskTitle:       taskTitle,
		TaskPriority:    false,
		CollabID:        collabId,
		StatusUpdatedAt: time.Now(),
	}

	// Save the task record
	err = database.Db.Create(&taskRecord).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error adding task"})
		return
	}

	// Check if task status exists
	var taskStatus model.TaskStatus
	err = database.Db.Where("task_status_id = ?", taskStatusId).First(&taskStatus).Error
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid task status"})
		return
	}

	// Create new status task record
	taskStatusRecord := model.StatusTask{
		TaskID:       taskRecord.TaskID,
		TaskStatusID: taskStatusId,
	}

	// Save the status task record
	err = database.Db.Create(&taskStatusRecord).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error adding task status"})
		return
	}

	// Insert notification
	err = InsertNotificationForTask(collabId, "create", taskTitle, c.MustGet("userID").(int))
	if err != nil {
		fmt.Println("Error inserting notification for task:", err)
	}

	c.JSON(200, gin.H{
		"message": "Task added successfully",
		"task":    taskRecord,
	})
}

// Update task priority
func HandleUpdateTaskPriority(c *gin.Context) {
	// Get task id from post form
	taskId, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid task id"})
		return
	}

	// Get collab id from param
	collabId, err := strconv.Atoi(c.Param("collab_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid collab id"})
		return
	}

	// Get the current task
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		c.JSON(500, gin.H{"error": "Task not found"})
		return
	}

	// Check if the task belongs to the collab
	if task.CollabID != collabId {
		c.JSON(400, gin.H{"error": "Task does not belong to the collab"})
		return
	}

	// Toggle task priority
	taskPriority := !task.TaskPriority

	// Update task priority
	err = database.Db.Model(&model.Task{}).
		Where("task_id = ?", taskId).
		Update("task_priority", taskPriority).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error updating task priority"})
		return
	}

	c.JSON(200, gin.H{"message": "Task priority updated successfully"})
}

// ADDED NOTIFICATION
// Delete task
func HandleDeleteTask(c *gin.Context) {
	// Get task id from post form
	taskId, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid task id"})
		return
	}

	// Get collab id from param
	collabId, err := strconv.Atoi(c.Param("collab_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid collab id"})
		return
	}

	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Get the current task
	var task model.Task
	if err := tx.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Task not found"})
		return
	}

	// Check if the task belongs to the collab
	if task.CollabID != collabId {
		tx.Rollback()
		c.JSON(400, gin.H{"error": "Task does not belong to the collab"})
		return
	}

	// Delete task
	if err := deleteTask(tx, taskId); err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Error deleting task"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Insert notification
	err = InsertNotificationForTask(collabId, "delete", task.TaskTitle, c.MustGet("userID").(int))
	if err != nil {
		fmt.Println("Error inserting notification for task:", err)
	}

	c.JSON(200, gin.H{"message": "Task deleted successfully"})
}

// ADDED NOTIFICATION
// Update task name in popup
func HandleUpdateTaskName(c *gin.Context) {
	// Get task id from post form
	taskId, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid task id"})
		return
	}

	// Get task title from post form
	taskTitle := c.PostForm("task_title")
	if taskTitle == "" {
		c.JSON(400, gin.H{"error": "Task title cannot be empty"})
		return
	}

	// Get collab id from param
	collabId, err := strconv.Atoi(c.Param("collab_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid collab id"})
		return
	}

	// Get the current task
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		c.JSON(500, gin.H{"error": "Task not found"})
		return
	}

	// Check if the task belongs to the collab
	if task.CollabID != collabId {
		c.JSON(400, gin.H{"error": "Task does not belong to the collab"})
		return
	}

	// Update task title
	task.TaskTitle = taskTitle
	err = database.Db.Save(&task).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error updating task title"})
		return
	}

	// Insert notification
	err = InsertNotificationForTask(collabId, "rename", taskTitle, c.MustGet("userID").(int))
	if err != nil {
		fmt.Println("Error inserting notification for task:", err)
	}

	c.JSON(200, gin.H{"message": "Task title updated successfully"})
}

// ADDED NOTIFICATION
// Update task status
func HandleUpdateTaskStatus(c *gin.Context) {
	// Get task id from post form
	taskId, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid task id"})
		return
	}

	// Get task status id from post form
	taskStatusId, errr := strconv.Atoi(c.PostForm("task_status_id"))
	if errr != nil {
		c.JSON(400, gin.H{"error": "Invalid task status"})
		return
	}

	// Get the current task
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		c.JSON(500, gin.H{"error": "Task not found"})
		return
	}

	// Get the current task status
	var statusTask model.StatusTask
	if err := database.Db.Where("task_id = ?", taskId).First(&statusTask).Error; err != nil {
		c.JSON(500, gin.H{"error": "Task status not found"})
		return
	}

	// Check if the task status is the same as the new status
	if statusTask.TaskStatusID == taskStatusId {
		c.JSON(400, gin.H{"error": "Task status is already the same"})
		return
	}

	// Update task status
	err = database.Db.Model(&model.StatusTask{}).
		Where("task_id = ?", taskId).
		Update("task_status_id", taskStatusId).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error updating task status"})
		return
	}

	// Update status updated at
	err = database.Db.Model(&model.Task{}).
		Where("task_id = ?", taskId).
		Update("status_updated_at", time.Now()).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error updating status updated at"})
		return
	}

	// Insert notification
	err = InsertNotificationForTask(task.CollabID, "updateStatus", task.TaskTitle, c.MustGet("userID").(int))
	if err != nil {
		fmt.Println("Error inserting notification for task:", err)
	}

	c.JSON(200, gin.H{"message": "Task status updated successfully"})
}

// Get task assignees (I think this is unnecessary but still good to have)
func HandleGetTaskAssignees(c *gin.Context) {
	// Get task id from the query parameter
	taskIdStr := c.Query("task_id")
	taskId, err := strconv.Atoi(taskIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Check if task exist
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Fetch assignees related to the task
	var assignees []model.TaskAssignee
	if err := database.Db.Where("task_id = ?", taskId).Find(&assignees).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No assignees found for the task"})
		return
	}

	// Fetch user details for each assignee
	assigneeDetails := make([]constant.SimpleUser, 0, len(assignees))
	for _, assignee := range assignees {
		var user model.ScholarizeUser
		if err := database.Db.Where("user_id = ?", assignee.UserID).First(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignee details"})
			return
		}
		assigneeDetails = append(assigneeDetails, constant.SimpleUser{
			UserID:         user.UserID,
			UserName:       user.UserName,
			UserProfileImg: user.UserProfileImg,
			UserEmail:      user.UserEmail,
		})
	}

	// Return the list of assignees as JSON
	c.JSON(http.StatusOK, gin.H{
		"task_id":   taskId,
		"assignees": assigneeDetails,
	})
}

// ADDED NOTIFICATION
// Assign assignees to task
func HandleAssignAssigneeToTask(c *gin.Context) {
	// Get task id from post form
	taskId, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task id"})
		return
	}

	// Get array of assignee ids from post form
	assigneeIdsStr := c.PostFormArray("assignee_ids")
	assigneeIds := make([]int, 0, len(assigneeIdsStr))
	for _, idStr := range assigneeIdsStr {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid assignee ID format"})
			return
		}
		assigneeIds = append(assigneeIds, id)
	}

	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Get the current task
	var task model.Task
	if err := tx.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Fetch current assignees
	var currentAssignees []model.TaskAssignee
	if err := tx.Where("task_id = ?", taskId).Find(&currentAssignees).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch current assignees"})
		return
	}

	// Map to track existing assignees for easy lookup
	currentAssigneeMap := make(map[int]bool)
	for _, assignee := range currentAssignees {
		currentAssigneeMap[assignee.UserID] = true
	}

	// Track changes to determine if updates are necessary
	updatesMade := false

	// Check for new assignees and add them
	for _, newAssigneeId := range assigneeIds {
		if !currentAssigneeMap[newAssigneeId] {
			if !middleware.IsUserCollabOwnerOrMember(newAssigneeId, task.CollabID) {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Assignee does not belong to the collaboration"})
				return
			}
			newAssignee := model.TaskAssignee{TaskID: taskId, UserID: newAssigneeId}
			if err := tx.Create(&newAssignee).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add new assignee"})
				return
			}
			updatesMade = true
		}
	}

	// Remove assignees that are no longer selected
	for _, currentAssignee := range currentAssignees {
		if !contains(assigneeIds, currentAssignee.UserID) {
			if err := tx.Delete(&model.TaskAssignee{}, "task_id = ? AND user_id = ?", taskId, currentAssignee.UserID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove assignee"})
				return
			}
			updatesMade = true
		}
	}

	// Commit or respond with no changes
	if updatesMade {
		if err := tx.Commit().Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
			return
		}

		// Insert notification
		err = InsertNotificationForAssignee(task.CollabID, task.TaskTitle, assigneeIds, c.MustGet("userID").(int))
		if err != nil {
			fmt.Println("Error inserting notification for assignee:", err)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Assignees updated successfully"})
	} else {
		tx.Rollback() // No changes so rollback any preliminary reads
		c.JSON(http.StatusOK, gin.H{"message": "No updates made"})
	}
}

// Helper function to check if a slice contains an integer
func contains(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Handle Create subtask
func HandleCreateSubtask(c *gin.Context) {
	// Get task id from post form
	taskId, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid task id"})
		return
	}

	// Get subtask title from post form
	subtaskTitle := c.PostForm("subtask_title")
	if subtaskTitle == "" {
		c.JSON(400, gin.H{"error": "Subtask title cannot be empty"})
		return
	}

	// Create new subtask record
	subtaskRecord := model.Subtask{
		SubtaskTitle: subtaskTitle,
		TaskID:       taskId,
	}

	// Save the subtask record
	err = database.Db.Create(&subtaskRecord).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error adding subtask"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Subtask added successfully",
		"subtask": subtaskRecord,
	})
}

// Handle update subtask name
func HandleUpdateSubtaskName(c *gin.Context) {
	// Get subtask id from post form
	subtaskId, err := strconv.Atoi(c.PostForm("subtask_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid subtask id"})
		return
	}

	// Get subtask title from post form
	subtaskTitle := c.PostForm("subtask_title")
	if subtaskTitle == "" {
		c.JSON(400, gin.H{"error": "Subtask title cannot be empty"})
		return
	}

	// Get the current subtask
	var subtask model.Subtask
	if err := database.Db.Where("subtask_id = ?", subtaskId).First(&subtask).Error; err != nil {
		c.JSON(500, gin.H{"error": "Subtask not found"})
		return
	}

	// Check if title is the same
	if subtask.SubtaskTitle == subtaskTitle {
		c.JSON(400, gin.H{"error": "Subtask title is already the same, no updates are made"})
		return
	}

	// Update subtask title
	subtask.SubtaskTitle = subtaskTitle
	err = database.Db.Save(&subtask).Error
	if err != nil {
		c.JSON(500, gin.H{"error": "Error updating subtask title"})
		return
	}

	c.JSON(200, gin.H{"message": "Subtask title updated successfully"})
}

// Handle delete subtask
func HandleDeleteSubtask(c *gin.Context) {
	// Get subtask id from post form
	subtaskId, err := strconv.Atoi(c.PostForm("subtask_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid subtask id"})
		return
	}

	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Get the current subtask
	var subtask model.Subtask
	if err := tx.Where("subtask_id = ?", subtaskId).First(&subtask).Error; err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Subtask not found"})
		return
	}

	// Delete subtask
	errr := deleteSubtask(tx, subtaskId)
	if errr != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Error deleting subtask"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Subtask deleted successfully",
		"subtask": subtask,
	})
}

// Handle get all subtasks of a task
func HandleGetAllSubtasks(c *gin.Context) {
	// Get task id from the query parameter
	taskIdStr := c.Query("task_id")
	taskId, err := strconv.Atoi(taskIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Check if task exist
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Fetch subtasks related to the task
	var subtasks []model.Subtask
	if err := database.Db.Where("task_id = ?", taskId).
		Order("subtask_id ASC").
		Find(&subtasks).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No subtasks found for the task"})
		return
	}

	// Return the list of subtasks as JSON
	c.JSON(http.StatusOK, gin.H{
		"task_id":  taskId,
		"subtasks": subtasks,
	})
}

// Fix TASK COMMENT
// Handle Get all comments of a task
func HandleGetTaskComments(c *gin.Context) {
	// Get task id from the query parameter
	taskIdStr := c.Query("task_id")
	taskId, err := strconv.Atoi(taskIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
		return
	}

	// Check if the task exists
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Fetch comments related to the task
	var comments []model.Comment
	if err := database.Db.Table("comment").
		Joins("JOIN taskcomment ON taskcomment.comment_id = comment.comment_id").
		Where("taskcomment.task_id = ?", taskId).
		Order("comment.commented_at ASC").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Error selecting comments for the task"})
		return
	}

	// Fetch user details for each comment
	commentDetails := make([]constant.TaskCommentDetails, 0, len(comments))
	for _, comment := range comments {
		var user model.ScholarizeUser
		if err := database.Db.Where("user_id = ?", comment.UserID).First(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comment details"})
			return
		}
		userDetail := constant.SimpleUser{
			UserID:         user.UserID,
			UserName:       user.UserName,
			UserProfileImg: user.UserProfileImg,
			UserEmail:      user.UserEmail,
		}
		commentDetails = append(commentDetails, constant.TaskCommentDetails{
			CommentID:   comment.CommentID,
			CommentText: comment.CommentText,
			CommentedBy: userDetail,
			CommentedAt: comment.CommentAt.Format("2006-01-02 15:04:05"),
		})
	}

	// Return the list of comments as JSON
	c.JSON(http.StatusOK, gin.H{
		"task_id":  taskId,
		"comments": commentDetails,
	})
}

// Handle Add comment to task
func HandleAddCommentToTask(c *gin.Context) {
	// Get task id from post form
	taskId, err := strconv.Atoi(c.PostForm("task_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid task id"})
		return
	}

	// Get the current task
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		c.JSON(500, gin.H{"error": "Task not found"})
		return
	}

	// Get comment text from post form
	commentText := c.PostForm("comment_text")
	if commentText == "" {
		c.JSON(400, gin.H{"error": "Comment text cannot be empty"})
		return
	}

	// Get user from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(500, gin.H{"error": "User not found"})
		return
	}

	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Create new comment record
	commentRecord := model.Comment{
		CommentText: commentText,
		CommentAt:   time.Now(),
		UserID:      userID.(int),
	}

	// Save the comment record
	err = database.Db.Create(&commentRecord).Error
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Error adding comment"})
		return
	}

	// Get comment id
	commentId := commentRecord.CommentID

	// Create new task comment record
	taskCommentRecord := model.TaskComment{
		TaskID:    taskId,
		CommentID: commentId,
	}

	// Save the task comment record
	err = database.Db.Create(&taskCommentRecord).Error
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Error adding task comment"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Comment added successfully",
		"comment": commentRecord,
	})
}

// Handle Get all comments of a subtask
func HandleGetSubtaskComments(c *gin.Context) {
	// Get task id from the query parameter
	subtaskIdStr := c.Query("subtask_id")
	subtaskId, err := strconv.Atoi(subtaskIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	// Check if the subtask exists
	var subtask model.Subtask
	if err := database.Db.Where("subtask_id = ?", subtaskId).First(&subtask).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subtask not found"})
		return
	}

	// Fetch comments related to the task
	var comments []model.Comment
	if err := database.Db.Table("comment").
		Joins("JOIN subtaskcomment ON subtaskcomment.comment_id = comment.comment_id").
		Where("subtaskcomment.subtask_id = ?", subtaskId).
		Order("comment.commented_at ASC").
		Find(&comments).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Error selecting comments for the task"})
		return
	}

	// Fetch user details for each comment
	commentDetails := make([]constant.TaskCommentDetails, 0, len(comments))
	for _, comment := range comments {
		var user model.ScholarizeUser
		if err := database.Db.Where("user_id = ?", comment.UserID).First(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch comment details"})
			return
		}
		userDetail := constant.SimpleUser{
			UserID:         user.UserID,
			UserName:       user.UserName,
			UserProfileImg: user.UserProfileImg,
			UserEmail:      user.UserEmail,
		}
		commentDetails = append(commentDetails, constant.TaskCommentDetails{
			CommentID:   comment.CommentID,
			CommentText: comment.CommentText,
			CommentedBy: userDetail,
			CommentedAt: comment.CommentAt.Format("2006-01-02 15:04:05"),
		})
	}

	// Return the list of comments as JSON
	c.JSON(http.StatusOK, gin.H{
		"task_id":    subtask.TaskID,
		"subtask_id": subtaskId,
		"comments":   commentDetails,
	})
}

// Handle Add comment to a subtask
func HandleAddCommentToSubtask(c *gin.Context) {
	// Get task id from post form
	subtaskId, err := strconv.Atoi(c.PostForm("subtask_id"))
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid subtask id"})
		return
	}

	// Check if the subtask exists
	var subtask model.Subtask
	if err := database.Db.Where("subtask_id = ?", subtaskId).First(&subtask).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Subtask not found"})
		return
	}

	// Get comment text from post form
	commentText := c.PostForm("comment_text")
	if commentText == "" {
		c.JSON(400, gin.H{"error": "Comment text cannot be empty"})
		return
	}

	// Get user from context
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(500, gin.H{"error": "User not found"})
		return
	}

	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Create new comment record
	commentRecord := model.Comment{
		CommentText: commentText,
		CommentAt:   time.Now(),
		UserID:      userID.(int),
	}

	// Save the comment record
	err = database.Db.Create(&commentRecord).Error
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Error adding comment"})
		return
	}

	// Get comment id
	commentId := commentRecord.CommentID

	// Create new task comment record
	taskCommentRecord := model.SubtaskComment{
		SubtaskID: subtaskId,
		CommentID: commentId,
	}

	// Save the task comment record
	err = database.Db.Create(&taskCommentRecord).Error
	if err != nil {
		tx.Rollback()
		c.JSON(500, gin.H{"error": "Error adding task comment"})
		return
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(200, gin.H{
		"message": "Comment added successfully",
		"comment": commentRecord,
	})
}

// Check if subtask belongs to task
func isSubtaskBelongsToTask(subtaskId, taskId int) bool {
	var subtask model.Subtask
	if err := database.Db.Where("subtask_id = ?", subtaskId).First(&subtask).Error; err != nil {
		return false
	}
	fmt.Println("Subtask: ", subtaskId, "Task", taskId, subtask.TaskID == taskId)

	return subtask.TaskID == taskId
}

// Check if task belongs to collab
func isTaskBelongsToCollab(taskId, collabId int) bool {
	var task model.Task
	if err := database.Db.Where("task_id = ?", taskId).First(&task).Error; err != nil {
		return false
	}
	fmt.Println("Task: ", taskId, "Collab", collabId, task.CollabID == collabId)
	return task.CollabID == collabId
}

// Delete subtask and its comments
func deleteSubtask(tx *gorm.DB, subtaskID int) error {
	// Get the current subtask
	var subtask model.Subtask
	if err := tx.Where("subtask_id = ?", subtaskID).First(&subtask).Error; err != nil {
		return err
	}

	// Delete subtask
	if err := tx.Where("subtask_id = ?", subtaskID).Delete(&model.Subtask{}).Error; err != nil {
		return err
	}

	// Check if subtask has comments and delete them
	var subtaskComments []model.SubtaskComment
	if err := tx.Where("subtask_id = ?", subtaskID).Find(&subtaskComments).Error; err != nil {
		return err
	}
	for _, subtaskComment := range subtaskComments {
		if err := tx.Where("comment_id = ?", subtaskComment.CommentID).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
	}

	// Delete subtask comments
	if err := tx.Where("subtask_id = ?", subtaskID).Delete(&model.SubtaskComment{}).Error; err != nil {
		return err
	}

	fmt.Printf("Subtask %d deleted from %d\n", subtaskID, subtask.TaskID)
	return nil
}

// Delete task, it subtasks, its status and its comments
func deleteTask(tx *gorm.DB, taskID int) error {
	// Get the current task
	var task model.Task
	if err := tx.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		return err
	}

	// Delete task
	if err := tx.Where("task_id = ?", taskID).Delete(&model.Task{}).Error; err != nil {
		return err
	}

	// Check if task has subtasks and delete them
	var subtasks []model.Subtask
	if err := tx.Where("task_id = ?", taskID).Find(&subtasks).Error; err != nil {
		return err
	}

	// Delete subtasks
	for _, subtask := range subtasks {
		if err := deleteSubtask(tx, subtask.SubtaskID); err != nil {
			return err
		}
	}

	// Delete task status
	if err := tx.Where("task_id = ?", taskID).Delete(&model.StatusTask{}).Error; err != nil {
		return err
	}

	// Check if task has comments and delete them
	var taskComments []model.TaskComment
	if err := tx.Where("task_id = ?", taskID).Find(&taskComments).Error; err != nil {
		return err
	}
	for _, taskComment := range taskComments {
		if err := tx.Where("comment_id = ?", taskComment.CommentID).Delete(&model.Comment{}).Error; err != nil {
			return err
		}
	}

	// Delete task comments
	if err := tx.Where("task_id = ?", taskID).Delete(&model.TaskComment{}).Error; err != nil {
		return err
	}

	// Delete task assignees
	if err := tx.Where("task_id = ?", taskID).Delete(&model.TaskAssignee{}).Error; err != nil {
		return err
	}

	fmt.Printf("Task %d deleted from collab %d\n", taskID, task.CollabID)
	return nil
}

// Delete all tasks and subtasks of a collab
func deleteAllTasksAndSubtasksOfCollab(tx *gorm.DB, collabID int) error {
	// Get all tasks of the collab
	var tasks []model.Task
	if err := tx.Where("collab_id = ?", collabID).Find(&tasks).Error; err != nil {
		return err
	}

	// Delete all tasks and their subtasks
	for _, task := range tasks {
		if err := deleteTask(tx, task.TaskID); err != nil {
			return err
		}
	}

	fmt.Printf("All tasks and subtasks of collab %d deleted\n", collabID)
	return nil
}
