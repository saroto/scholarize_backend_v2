package repository

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"root/constant"
	"root/controllers/notification"
	"root/database"
	"root/model"
	"root/permission"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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
	// if approvalStatus == "approve" {
	// 	result := database.Db.Model(&paper).Update("research_paper_status", "published")
	// 	if result.Error != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error approving research paper"})
	// 		return
	// 	}

	// 	// Update the published date
	// 	result = database.Db.Model(&paper).Update("published_at", time.Now())
	// 	if result.Error != nil {
	// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating published date"})
	// 		return
	// 	}
	// 	result = database.Db.Model(&paper).Update("pdf_processing_status", string(model.PdfProcessingStatusProcessing))

	// 	// Send request to python (AI service) after the paper is approved
	// 	go handleSentRequestToPythonService(paper.ResearchPaperID)

	// }

	// TO DO: using research_paper_id and go routine
	if approvalStatus == "approve" {
		// First update to published status
		result := database.Db.Model(&paper).Update("research_paper_status", "published")
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error approving research paper"})
			return
		}

		// Update the published date
		result = database.Db.Model(&paper).Update("published_at", time.Now())
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating published date"})
			return
		}

		// Set PDF processing status to processing
		result = database.Db.Model(&paper).Update("pdf_processing_status", string(model.PdfProcessingStatusProcessing))
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating PDF processing status"})
			return
		}

		// Process PDF asynchronously with proper error handling
		go func(paperID int) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PDF processing panicked for paper %d: %v", paperID, r)
					// Update status to failed if panic occurs
					database.Db.Model(&model.ResearchPaper{}).Where("research_paper_id = ?", paperID).
						Update("pdf_processing_status", string(model.PdfProcessingStatusFailed))
				}
			}()

			if err := handleSentRequestToPythonService(paperID); err != nil {
				log.Printf("PDF processing failed for paper %d: %v", paperID, err)
				// The handleSentRequestToPythonService function already handles status updates
				// but you could add additional logic here if needed
			} else {
				log.Printf("PDF processing completed successfully for paper %d", paperID)
			}
		}(paper.ResearchPaperID)
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

// sent request to python (AI service) after the paper is approved
// argument: paper info
func handleSentRequestToPythonService(paperID int) error {
	// Get paper info
	var paperInfo model.ResearchPaper

	var response constant.PythonResponse
	python_endpoint := viper.GetString("semantic_search.host")

	result := database.Db.First(&paperInfo, paperID)
	if result.Error != nil {
		log.Printf("Error fetching research paper info: %v", result.Error)
		return result.Error
	}

	// Construct the URL for the Python service - FIXED URL formatting
	endpoint := fmt.Sprintf("%s/api/pdf", python_endpoint)
	// get paper type

	// Create form data
	formData := url.Values{}
	formData.Set("research_paper_id", fmt.Sprintf("%d", paperID))
	formData.Set("research_title", paperInfo.ResearchTitle)
	formData.Set("author", paperInfo.Author)
	formData.Set("advisors", paperInfo.Advisor) // Note: Changed from advisor to advisors to match FastAPI
	formData.Set("pdf_path", paperInfo.PDFPath)
	formData.Set("tag", paperInfo.Tag)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with form data
	req, err := http.NewRequest("POST", endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return fmt.Errorf("error creating request: %v", err)
	}

	// Set headers for form data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request to AI service: %v", err)
		return fmt.Errorf("error sending request to AI service: %v", err)
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
		// Update status to failed
		database.Db.Model(&paperInfo).Update("pdf_processing_status", string(model.PdfProcessingStatusFailed))
		return fmt.Errorf("error reading response body: %v", err)
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		log.Printf("Error parsing JSON response: %v", err)
		log.Printf("Raw response: %s", string(respBody))
		return fmt.Errorf("error parsing JSON response: %v", err)
	}

	if response.PDFProcessStatus == "failed" {
		log.Println("reach here")
		result = database.Db.Model(&paperInfo).Update("research_paper_status", "awaiting")
		if result.Error != nil {
			log.Printf("Error updating research paper status to awaiting: %v", result.Error)
			return fmt.Errorf("error updating research paper status to awaiting: %v", result.Error)
		}
		result = database.Db.Model(&paperInfo).Update("pdf_processing_status", string(model.PdfProcessingStatusFailed))
		if result.Error != nil {
			log.Printf("Error updating PDF processing status to failed: %v", result.Error)
			return fmt.Errorf("error updating PDF processing status to failed: %v", result.Error)
		}
		// failPdfProcessSMTP := viper.GetBool("mailsmtp.toggle.userpanel.fail_pdf_process")
		// if failPdfProcessSMTP {
		// 	emailBody := mail.EmailTemplateData{
		// 		PreviewHeader: "PDF Processing Failed - " + paperInfo.ResearchTitle,
		// 		EmailPurpose:  "Your PDF processing has failed. Please check the system for more details.",
		// 		ActionURL:     viper.GetString("client.userpanel") + "/papers/" + fmt.Sprintf("%d", paperInfo.ResearchPaperID),
		// 		Action:        "View Paper",
		// 		EmailEnding:   "If you believe this is a mistake, please ignore this message.",
		// 	}
		// 	emailBodyData, err := mail.CustomizeHTML(emailBody)
		// 	if err != nil {
		// 		log.Printf("Error customizing email template: %v", err)
		// 		return fmt.Errorf("error customizing email template: %v", err)
		// 	}

		// 	// Send email to user
		// 	errSending := mail.SendEmail(user.UserEmail, "Scholarize - You have been invited to join collaboration", emailBodyData)
		// 	if errSending != nil {
		// 		log.Printf("Error sending email: %v", errSending)
		// 		return fmt.Errorf("error sending email: %v", errSending)
		// 	}
		// }
	} else {
		result = database.Db.Model(&paperInfo).Update("pdf_processing_status", string(model.PdfProcessingStatusSuccess))
		if result.Error != nil {
			log.Printf("Error updating PDF processing status to success: %v", result.Error)
			return fmt.Errorf("error updating PDF processing status to success: %v", result.Error)
		}
		// Insert approval notification
		err := notification.InsertApprovalNotification(paperInfo.ResearchTitle, paperInfo.UserID)
		if err != nil {
			log.Printf("Error inserting approval notification: %v", err)
			return fmt.Errorf("error inserting approval notification: %v", err)
		}
	}
	// Log successful embedding creation
	log.Printf("Successfully sent paper ID %d to AI service for embedding creation", paperID)

	return nil
}
