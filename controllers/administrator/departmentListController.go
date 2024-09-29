package administrator

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"root/constant"
	"root/database"
	"root/model"
	"root/permission"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const departmentNotExistError = "Department does not exist"

// Get all departments and their HODs
func GetDepartmentsList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	searchTerm := strings.ToLower(c.DefaultQuery("search", ""))
	offset := (page - 1) * count

	var departments []model.Department

	var totalDep int64
	database.Db.Model(&departments).Count(&totalDep)

	query := database.Db.Model(&model.Department{})

	// Apply case-insensitive filtering if a search term is provided
	if searchTerm != "" {
		searchTermPattern := "%" + searchTerm + "%"
		query = query.Where("LOWER(department_name) LIKE ? OR LOWER(department_tag) LIKE ?", searchTermPattern, searchTermPattern)
	}

	// Apply pagination after counting
	// Order by department name and then department status
	query = query.Order("CASE WHEN department_status = true THEN 1 ELSE 2 END, department_name ASC").
		Offset(offset).Limit(count)

	// Execute the query with pagination and store the result in departments
	query.Find(&departments)

	// Fetch all users with HOD role who are not assigned to any department
	var eligibleHODUsers []model.ScholarizeUser
	if err := database.Db.Raw(`
        SELECT * FROM scholarize_user
        WHERE user_id IN (
            SELECT user_id FROM userrole WHERE role_id = (SELECT role_id FROM role WHERE role_name = 'HOD')
        ) AND user_id NOT IN (
            SELECT user_id FROM departmenthead
        )
    `).Scan(&eligibleHODUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var departmentInfos []constant.AdminDepartmentInfo
	for _, dept := range departments {
		var info constant.AdminDepartmentInfo
		info.Department = constant.DepartmentInfo{
			DepartmentID:     dept.DepartmentID,
			DepartmentName:   dept.DepartmentName,
			DepartmentTag:    dept.DepartmentTag,
			DepartmentColor:  dept.DepartmentColor,
			DepartmentStatus: dept.DepartmentStatus,
		}

		// Find all HODs for this department
		var departmentHeads []model.DepartmentHead
		if err := database.Db.Where("department_id = ?", dept.DepartmentID).Find(&departmentHeads).Error; err != nil {
			continue
		}

		for _, dh := range departmentHeads {
			var hod model.ScholarizeUser
			if err := database.Db.Where("user_id = ?", dh.UserID).First(&hod).Error; err != nil {
				continue
			}

			simpleHod := constant.SimpleUser{
				UserID:         hod.UserID,
				UserName:       hod.UserName,
				UserProfileImg: hod.UserProfileImg,
				UserEmail:      hod.UserEmail,
			}
			info.DepartmentHODs = append(info.DepartmentHODs, simpleHod)
		}

		// Map eligible HOD users to SimpleUser struct
		var simpleEligibleHODUsers []constant.SimpleUser
		for _, user := range eligibleHODUsers {
			simpleUser := constant.SimpleUser{
				UserID:         user.UserID,
				UserName:       user.UserName,
				UserProfileImg: user.UserProfileImg,
				UserEmail:      user.UserEmail,
			}
			simpleEligibleHODUsers = append(simpleEligibleHODUsers, simpleUser)
		}

		info.EligibleHODUsers = simpleEligibleHODUsers
		departmentInfos = append(departmentInfos, info)
	}

	// Count total departments after filtering
	totalResult := int64(len(departmentInfos))

	// Return JSON response
	c.JSON(http.StatusOK, gin.H{
		"departments":      departmentInfos,
		"total_result":     totalResult,
		"total_department": totalDep,
	})
}

