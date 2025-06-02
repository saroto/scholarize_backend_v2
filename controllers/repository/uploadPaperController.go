package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"root/config"
	"root/constant"
	"root/controllers/notification"
	"root/database"
	"root/generator"
	"root/model"
	"time"

	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Upload form of research paper
func GetResearchPaperUploadFormData(c *gin.Context) {
	// Get the research data from the database where status is true
	var researchTypes []model.ResearchType
	result := database.Db.Where("research_type_status = ?", true).
		Order("research_type_name ASC").
		Find(&researchTypes)
	if result.Error != nil {
		c.JSON(500, gin.H{
			"message": "Failed to get research types",
		})
		return
	}

	// Get the department data from the database where status is true
	var departments []model.Department
	result = database.Db.Where("department_status = ?", true).
		Order("department_name ASC").
		Find(&departments)
	if result.Error != nil {
		c.JSON(500, gin.H{
			"message": "Failed to get departments",
		})
		return
	}
	// Return the research paper upload form
	c.JSON(200, gin.H{
		"research_types": researchTypes,
		"departments":    departments,
	})
}

// TODO: using public_id
// Handle upload of research paper
func HandleResearchPaperUpload(c *gin.Context) {
	// Check if postform is missing
	if c.PostForm("research_title") == "" || c.PostForm("research_type_id") == "" || c.PostForm("abstract") == "" || c.PostForm("tag") == "" || c.PostForm("author") == "" || c.PostForm("department_id") == "" || c.PostForm("full_text") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required fields"})
		return
	}

	fmt.Println("Received request to upload research paper")
	// Get user_id from context
	userIdCont, _ := c.Get("userID")

	// Convert userId to int
	userId, ok := userIdCont.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID"})
		return
	}

	// Get the research title
	title := c.PostForm("research_title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research title is required"})
		return
	}

	// Get the research type ID from the POST request
	researchTypeIdStr := c.PostForm("research_type_id")
	if researchTypeIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research type ID is required"})
		return
	}
	researchTypeId, _ := strconv.Atoi(researchTypeIdStr)

	// Get the research abstract
	abstract := c.PostForm("abstract")
	if abstract == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Abstract is required"})
		return
	}

	// Get research Tags
	tag := c.PostForm("tag")
	if tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tag is required"})
		return
	}

	// Get the author of the research
	author := c.PostForm("author")
	if author == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Author is required"})
		return
	}

	// Get the advisor of the research
	advisor := c.PostForm("advisor")
	if advisor == "" {
		advisor = "None"
	}

	// Get the department ID from the POST request
	departmentIdStr := c.PostForm("department_id")
	if departmentIdStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Department ID is required"})
		return
	}

	// Split the department ID string by , into an array of department IDs
	departmentIds := strings.Split(departmentIdStr, ",")

	// Get the paper file
	file, err := c.FormFile("paper_file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File not found"})
		return
	}

	// Check if file is not a PDF
	if file.Header.Get("Content-Type") != "application/pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File must be a PDF"})
		return
	}

	// Check if title is a duplicate
	var researchPaperCheck model.ResearchPaper
	res := database.Db.Where("research_title = ?", title).First(&researchPaperCheck)
	if res.Error == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Research title already exists"})
		return
	}

	// Upload full text
	fullText := c.PostForm("full_text")
	if fullText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Full text is required"})
		return
	}

	laravelURL := config.GetFileServiceURL("storefile_cleantext")

	uploadResp, err := UploadFileAndTextToLaravel(file, fullText, laravelURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error uploading file"})
		return
	}

	// Upload the clean text to the database
	cleanTextId, err := uploadCleanTextToDatabase(uploadResp.OptimizedText)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload clean text to database"})
		return
	}

	fullTextId, err := uploadFullTextToDatabase(fullText)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload full text to database"})
		return
	}

	uniqueID, err := generator.GenerateResearchPaperUUID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate unique ID"})
		return
	}

	// Create a new research paper record
	researchPaper := model.ResearchPaper{
		PublicID:            uniqueID,
		ResearchTitle:       title,
		ResearchTypeID:      researchTypeId,
		Abstract:            abstract,
		Tag:                 tag,
		Author:              author,
		Advisor:             advisor,
		PDFPath:             uploadResp.FileName,
		ResearchPaperStatus: "awaiting",
		RejectedReason:      "none",
		FulltextID:          &fullTextId,
		CleantextID:         &cleanTextId,
		UserID:              userId,
		SubmittedAt:         time.Now(),
		PublishedAt:         time.Now(),
	}

	// Insert research paper record into the database
	result := database.Db.Create(&researchPaper)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload research paper to database"})
		return
	}

	// Define a struct to hold the department info
	type DepartmentInfo struct {
		ID   int    `gorm:"column:department_id" json:"department_id"`
		Name string `gorm:"column:department_name" json:"department_name"`
		Tag  string `gorm:"column:department_tag" json:"department_tag"`
	}

	// Create a slice to hold the department info
	var departmentInfos []DepartmentInfo

	// Insert ResearchPaperDepartment
	for _, departmentIdStr := range departmentIds {
		departmentId, err := strconv.Atoi(departmentIdStr)
		if err != nil {
			// Handle conversion error
			continue
		}

		// Create a new DepartmentResearchPaper instance
		departmentPaper := model.ResearchPaperDepartment{
			DepartmentID:    departmentId,
			ResearchPaperID: researchPaper.ResearchPaperID,
		}

		// Insert the DepartmentResearchPaper instance into the database
		result := database.Db.Create(&departmentPaper)
		if result.Error != nil {
			continue
		}

		// Fetch the department info from the database
		var departmentInfo DepartmentInfo
		result = database.Db.Model(&model.Department{}).Where("department_id = ?", departmentId).First(&departmentInfo)
		if result.Error != nil {
			continue
		}

		// Add the department info to the slice
		departmentInfos = append(departmentInfos, departmentInfo)

		// Get the department heads by department ID
		departmentHeadIDs := GetDepartmentHeadsByDepartmentID(departmentId)

		// Insert notifications for department heads
		for _, departmentHeadID := range departmentHeadIDs {
			insertErr := notification.InsertNewPaperNotification(title, departmentHeadID)
			if insertErr != nil {
				fmt.Println("Failed to insert notification for department head")
			}
		}
	}

	// Return Response
	c.JSON(http.StatusOK, gin.H{
		"message":     "Research paper uploaded successfully",
		"paper":       researchPaper,
		"departments": departmentInfos,
	})
}

