package administrator

import (
	"fmt"
	"net/http"
	"root/constant"
	"root/database"
	"root/model"
	"root/permission"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func HandleGetResearchPaperList(c *gin.Context) {
	// Get the page and count parameters from the query string
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))
	searchTerm := strings.ToLower(c.DefaultQuery("search", ""))
	offset := (page - 1) * count

	query := database.Db.Model(&model.ResearchPaper{}).
		Where("research_paper_status IN (?)", []string{"published", "unpublished"}).
		Order("CASE WHEN research_paper_status = 'published' THEN 1 ELSE 2 END, published_at DESC")

	var totalPaper int64
	query.Count(&totalPaper)

	// Apply case-insensitive filtering if a search term is provided
	if searchTerm != "" {
		searchTermPattern := "%" + searchTerm + "%"
		query = query.Where("LOWER(research_title) LIKE ? OR LOWER(author) LIKE ?", searchTermPattern, searchTermPattern)
	}

	var researchPaper []model.ResearchPaper
	query.Limit(count).Offset(offset).Find(&researchPaper)

	// Get Information of Research Paper
	var researchPaperInfo []constant.AdminResearchPaperList
	for _, paper := range researchPaper {
		var paperInfo constant.AdminResearchPaperList
		paperInfo.ResearchPaperID = paper.ResearchPaperID
		paperInfo.Title = paper.ResearchTitle
		paperInfo.Author = paper.Author

		// Get the user information of the paper
		var userInfo constant.SimpleUser
		userData := permission.GetUserById(paper.UserID)
		userInfo.UserID = userData.UserID
		userInfo.UserName = userData.UserName
		userInfo.UserProfileImg = userData.UserProfileImg
		userInfo.UserEmail = userData.UserEmail

		paperInfo.UserInfo = userInfo
		paperInfo.ResearchPaperStatus = paper.ResearchPaperStatus
		paperInfo.PublishedAt = paper.PublishedAt.Format("2006-01-02 15:04:05")

		// Get the department information of the paper
		paperInfo.DepartmentInfo = getDepartmentsByResearchPaperID(paper.ResearchPaperID)

		researchPaperInfo = append(researchPaperInfo, paperInfo)
	}

	total := int64(len(researchPaperInfo))

	c.JSON(http.StatusOK, gin.H{
		"research_paper_list":  researchPaperInfo,
		"total_result":         total,
		"total_research_paper": totalPaper,
	})
}

func getDepartmentsByResearchPaperID(researchPaperID int) []constant.DepartmentInfoForPaper {
	var results []constant.DepartmentInfoForPaper

	err := database.Db.Model(&model.ResearchPaperDepartment{}).
		Select("department.department_id, department.department_name, department.department_tag, department.department_color").
		Joins("JOIN department ON department.department_id = researchpaperdepartment.department_id").
		Where("researchpaperdepartment.research_paper_id = ?", researchPaperID).
		Group("department.department_id, department.department_name, department.department_tag, department.department_color").
		Find(&results).Error

	if err != nil {
		fmt.Println("Error fetching department info for research paper:", err)
		return nil
	}
	return results
}

func HandleUpdateResearchPaper(c *gin.Context) {
	paperIDStr := c.PostForm("research_paper_id")
	if paperIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paper ID is required"})
		return
	}
	paperID, _ := strconv.Atoi(paperIDStr)

	// Fetch current status
	var paper model.ResearchPaper
	if err := database.Db.Select("research_paper_status").Where("research_paper_id = ?", paperID).First(&paper).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch paper status"})
		return
	}

	// Check if the paper status is not published or not unpublished
	if paper.ResearchPaperStatus != "published" && paper.ResearchPaperStatus != "unpublished" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid paper status"})
		return
	}

	// Toggle status
	newStatus := "published"
	if paper.ResearchPaperStatus == "published" {
		newStatus = "unpublished"
	}

	// Update the status of paper
	if err := database.Db.Model(&model.ResearchPaper{}).Where("research_paper_id = ?", paperID).Update("research_paper_status", newStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper status updated successfully", "new_status": newStatus})
}

func HandleUpdateResearchPaperTitle(c *gin.Context) {

	paperIDStr := c.PostForm("research_paper_id")
	if paperIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paper ID is required"})
		return
	}
	paperID, _ := strconv.Atoi(paperIDStr)

	newTitle := c.PostForm("new_title")
	if newTitle == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New title is required"})
		return
	}

	// Check if the paper exists
	var paper model.ResearchPaper
	if err := database.Db.Select("research_paper_id", "research_title").Where("research_paper_id = ?", paperID).First(&paper).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paper not found"})
		return
	}

	// Check if the title already exist and it is not the same paper
	var existingPaper model.ResearchPaper
	if err := database.Db.Select("research_paper_id").Where("research_title = ?", newTitle).First(&existingPaper).Error; err == nil && existingPaper.ResearchPaperID != paperID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Title already exists"})
		return
	}
	old_paper_title := paper.ResearchTitle
	fmt.Println("Old Paper Title:", old_paper_title)
	if err := database.Db.Table("langchain_pg_collection").Where("name = ?", old_paper_title).Update("name", newTitle).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Collection not found"})
		return
	}
	// Update the title of paper
	if err := database.Db.Model(&model.ResearchPaper{}).Where("research_paper_id = ?", paperID).Update("research_title", newTitle).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update title"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper title updated successfully"})
}

func HandleUpdateResearchPaperDate(c *gin.Context) {
	paperIDStr := c.PostForm("research_paper_id")
	if paperIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Paper ID is required"})
		return
	}
	paperID, _ := strconv.Atoi(paperIDStr)

	newDate := c.PostForm("new_date")
	if newDate == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New date is required"})
		return
	}

	// Parse the date part
	datePart, err := time.Parse("2006-01-02", newDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	// Get the current time
	now := time.Now()

	// Combine the parsed date with the current time
	combinedDateTime := time.Date(datePart.Year(), datePart.Month(), datePart.Day(), now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())

	// Format the combined date-time for PostgreSQL
	formattedDateTime := combinedDateTime.Format("2006-01-02 15:04:05")

	// Update the date of paper
	if err := database.Db.Model(&model.ResearchPaper{}).Where("research_paper_id = ?", paperID).Update("published_at", formattedDateTime).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update date"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Paper date updated successfully"})
}