// Add a new department
func HandleAddDepartment(c *gin.Context) {
	// Get department details from the request post form
	departmentName := c.PostForm("department_name")
	departmentTag := c.PostForm("department_tag")
	departmentColor := c.PostForm("department_color")
	departmentStatus := c.PostForm("department_status")
	departmentStatusBool, _ := strconv.ParseBool(departmentStatus)
	hodUserIds := c.PostFormArray("hod_user_ids")

	// Basic input validation
	if departmentName == "" || departmentTag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department name and tag are required"})
		return
	}

	if departmentColor == "" {
		departmentColor = "#333333" // Default color
	}

	// Convert department name to lowercase
	departmentNameLower := strings.ToLower(departmentName)

	// Convert department tag to uppercase
	departmentTag = strings.ToUpper(departmentTag)

	// Check if the department name already exists
	var department model.Department
	if err := database.Db.Where("LOWER(department_name) = ?", departmentNameLower).First(&department).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department name already exists"})
		return
	}

	// Check if the department tag already exists
	if err := database.Db.Where("UPPER(department_tag) = ?", departmentTag).First(&department).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department tag already exists"})
		return
	}

	// Create and save the new department
	newDepartment := model.Department{
		DepartmentName:   departmentName,
		DepartmentTag:    departmentTag,
		DepartmentColor:  departmentColor,
		DepartmentStatus: departmentStatusBool,
	}

	if err := database.Db.Create(&newDepartment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Assign HODs to the new department only if HOD user IDs are provided
	if len(hodUserIds) > 0 {
		if err := checkUsersExistWithHODRole(hodUserIds, c); err != nil {
			// If error in checking user existence or roles, roll back department creation
			database.Db.Delete(&newDepartment)
			return // Error response handled within checkUsersExistWithHODRole
		}

		for _, userId := range hodUserIds {
			userIdInt, err := strconv.Atoi(userId)
			if err != nil {
				// Roll back department creation in case of error
				database.Db.Delete(&newDepartment)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
				return
			}

			departmentHead := model.DepartmentHead{
				DepartmentID: newDepartment.DepartmentID,
				UserID:       userIdInt,
			}

			if err := database.Db.Create(&departmentHead).Error; err != nil {
				// Roll back department creation in case of error
				database.Db.Delete(&newDepartment)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Department added successfully!",
		"department_id":     newDepartment.DepartmentID,
		"department_name":   newDepartment.DepartmentName,
		"department_tag":    newDepartment.DepartmentTag,
		"department_color":  newDepartment.DepartmentColor,
		"department_status": newDepartment.DepartmentStatus,
		"hod_user_ids":      hodUserIds,
	})
}

// Update department details
func HandleUpdateDepartment(c *gin.Context) {
	departmentID := c.PostForm("department_id")
	departmentIDInt, err := strconv.Atoi(departmentID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID"})
		return
	}

	departmentName := c.PostForm("department_name")
	departmentTag := c.PostForm("department_tag")
	departmentColor := c.PostForm("department_color")
	departmentStatus := c.PostForm("department_status")
	departmentStatusBool, err := strconv.ParseBool(departmentStatus)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department status"})
		return
	}

	hodUserIds := c.PostFormArray("hod_user_ids")

	// Start transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction start failed"})
		return
	}

	// Check if the HOD user IDs are valid
	if len(hodUserIds) > 0 {
		if err := checkUsersExistWithHODRole(hodUserIds, c); err != nil {
			tx.Rollback()
			return
		}
	}

	// Check if the department exists
	if err := checkDepartmentExists(departmentIDInt, c); err != nil {
		tx.Rollback()
		return
	}

	// Check if the department name already exists
	if err := checkDepartmentNameExists(departmentIDInt, departmentName, c); err != nil {
		tx.Rollback()
		return
	}

	// Check if the department tag already exists
	departmentTag = strings.ToUpper(departmentTag)

	var existingDepartment model.Department
	if err := database.Db.Where("UPPER(department_tag) = ? AND department_id != ?", departmentTag, departmentIDInt).First(&existingDepartment).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department tag already exists"})
		tx.Rollback()
		return
	}

	// Fetch the current department details
	var currentDepartment model.Department
	if err := database.Db.Where("department_id = ?", departmentIDInt).First(&currentDepartment).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": departmentNotExistError})
		tx.Rollback()
		return
	}

	// Only update if there are changes
	if departmentName != currentDepartment.DepartmentName ||
		departmentTag != currentDepartment.DepartmentTag ||
		departmentColor != currentDepartment.DepartmentColor ||
		departmentStatusBool != currentDepartment.DepartmentStatus {

		if err := updateDepartmentDetails(departmentIDInt, departmentName, departmentTag, departmentColor, departmentStatusBool); err != nil {
			tx.Rollback()
			return
		}
	}

	// Check if the HOD user IDs have changed
	var currentDepartmentHeads []model.DepartmentHead
	database.Db.Where("department_id = ?", departmentIDInt).Find(&currentDepartmentHeads)

	var currentHodUserIds []string
	for _, dh := range currentDepartmentHeads {
		currentHodUserIds = append(currentHodUserIds, strconv.Itoa(dh.UserID))
	}

	if !reflect.DeepEqual(hodUserIds, currentHodUserIds) {
		// Assign HODs to the department
		if err := assignHODsToDepartment(tx, departmentIDInt, hodUserIds, c); err != nil {
			tx.Rollback()
			return
		}
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Department updated successfully",
		"new_department_info": constant.DepartmentInfo{
			DepartmentID:     departmentIDInt,
			DepartmentName:   departmentName,
			DepartmentTag:    departmentTag,
			DepartmentColor:  departmentColor,
			DepartmentStatus: departmentStatusBool,
		},
		"new_hod_user_info": hodUserIds,
	})
}