// Upload full text
func uploadFullTextToDatabase(fullText string) (int, error) {
	// Create a new full text record
	fullTextRecord := model.Fulltext{
		FulltextContent: fullText,
	}

	// Insert the full text record into the database
	result := database.Db.Create(&fullTextRecord)
	if result.Error != nil {
		return 0, result.Error
	}

	return fullTextRecord.FulltextID, nil
}

// Upload clean text
func uploadCleanTextToDatabase(cleanText string) (int, error) {
	// Create a new clean text record
	cleanTextRecord := model.Cleantext{
		CleantextContent: cleanText,
	}

	// Insert the clean text record into the database
	result := database.Db.Create(&cleanTextRecord)
	if result.Error != nil {
		return 0, result.Error
	}

	return cleanTextRecord.CleantextID, nil
}

func UploadFileAndTextToLaravel(fileHeader *multipart.FileHeader, cleanText, laravelURL string) (*constant.LaravelUploadResponse, error) {
	// Open the file from FileHeader
	file, err := fileHeader.Open()
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()
	fmt.Println("File opened successfully")

	// Prepare the multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the file part to the multipart
	filePart, err := writer.CreateFormFile("pdf_file", fileHeader.Filename)
	if err != nil {
		fmt.Println("Error creating form file:", err)
		return nil, err
	}
	_, err = io.Copy(filePart, file)
	if err != nil {
		fmt.Println("Error copying file data:", err)
		return nil, err
	}

	// Add the clean text part
	textPart, err := writer.CreateFormField("full_text")
	if err != nil {
		fmt.Println("Error creating form field:", err)
		return nil, err
	}
	_, err = textPart.Write([]byte(cleanText))
	if err != nil {
		fmt.Println("Error writing text data:", err)
		return nil, err
	}

	writer.Close()

	// Create the request
	req, err := http.NewRequest("POST", laravelURL, body)
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	respBody, _ := io.ReadAll(resp.Body)

	// Try to decode the response body as JSON
	var uploadResp constant.LaravelUploadResponse
	if err := json.Unmarshal(respBody, &uploadResp); err != nil {
		fmt.Println("Error decoding response:", err)
		return nil, fmt.Errorf("error decoding response: %v, response body: %s", err, respBody)
	}
	fmt.Printf("Response: %+v\n", uploadResp)

	return &uploadResp, nil
}

// Clean text only
func CleanTextAtLaravel(cleanText, laravelURL string) (string, error) {
	// Prepare the request body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add the clean text part
	textPart, err := writer.CreateFormField("full_text")
	if err != nil {
		fmt.Println("Error creating form field:", err)
		return "", err
	}
	_, err = textPart.Write([]byte(cleanText))
	if err != nil {
		fmt.Println("Error writing text data:", err)
		return "", err
	}

	writer.Close()

	// Create the request
	req, err := http.NewRequest("POST", laravelURL, body)
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return "", err
	}
	defer resp.Body.Close()

	// Struct response
	type CleanTextResponse struct {
		OptimizedText string `json:"optimized_text"`
	}

	// Read the response
	var cleanTextResp CleanTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&cleanTextResp); err != nil {
		fmt.Println("Error decoding response:", err)
		return "", err
	}

	return cleanTextResp.OptimizedText, nil
}

// Get Department Heads by department ID
func GetDepartmentHeadsByDepartmentID(departmentID int) []int {
	// Create a slice to hold the department heads
	var departmentHeads []model.DepartmentHead

	// Create a slice to hold the department head IDs
	var departmentHeadIDs []int

	// Fetch the department heads from the database
	result := database.Db.Where("department_id = ?", departmentID).Find(&departmentHeads)
	if result.Error != nil {
		return departmentHeadIDs
	}

	// Add the department head IDs to the slice
	for _, departmentHead := range departmentHeads {
		departmentHeadIDs = append(departmentHeadIDs, departmentHead.UserID)
	}

	return departmentHeadIDs
}
