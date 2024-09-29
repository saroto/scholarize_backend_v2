package repository

import (
	"fmt"
	"net/http"
	"root/config"
	"root/constant"
	"root/controllers/notification"
	"root/database"
	"root/model"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Get published research papers ids by user id
func GetPublishedResearchPapersByUserId(userId int) ([]int, error) {
	var researchPapers []model.ResearchPaper
	result := database.Db.Model(&model.ResearchPaper{}).
		Where("user_id = ?", userId).
		Where("LOWER(research_paper_status) = ?", "published").
		Order("published_at desc").
		Find(&researchPapers)

	// Check if no paper
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	// Extract the research paper IDs
	var paperIds []int
	for _, paper := range researchPapers {
		paperIds = append(paperIds, paper.ResearchPaperID)
	}

	return paperIds, nil
}

// Get awaiting research papers ids by user id
func GetAwaitingResearchPapersByUserId(userId int) ([]int, error) {
	var researchPapers []model.ResearchPaper
	result := database.Db.Model(&model.ResearchPaper{}).
		Where("user_id = ?", userId).
		Where("LOWER(research_paper_status) = ?", "awaiting").
		Find(&researchPapers)

	// Check if no paper
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	// Extract the research paper IDs
	var paperIds []int
	for _, paper := range researchPapers {
		paperIds = append(paperIds, paper.ResearchPaperID)
	}

	return paperIds, nil
}

// Get rejected research papers ids by user id
func GetRejectedResearchPapersByUserId(userId int) ([]int, error) {
	var researchPapers []model.ResearchPaper
	result := database.Db.Model(&model.ResearchPaper{}).
		Where("user_id = ?", userId).
		Where("LOWER(research_paper_status) = ?", "rejected").
		Find(&researchPapers)

	// Check if no paper
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	// Extract the research paper IDs
	var paperIds []int
	for _, paper := range researchPapers {
		paperIds = append(paperIds, paper.ResearchPaperID)
	}

	return paperIds, nil
}

// Get title of research paper
func GetResearchPaperTitle(paperId int) (string, error) {
	var paper model.ResearchPaper
	result := database.Db.First(&paper, paperId)
	if result.Error != nil {
		return "", result.Error
	}
	return paper.ResearchTitle, nil
}

// Get public_id of research paper
func GetResearchPaperPublicId(paperId int) (string, error) {
	var paper model.ResearchPaper
	result := database.Db.First(&paper, paperId)
	if result.Error != nil {
		return "", result.Error
	}
	return paper.PublicID, nil
}

// Check if the user is uploader of the paper
func IsUserUploader(userId, paperId int) (bool, error) {
	var paper model.ResearchPaper
	result := database.Db.First(&paper, paperId)
	if result.Error != nil {
		return false, result.Error
	}
	return paper.UserID == userId, nil
}

// Handle query research papers on my published
func HandleDisplayMyPublishedResearchPapers(c *gin.Context) {
	// Get user id from the context
	userId := c.MustGet("userID").(int)

	// Get the published research paper ids
	publishedPaperIds, err := GetPublishedResearchPapersByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching published research papers"})
		return
	}

	// Get the awaiting research paper ids
	awaitingPaperIds, err := GetAwaitingResearchPapersByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching awaiting research papers"})
		return
	}

	// Get the rejected research paper ids
	rejectedPaperIds, err := GetRejectedResearchPapersByUserId(userId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching rejected research papers"})
		return
	}

	// Get info of published papers
	var paperInfos []constant.QueryResearchPaperInfo
	for _, paperId := range publishedPaperIds {
		paperInfo, err := GetEachResearchPaperInfoOnQuery(paperId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper info"})
			return
		}
		paperInfos = append(paperInfos, paperInfo)
	}

	type ApprovalPaperInfo struct {
		ResearchPaperID int    `json:"research_paper_id"`
		ResearchTitle   string `json:"research_title"`
		PublicID        string `json:"public_id"`
	}

	// Get title of awaiting papers and rejected papers
	var awaitingPapers []ApprovalPaperInfo
	for _, paperId := range awaitingPaperIds {
		title, err := GetResearchPaperTitle(paperId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper title"})
			return
		}
		publicId, err := GetResearchPaperPublicId(paperId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper public_id"})
			return
		}
		awaitingPapers = append(awaitingPapers, ApprovalPaperInfo{
			ResearchPaperID: paperId,
			ResearchTitle:   title,
			PublicID:        publicId,
		})
	}

	var rejectedPapers []ApprovalPaperInfo
	for _, paperId := range rejectedPaperIds {
		title, err := GetResearchPaperTitle(paperId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper title"})
			return
		}
		publicId, err := GetResearchPaperPublicId(paperId)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper public_id"})
			return
		}
		rejectedPapers = append(rejectedPapers, ApprovalPaperInfo{
			ResearchPaperID: paperId,
			ResearchTitle:   title,
			PublicID:        publicId,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"published_papers": paperInfos,
		"awaiting_papers":  awaitingPapers,
		"rejected_papers":  rejectedPapers,
	})
}