// checkDepartmentExists checks if a department exists in the database
func checkDepartmentExists(departmentID int, c *gin.Context) error {
	var department model.Department
	if err := database.Db.Where("department_id = ?", departmentID).First(&department).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": departmentNotExistError})
		return err
	}
	return nil
}

// checkDepartmentNameExists checks if a department name already exists in the database
func checkDepartmentNameExists(departmentID int, departmentName string, c *gin.Context) error {
	// Convert department name to lowercase
	departmentName = strings.ToLower(departmentName)

	var existingDepartment model.Department
	if err := database.Db.Where("LOWER(department_name) = ? AND department_id != ?", departmentName, departmentID).First(&existingDepartment).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department name already exists"})
		return errors.New("department name already exists")
	}
	return nil
}

// updateDepartmentDetails updates the department details in the database
func updateDepartmentDetails(departmentID int, departmentName string, departmentTag string, departmentColor string, departmentStatus bool) error {
	var department model.Department
	if err := database.Db.Where("department_id = ?", departmentID).First(&department).Error; err != nil {
		return err
	}

	department.DepartmentName = departmentName
	department.DepartmentTag = departmentTag
	department.DepartmentColor = departmentColor
	department.DepartmentStatus = departmentStatus

	if err := database.Db.Save(&department).Error; err != nil {
		return err
	}

	return nil
}

// assignHODsToDepartment assigns HODs to the department in the database
func assignHODsToDepartment(tx *gorm.DB, departmentID int, hodUserIds []string, c *gin.Context) error {
	var newHodUserIdsInt []int
	for _, userId := range hodUserIds {
		userIdInt, err := strconv.Atoi(userId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return err
		}
		newHodUserIdsInt = append(newHodUserIdsInt, userIdInt)
	}

	var currentDepartmentHeads []model.DepartmentHead
	if err := tx.Where("department_id = ?", departmentID).Find(&currentDepartmentHeads).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	// Remove department heads not in the new list
	for _, currentDH := range currentDepartmentHeads {
		if !contains(newHodUserIdsInt, currentDH.UserID) {
			if err := tx.Delete(&model.DepartmentHead{}, "department_id = ? AND user_id = ?", departmentID, currentDH.UserID).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return err
			}
		}
	}

	// Add new department heads if they are not already HODs in other departments
	for _, userIdInt := range newHodUserIdsInt {
		if !isAlreadyHOD(userIdInt, currentDepartmentHeads) {
			var existingHOD model.DepartmentHead
			if err := tx.Where("user_id = ? AND department_id != ?", userIdInt, departmentID).First(&existingHOD).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User %d is already a HOD in another department", userIdInt)})
				return errors.New("user is already a HOD in another department")
			}

			departmentHead := model.DepartmentHead{
				DepartmentID: departmentID,
				UserID:       userIdInt,
			}
			if err := tx.Create(&departmentHead).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return err
			}
		}
	}

	return nil
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func isAlreadyHOD(userID int, departmentHeads []model.DepartmentHead) bool {
	for _, dh := range departmentHeads {
		if dh.UserID == userID {
			return true
		}
	}
	return false
}

func checkUsersExistWithHODRole(hodUserIds []string, c *gin.Context) error {
	for _, userId := range hodUserIds {
		userIdInt, err := strconv.Atoi(userId)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return err
		}

		// Use the function to get the user by ID
		user := permission.GetUserById(userIdInt)
		if user == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User %d does not exist", userIdInt)})
			return errors.New("user does not exist")
		}

		// Use the function to get the user's role
		role := permission.GetFrontPanelUserRoleName(user.UserID)

		if role != "HOD" {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("User %d is not a HOD", userIdInt)})
			return errors.New("user is not a HOD")
		}
	}
	return nil
}
