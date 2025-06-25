package repository

import (
	"fmt"
	"net/http"
	"root/controllers/notification"
	"root/database"
	"root/mail"
	"root/model"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

func UpdatePaperStatus(c *gin.Context) {
	var paper model.ResearchPaper
	research_paper_id := c.PostForm("research_paper_id")
	status := c.PostForm("status")

	if research_paper_id == "" || status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research paper ID and status are required"})
		return
	}

	result := database.Db.Where("research_paper_id = ?", research_paper_id).First(&paper)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Research paper not found"})
		return
	}
	updateStatus := database.Db.Model(&paper).Update("research_paper_status", "published")
	if updateStatus.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update research paper status"})
		return
	}
	err := notification.InsertApprovalNotification(paper.ResearchTitle, paper.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error inserting approval notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Paper status updated successfully",
		"paper_id": research_paper_id,
		"status":   status,
	})
}

func NotifyUserForFailPaper(c *gin.Context) {
	approval_email := c.PostForm("approval_email")
	research_paper_id := c.PostForm("research_paper_id")
	if approval_email == "" || research_paper_id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Approval email and research paper ID are required"})
		return
	}
	var paper model.ResearchPaper
	result := database.Db.Where("research_paper_id = ?", research_paper_id).First(&paper)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Research paper not found"})
		return
	}
	failed_paper := viper.GetBool("mailsmtp.toggle.userpanel.fail_pdf_process")
	fmt.Printf("senting failed paper email: %t\n", failed_paper)
	if failed_paper {
		emailBody := mail.EmailTemplateData{
			PreviewHeader: "Paper Processing Failed",
			EmailPurpose:  "Please approve the paper " + paper.ResearchTitle + " again. Because the PDF processing failed.",
			ActionURL:     viper.GetString("client.userpanel") + "/dashboard/repository/hod-submission",
			Action:        "View Paper",
			EmailEnding:   "Please check the status of your paper.",
		}
		emailBodyData, err := mail.CustomizeHTML(emailBody)
		if err != nil {
			fmt.Errorf("Error customizing email: %v", err)
		}

		// Send email to user
		errSending := mail.SendEmail(approval_email, "Scholarize - Paper Processing Failed", emailBodyData)
		if errSending != nil {
			fmt.Errorf("Error sending email to user %s: %v", approval_email, errSending)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Notification email sent successfully"})
}

// func GetPaperStatusForPdfProcessing(c *gin.Context) {
// 	// Create a variable to hold the research paper
// 	var job model.JobQueue
// 	job_id := c.PostForm("job_id")
// 	fmt.Printf("Received Job ID: %s\n", job_id)
// 	if job_id == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Job ID is required"})
// 		return
// 	}
// 	// Find the paper by research_paper_id

// 	result := database.Db.Where("id = ?", job_id).First(&job)
// 	if result.Error != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "Research paper not found"})
// 		return
// 	}
// 	c.JSON(http.StatusOK, gin.H{
// 		"paper_id":    job.PaperID,
// 		"paper_title": job.PaperTitle,
// 		"status":      job.Status,
// 		"error":       job.Error,
// 		"attempts":    job.Attempts,
// 		"message":     "Paper status retrieved successfully",
// 	})

// }
