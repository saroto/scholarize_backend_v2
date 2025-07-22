package repository

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"root/database"
	"root/mail"
	"root/model"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// Get paperID from public_id
func getRawPaperInfoFromPublicID(publicID string) (model.ResearchPaper, error) {
	// Create a variable to hold the research paper
	var paper model.ResearchPaper

	// Find the paper by public_id
	result := database.Db.Where("public_id = ?", publicID).First(&paper)
	if result.Error != nil {
		return model.ResearchPaper{}, result.Error
	}

	// Check if the paper was found
	if result.RowsAffected == 0 {
		return model.ResearchPaper{}, errors.New("research paper not found")
	}

	return paper, nil
}

// Get paperID from public_id
func getPaperIDFromPublicID(publicID string) (int, error) {
	// Create a variable to hold the research paper
	var paper model.ResearchPaper

	// Find the paper by public_id
	result := database.Db.Where("public_id = ?", publicID).First(&paper)
	if result.Error != nil {
		return 0, result.Error
	}

	// Check if the paper was found
	if result.RowsAffected == 0 {
		return 0, errors.New("research paper not found")
	}

	return paper.ResearchPaperID, nil
}

func isPythonServiceAvailable(endpoint string) bool {
	client := http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Get(endpoint + "/api/health")
	if err != nil {
		log.Printf("Python service not available: %v", err)
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func handlePDFProcessingFailure(paper *model.ResearchPaper, approvalUserID string, errChan chan error) {
	if viper.GetBool("mailsmtp.toggle.userpanel.fail_pdf_process") {
		emailBody := mail.EmailTemplateData{
			PreviewHeader: "PDF Processing Failed - " + paper.ResearchTitle,
			EmailPurpose:  "Your PDF processing has failed. Please check the system for more details.",
			ActionURL:     fmt.Sprintf("%s/papers/%d", viper.GetString("client.userpanel"), paper.ResearchPaperID),
			Action:        "View Paper",
			EmailEnding:   "If you believe this is a mistake, please ignore this message.",
		}

		htmlBody, err := mail.CustomizeHTML(emailBody)
		if err != nil {
			errChan <- fmt.Errorf("error customizing email template: %w", err)
			return
		}

		userID, err := strconv.Atoi(approvalUserID)
		if err != nil {
			errChan <- fmt.Errorf("invalid user ID: %w", err)
			return
		}

		var user model.ScholarizeUser
		if err := database.Db.First(&user, userID).Error; err != nil {

			errChan <- fmt.Errorf("error fetching user email: %w", err)
			return
		}

		if err := mail.SendEmail(user.UserEmail, "Scholarize - PDF Processing Failed. Please check your submission!", htmlBody); err != nil {
			errChan <- fmt.Errorf("error sending failure email: %w", err)
			return
		}
	}

	// if err := database.Db.Model(paper).Update("pdf_processing_status", string(model.PdfProcessingStatusFailed)).Error; err != nil {
	// 	log.Printf("Failed to mark PDF processing as failed: %v", err)
	// }
	log.Printf("PDF processing failed for paper %d", paper.ResearchPaperID)
}