// TODO: using public_id
// Handle preview awaiting submission
func HandlePreviewAwaitingPaper(c *gin.Context) {
	// Get paper id from param
	paperIDStr := c.Param("id")
	if paperIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research paper ID not provided"})
		return
	}

	// // Convert paperID to int
	// paperID, err := strconv.Atoi(paperIDStr)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid research paper ID format"})
	// 	return
	// }

	// Get raw paper info by public_id
	var rawPaperInfo model.ResearchPaper
	result := database.Db.Where("public_id = ?", paperIDStr).First(&rawPaperInfo)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching raw research paper info"})
		return
	}

	paperID := rawPaperInfo.ResearchPaperID

	// Check if the paper was uploaded by the user
	userId := c.MustGet("userID").(int)
	if rawPaperInfo.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the uploader of this research paper"})
		return
	}
	fmt.Printf("User ID %d is the uploader of the paper %d\n", userId, paperID)

	// Check if the paper was awaiting approval
	if strings.ToLower(rawPaperInfo.ResearchPaperStatus) != "awaiting" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research paper is not awaiting approval"})
		return
	}

	// Get the paper info
	paper, err := GetEachResearchPaperInfoOnQuery(paperID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper info"})
		return
	}

	// Get fulltext

	var fullText model.Fulltext
	// Find the fulltext with the given ID
	ftResult := database.Db.First(&fullText, paperID)
	if ftResult.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error fetching fulltext",
		})
		return
	}

	// Check if the fulltext was found
	if ftResult.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fulltext not found",
		})
		return
	}

	// Return the full text content if everything is ok
	content := fullText.FulltextContent

	c.JSON(http.StatusOK, gin.H{
		"message":   "Research paper retrieved successfully!",
		"paper":     paper,
		"full_text": content,
		"paperInfo": rawPaperInfo,
	})
}

// TODO: using public_id
// Handle preview rejected submission
func HandlePreviewRejectedPaper(c *gin.Context) {
	// Get paper id from param
	paperIDStr := c.Param("id")
	if paperIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research paper ID not provided"})
		return
	}

	// // Convert paperID to int
	// paperID, err := strconv.Atoi(paperIDStr)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid research paper ID format"})
	// 	return
	// }

	// Get raw paper info
	var rawPaperInfo model.ResearchPaper
	result := database.Db.Where("public_id = ?", paperIDStr).First(&rawPaperInfo)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching raw research paper info"})
		return
	}

	paperID := rawPaperInfo.ResearchPaperID

	// Check if the paper was uploaded by the user
	userId := c.MustGet("userID").(int)
	if rawPaperInfo.UserID != userId {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the uploader of this research paper"})
		return
	}
	fmt.Printf("User ID %d is the uploader of the paper %d\n", userId, paperID)

	// Check if the paper was rejected
	if strings.ToLower(rawPaperInfo.ResearchPaperStatus) != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research paper is not rejected"})
		return
	}

	// Query rejected reason from rawPaperInfo
	rejectedReason := rawPaperInfo.RejectedReason

	// Get the paper info
	paper, err := GetEachResearchPaperInfoOnQuery(paperID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper info"})
		return
	}

	// Get fulltext
	var fullText model.Fulltext
	// Find the fulltext with the given ID
	ftResult := database.Db.First(&fullText, paperID)
	if ftResult.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error fetching fulltext",
		})
		return
	}

	// Check if the fulltext was found
	if ftResult.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fulltext not found",
		})
		return
	}

	// Return the full text content if everything is ok
	content := fullText.FulltextContent

	c.JSON(http.StatusOK, gin.H{
		"message":         "Research paper retrieved successfully!",
		"paper":           paper,
		"full_text":       content,
		"paperInfo":       rawPaperInfo,
		"rejected_reason": rejectedReason,
	})
}

