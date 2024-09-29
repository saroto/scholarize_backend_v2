package administrator

import (
	"net/http"
	"root/database"
	"root/model"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func HandleGetResearchTypeList(c *gin.Context) {
	// Get the page and count parameters from the query string
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	searchTerm := strings.ToLower(c.DefaultQuery("search", ""))
	offset := (page - 1) * count

	query := database.Db.Model(&model.ResearchType{}).
		Order("CASE WHEN research_type_status = true THEN 1 ELSE 2 END,research_type_name ASC")

	var totalType int64
	query.Count(&totalType)

	// Apply case-insensitive filtering if a search term is provided
	if searchTerm != "" {
		searchTermPattern := "%" + searchTerm + "%"
		query = query.Where("LOWER(research_type_name) LIKE ?", searchTermPattern)
	}

	var researchType []model.ResearchType
	query.Limit(count).Offset(offset).Find(&researchType)

	// Get Information of Research Type
	var researchTypeInfo []model.ResearchType
	for _, t := range researchType {
		var typeInfo model.ResearchType
		typeInfo.ResearchTypeID = t.ResearchTypeID
		typeInfo.ResearchTypeName = t.ResearchTypeName
		typeInfo.ResearchTypeStatus = t.ResearchTypeStatus

		researchTypeInfo = append(researchTypeInfo, typeInfo)
	}

	totalResult := int64(len(researchTypeInfo))

	c.JSON(http.StatusOK, gin.H{
		"types":        researchTypeInfo,
		"total_result": totalResult,
		"total_type":   totalType,
	})
}

func HandleAddResearchType(c *gin.Context) {
	// Get research type Id from the request
	researchTypeName := c.PostForm("research_type_name")

	// Check if the research type name is empty
	if researchTypeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type name is required"})
		return
	}

	// Convert name to lower
	researchTypeNameLower := strings.ToLower(researchTypeName)

	// Check if the research type name already exists
	var count int64
	database.Db.Model(&model.ResearchType{}).Where("LOWER(research_type_name) = ?", researchTypeNameLower).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type name already exists"})
		return
	}

	// Create a new research type
	researchType := model.ResearchType{
		ResearchTypeName: researchTypeName,
	}

	// Save the research type to the database
	err := database.Db.Create(&researchType).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding research type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Research type added successfully",
		"research_type": researchType,
	})
}

func HandleUpdateResearchType(c *gin.Context) {
	// Get the research type ID from the request
	researchTypeIDStr := c.PostForm("research_type_id")

	// Check if the research type ID is empty
	if researchTypeIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type ID is required"})
		return
	}

	// Convert the research type ID to an integer
	researchTypeID, _ := strconv.Atoi(researchTypeIDStr)

	// Fetch the current research type data from the database
	var researchType model.ResearchType
	if err := database.Db.Where("research_type_id = ?", researchTypeID).First(&researchType).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type not found"})
		return
	}

	// Get the research type name and status from the request
	researchTypeName := c.PostForm("research_type_name")
	researchTypeStatus := c.PostForm("research_type_status")

	// Check if the research type name is empty
	if researchTypeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type name is required"})
		return
	}

	// Check if the research type status is empty
	if researchTypeStatus == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type status is required"})
		return
	}

	// Convert name to lower for checking
	researchTypeNameLower := strings.ToLower(researchTypeName)

	// Check if the research type name already exists
	var count int64
	database.Db.Model(&model.ResearchType{}).Where("LOWER(research_type_name) = ? AND research_type_id != ?", researchTypeNameLower, researchTypeID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type name already exists"})
		return
	}

	// Convert research type status to bool
	researchTypeStatusBool, _ := strconv.ParseBool(researchTypeStatus)

	// Update the research type name and status
	if err := database.Db.Model(&model.ResearchType{}).Where("research_type_id = ?", researchTypeID).Updates(map[string]interface{}{
		"research_type_name":   researchTypeName,
		"research_type_status": researchTypeStatusBool,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating research type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Research type updated successfully",
		"research_type": model.ResearchType{
			ResearchTypeID:     researchTypeID,
			ResearchTypeName:   researchTypeName,
			ResearchTypeStatus: researchTypeStatusBool,
		},
	})
}
