package repository

import (
	"errors"
	"root/database"
	"root/model"
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
