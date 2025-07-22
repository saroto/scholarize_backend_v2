package repository

import (
	"encoding/json"
	"log"
	"net/http"
	"root/constant"
	"root/controllers/notification"
	"root/controllers/queue"
	"root/database"
	"root/model"
	"root/permission"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// get research papers by department and return model Research Paper
func GetSubmissionResearchPapersByDepartment(departmentID int) ([]model.ResearchPaper, error) {
	var papers []model.ResearchPaper
	result := database.Db.Model(&papers).
		Joins("JOIN researchpaperdepartment ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Where("researchpaperdepartment.department_id = ?", departmentID).
		Where("LOWER(research_paper.research_paper_status) = ?", "awaiting").
		Order("research_paper.submitted_at DESC").
		Find(&papers)
	if result.Error != nil {
		return nil, result.Error
	}

	return papers, nil
}

// TODO: using public_id
// Query research paper by department
func QueryResearchPaperByDepartment(c *gin.Context) {
	depID, ok := c.Get("departmentID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department ID not provided"})
		return
	}

	depIDInt, ok := depID.(int) // Type assertion to int
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID format"})
		return
	}

	// Get research papers by department
	papers, err := GetSubmissionResearchPapersByDepartment(depIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research papers"})
		return
	}

	var paperInfos []constant.QueryResearchPaperInfo
	for _, p := range papers {
		paperInfo, err := GetEachResearchPaperInfoOnQuery(p.ResearchPaperID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper info"})
			return
		}
		paperInfos = append(paperInfos, paperInfo)
	}

	totalResearchPaper, _ := GetTotalResearchPapersByDepartment(depIDInt)
	totalAwaiting, _ := GetTotalAwaitingResearchPapersByDepartment(depIDInt)
	totalRejected, _ := GetTotalRejectedResearchPapersByDepartment(depIDInt)

	var department model.Department
	// Get the department
	result := database.Db.First(&department, depIDInt)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching department info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "Research papers retrieved successfully!",
		"department":            department,
		"papers":                paperInfos,
		"total_research_papers": totalResearchPaper,
		"total_awaiting":        totalAwaiting,
		"total_rejected":        totalRejected,
	})
}

// Get total number of research papers by department
func GetTotalResearchPapersByDepartment(departmentID int) (int64, error) {
	var total int64
	result := database.Db.Model(&model.ResearchPaper{}).
		Joins("JOIN researchpaperdepartment ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Where("researchpaperdepartment.department_id = ?", departmentID).
		Count(&total)
	if result.Error != nil {
		return 0, result.Error
	}

	return total, nil
}

// Get total number of research papers that are awaiting approval
func GetTotalAwaitingResearchPapersByDepartment(departmentID int) (int64, error) {
	var total int64
	result := database.Db.Model(&model.ResearchPaper{}).
		Joins("JOIN researchpaperdepartment ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Where("researchpaperdepartment.department_id = ?", departmentID).
		Where("LOWER(research_paper.research_paper_status) = ?", "awaiting").
		Count(&total)
	if result.Error != nil {
		return 0, result.Error
	}

	return total, nil
}

// Get total number of research papers that are in rejected
func GetTotalRejectedResearchPapersByDepartment(departmentID int) (int64, error) {
	var total int64
	result := database.Db.Model(&model.ResearchPaper{}).
		Joins("JOIN researchpaperdepartment ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Where("researchpaperdepartment.department_id = ?", departmentID).
		Where("LOWER(research_paper.research_paper_status) = ?", "rejected").
		Count(&total)
	if result.Error != nil {
		return 0, result.Error
	}

	return total, nil
}

// Get user who publish
func GetSubmitUser(userId int) (constant.UserList, error) {
	var user model.ScholarizeUser
	result := database.Db.Where("user_id = ?", userId).First(&user)
	if result.Error != nil {
		return constant.UserList{}, result.Error
	}

	userRole, _ := permission.GetFrontUserRoleData(user.UserID)

	return constant.UserList{
		UserId:         user.UserID,
		UserName:       user.UserName,
		UserEmail:      user.UserEmail,
		UserProfileImg: user.UserProfileImg,
		UserStatus:     user.UserStatus,
		UserRole:       userRole,
	}, nil
}

// TODO: using public_id
// Preview each paper submission
func HandlePreviewSubmission(c *gin.Context) {
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

	// Get the paper id
	paperID, err := getPaperIDFromPublicID(paperIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid research paper ID format"})
		return
	}

	// Get department ID from context
	depID, ok := c.Get("departmentID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department ID not provided"})
		return
	}

	depIDInt, ok := depID.(int)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID format"})
		return
	}

	// Check if the paper belongs to the department
	if !isPaperBelongsToDepartment(depIDInt, paperID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized! Research paper does not belong to the department"})
		return
	}

	// Get raw paper info
	var rawPaperInfo model.ResearchPaper
	result := database.Db.First(&rawPaperInfo, paperID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching raw research paper info"})
		return
	}

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

	// Get the submit user
	submitUser, err := GetSubmitUser(paper.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching submit user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Research paper retrieved successfully!",
		"paper":     paper,
		"user":      submitUser,
		"full_text": content,
		"paperInfo": rawPaperInfo,
	})
}