// TODO: using research_paper_id
// Handle resubmit the rejected research paper
func HandleResubmitRejectedPaper(c *gin.Context) {
	// Check if postform is missing
	if c.PostForm("research_paper_id") == "" || c.PostForm("research_title") == "" ||
		c.PostForm("research_type_id") == "" || c.PostForm("abstract") == "" ||
		c.PostForm("tag") == "" || c.PostForm("author") == "" ||
		c.PostForm("department_id") == "" || c.PostForm("full_text") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	// Get research paper ID from POST request
	paperIDStr := c.PostForm("research_paper_id")
	paperID, err := strconv.Atoi(paperIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid research paper ID format"})
		return
	}

	// Check if the user is the uploader of the paper
	userId := c.MustGet("userID").(int)
	isUploader, err := IsUserUploader(userId, paperID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking if user is uploader"})
		return
	}
	if !isUploader {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not the uploader of this research paper"})
		return
	}

	// Check if the paper was rejected
	var paperStatus string
	if err := database.Db.Model(&model.ResearchPaper{}).Where("research_paper_id = ?", paperID).Pluck("research_paper_status", &paperStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper status"})
		return
	}
	if strings.ToLower(paperStatus) != "rejected" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research paper is not rejected"})
		return
	}

	// Get the other fields from POST request
	title := c.PostForm("research_title")

	// Check if the new title is unique, excluding the current paper
	var titleCheck model.ResearchPaper
	if err := database.Db.Where("research_title = ? AND research_paper_id != ?", title, paperID).First(&titleCheck).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research title already exists"})
		return
	}
	researchTypeIdStr := c.PostForm("research_type_id")
	abstract := c.PostForm("abstract")
	tag := c.PostForm("tag")
	author := c.PostForm("author")
	advisor := c.PostForm("advisor")
	if advisor == "" {
		advisor = "None"
	}
	departmentIdStr := c.PostForm("department_id")
	fullText := c.PostForm("full_text")

	// Convert researchTypeId to int
	researchTypeId, _ := strconv.Atoi(researchTypeIdStr)

	// Handle file upload if present
	file, _ := c.FormFile("paper_file")

	laravelURL := config.GetFileServiceURL("storefile_cleantext")
	var uploadResp *constant.LaravelUploadResponse
	if file != nil {
		uploadResp, err = UploadFileAndTextToLaravel(file, fullText, laravelURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error uploading file"})
			return
		}
	}

	// Begin transaction
	tx := database.Db.Begin()

	// Update research paper record
	researchPaper := model.ResearchPaper{
		ResearchTitle:  title,
		ResearchTypeID: researchTypeId,
		Abstract:       abstract,
		Tag:            tag,
		Author:         author,
		Advisor:        advisor,
		SubmittedAt:    time.Now(),
	}

	if file != nil {
		researchPaper.PDFPath = uploadResp.FileName
	}

	if err := tx.Model(&model.ResearchPaper{}).Where("research_paper_id = ?", paperID).Updates(researchPaper).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update research paper"})
		return
	}

	// Update full text and clean text if they have changed
	var existingPaper model.ResearchPaper
	if err := tx.Where("research_paper_id = ?", paperID).First(&existingPaper).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch existing research paper"})
		return
	}

	// Update full text
	if err := tx.Model(&model.Fulltext{}).Where("fulltext_id = ?", existingPaper.FulltextID).Update("fulltext_content", fullText).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update full text"})
		return
	}

	// Update clean text
	if file != nil {
		if err := tx.Model(&model.Cleantext{}).Where("cleantext_id = ?", existingPaper.CleantextID).Update("cleantext_content", uploadResp.OptimizedText).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update clean text"})
			return
		}
	} else {
		// Clean the text only
		cleanTextURL := config.GetFileServiceURL("cleantext")
		cleanedText, err := CleanTextAtLaravel(fullText, cleanTextURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error cleaning text"})
			return
		}

		if err := tx.Model(&model.Cleantext{}).Where("cleantext_id = ?", existingPaper.CleantextID).Update("cleantext_content", cleanedText).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update clean text"})
			return
		}
	}

	// Update department associations
	departmentIds := strings.Split(departmentIdStr, ",")
	// Delete old associations
	if err := tx.Where("research_paper_id = ?", paperID).Delete(&model.ResearchPaperDepartment{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update department associations"})
		return
	}
	// Insert new associations
	for _, idStr := range departmentIds {
		departmentId, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		departmentPaper := model.ResearchPaperDepartment{
			DepartmentID:    departmentId,
			ResearchPaperID: paperID,
		}
		if err := tx.Create(&departmentPaper).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new department association"})
			return
		}

		// Get the department heads by department ID
		departmentHeadIDs := GetDepartmentHeadsByDepartmentID(departmentId)

		// Insert notifications for department heads
		for _, departmentHeadID := range departmentHeadIDs {
			insertErr := notification.InsertResubmitPaperNotification(title, departmentHeadID)
			if insertErr != nil {
				fmt.Println("Failed to insert notification for department head")
			}
		}
	}

	// Update research paper status
	if err := tx.Model(&model.ResearchPaper{}).
		Where("research_paper_id = ?", paperID).
		Update("research_paper_status", "awaiting").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update research paper status"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	// Return response
	c.JSON(http.StatusOK, gin.H{
		"message":   "Research paper resubmitted successfully",
		"paperInfo": researchPaper,
	})
}