// TODO: using research_paper_id
// Handle approve/reject submission
func HandleApproveRejectSubmission(c *gin.Context) {
	// Get paper id from post form
	paperIDStr := c.PostForm("research_paper_id")
	// Get approval status from post form
	approvalStatus := c.PostForm("approval_status")
	// Get approval user id
	// approvalUserID := c.PostForm("userID")
	// Check if postform empty
	if paperIDStr == "" || approvalStatus == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Some fields are not missing"})
		return
	}

	// Convert paperID to int
	paperID, err := strconv.Atoi(paperIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid research paper ID format"})
		return
	}

	// Get department ID from context
	depID, ok := c.Get("departmentID")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department ID not provided"})
		return
	}

	depIDInt, ok := depID.(int) // Type assertion to int
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid department ID format"})
		return
	}

	// Check if the paper belongs to the department
	if !isPaperBelongsToDepartment(depIDInt, paperID) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized! Research paper does not belong to the department"})
		return
	}

	// Check if approval status is valid
	if approvalStatus != "approve" && approvalStatus != "reject" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval status"})
		return
	}

	// If approval is rejected, need to provide a reason
	if approvalStatus == "reject" {
		rejectedReason := c.PostForm("rejected_reason")
		if rejectedReason == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Reason for rejection is required"})
			return
		}
	}

	// Get the paper info
	var paper model.ResearchPaper
	result := database.Db.First(&paper, paperID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching research paper info"})
		return
	}

	// Check if the paper is awaiting approval
	if strings.ToLower(paper.ResearchPaperStatus) != "awaiting" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research paper is not awaiting approval"})
		return
	}

	// Approve the paper
	if approvalStatus == "approve" {

		// Update the published date
		result := database.Db.Model(&paper).Update("research_paper_status", "in_progress")
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error approving research paper"})
			return
		}

		approvalEmail := c.PostForm("approval_email")
		if approvalEmail == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Approval email is required"})
			return
		}
		// publish to rabbitmq
		data := map[string]interface{}{
			"paper_id":       paper.ResearchPaperID,
			"research_title": paper.ResearchTitle,
			"pdf_path":       paper.PDFPath,
			"advisor":        paper.Advisor,
			"author":         paper.Author,
			"approval_email": approvalEmail,
			"tags":           paper.Tag,
		}
		body, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("failed to marshal data: %v", err)
		}
		err = queue.Producer(body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish to RabbitMQ"})
			return
		}

	}

	// Reject the paper
	if approvalStatus == "reject" {
		result := database.Db.Model(&paper).Updates(model.ResearchPaper{ResearchPaperStatus: "rejected", RejectedReason: c.PostForm("rejected_reason")})
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error rejecting research paper"})
			return
		}
		// Update the published date
		result = database.Db.Model(&paper).Update("submitted_at", time.Now())
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating published date"})
			return
		}

		// Insert reject notification
		err := notification.InsertRejectNotification(paper.ResearchTitle, paper.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting reject notification"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Research paper " + approvalStatus + "d successfully!",
		"updated_paper": paper,
	})
}

// Check if the paper belongs to the department
func isPaperBelongsToDepartment(departmentID, paperID int) bool {
	// Check if the paper belongs to the department
	var paperDepartment model.ResearchPaperDepartment
	result := database.Db.Where("research_paper_id = ? AND department_id = ?", paperID, departmentID).First(&paperDepartment)
	return result.Error == nil
}
