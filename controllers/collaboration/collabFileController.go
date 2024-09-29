package collaboration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"root/config"
	"root/constant"
	"root/database"
	"root/model"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FileUploadResponse struct {
	FileName string `json:"file_name"`
	FileSize string `json:"file_size"`
	FilePath string `json:"file_path"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

func validateFolderExist(folderId int) bool {
	var folder model.Folder
	if err := database.Db.Where("folder_id = ?", folderId).First(&folder).Error; err != nil {
		return false
	}
	return true
}

func uploadCollabFilesToLaravel(fileHeaders []*multipart.FileHeader, collabLaravelURL string) ([]FileUploadResponse, error) {
	// Prepare the multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	responses := []FileUploadResponse{}

	for _, fileHeader := range fileHeaders {
		// Open the file from FileHeader
		file, err := fileHeader.Open()
		if err != nil {
			fmt.Println("Error opening file:", err)
			return nil, err
		}
		defer file.Close()
		fmt.Println("File opened successfully:", fileHeader.Filename)

		// Add the file part to the multipart
		filePart, err := writer.CreateFormFile("files[]", fileHeader.Filename)
		if err != nil {
			fmt.Println("Error creating form file:", err)
			return nil, err
		}

		_, err = io.Copy(filePart, file)
		if err != nil {
			fmt.Println("Error copying file data:", err)
			return nil, err
		}
	}

	// Close the multipart writer to finalize the boundary
	writer.Close()

	// Create the request
	req, err := http.NewRequest("POST", collabLaravelURL, body)
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

	// Read the response into a slice of FileUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&responses); err != nil {
		fmt.Println("Error decoding response:", err)
		return nil, err
	}

	return responses, nil
}

func deleteCollabFileFromLaravel(filepath string, collabLaravelURL string) (MessageResponse, error) {
	// Encode form data
	formData := url.Values{}
	formData.Set("file", filepath)
	reqBody := bytes.NewBufferString(formData.Encode())

	// Create the request with the correct body
	req, err := http.NewRequest("POST", collabLaravelURL, reqBody)
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return MessageResponse{}, err
	}

	// Set the content type to application/x-www-form-urlencoded
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return MessageResponse{}, err
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return MessageResponse{}, fmt.Errorf("server responded with a non-OK status: %d", resp.StatusCode)
	}

	// Decode the JSON response into MessageResponse
	var response MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Println("Error decoding response:", err)
		return MessageResponse{}, err
	}

	return response, nil
}

// HandleGetAllFolders
func HandleGetAllFolders(c *gin.Context) {
	// Get all folders data from the database
	var folders []model.Folder

	if err := database.Db.Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching folders data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Folders fetched successfully",
		"folders": folders,
	})
}

// HandleUploadFilesToLaravel uploads files
func HandleUploadFilesToCollab(c *gin.Context) {
	// Get the files from the request
	if err := c.Request.ParseMultipartForm(200 << 20); err != nil { // 200 MB limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form: " + err.Error()})
		return
	}

	// Get the file headers
	fileHeaders, exists := c.Request.MultipartForm.File["files"]
	if !exists || len(fileHeaders) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No files found in the request"})
		return
	}

	// Get folder id from the request
	folderID := c.PostForm("folder_id")
	if folderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No folder id found in the request"})
		return
	}

	// Convert the folder id to int
	folderIDInt, ok := strconv.Atoi(folderID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid folder id"})
		return
	}

	// Validate folder
	if !validateFolderExist(folderIDInt) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Folder does not exist"})
		return
	}

	// Get the collab id from param
	collabID := c.Param("collab_id")

	// Covert the collab id to int
	collabIDInt, ok := strconv.Atoi(collabID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid collab id"})
		return
	}

	// Get the user id from the context
	userID, _ := c.MustGet("userID").(int)

	// Upload the files to the Laravel backend
	collabLaravelURL := config.GetFileServiceURL("collab_upload")

	response, errr := uploadCollabFilesToLaravel(fileHeaders, collabLaravelURL)
	if errr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error uploading files"})
		return
	}

	fmt.Println("Response from Laravel:", response)

	if len(response) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error uploading files"})
		return
	}

	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error starting transaction"})
		return

	}

	// Insert the file details into the database
	for _, file := range response {
		// Check if the file already exists skip
		var existingFile model.File
		err := tx.Joins("JOIN filefolder ON filefolder.file_id = file.file_id").
			Where("file.file_name = ? AND filefolder.folder_id = ? AND file.collab_id = ?", file.FileName, folderID, collabIDInt).
			First(&existingFile).Error

		if err == nil {
			// File exists, proceed with deletion
			deleteURL := config.GetFileServiceURL("collab_delete")
			response, err := deleteCollabFileFromLaravel(existingFile.FilePath, deleteURL)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting file"})
				return
			}
			fmt.Println("Deleting existing file:", response.Message)

			// Update the existing file path and time, update user id if different
			if err := tx.Model(&existingFile).Update("file_path", file.FilePath).Update("updated_at", time.Now()).Update("user_id", userID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating file details"})
				return
			}

			fmt.Printf("File %s already exists, updating file details\n", file.FileName)
			continue
		}

		eachFile := model.File{
			FileName:  file.FileName,
			FileSize:  file.FileSize,
			FilePath:  file.FilePath,
			CollabID:  collabIDInt,
			UpdatedAt: time.Now(),
			UserID:    userID,
		}

		if err := tx.Create(&eachFile).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving file details"})
			return
		}

		// Insert the file into the folder
		folderFile := model.FileFolder{
			FolderID: folderIDInt,
			FileID:   eachFile.FileID,
		}

		if err := tx.Create(&folderFile).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "Error saving file details"})
			return
		}
	}

	// Commit the transaction
	tx.Commit()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Files uploaded successfully",
		"files":   response,
	})
}

// Handle Delete Collab File
func HandleDeleteCollabFile(c *gin.Context) {
	// Get the file id from post
	fileID := c.PostForm("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file id found in the request"})
		return
	}

	// Convert the file id to int
	fileIDInt, ok := strconv.Atoi(fileID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid file id"})
		return
	}

	// Get the collab id from param
	collabID := c.Param("collab_id")
	if collabID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No collab id found in the request"})
		return
	}

	// Covert the collab id to int
	collabIDInt, ok := strconv.Atoi(collabID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid collab id"})
		return
	}

	// Get the file details from the database
	file := model.File{}
	if err := database.Db.Where("file_id = ?", fileIDInt).First(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching file details"})
		return
	}

	// Check if the file belongs to the collab
	if file.CollabID != collabIDInt {
		c.JSON(http.StatusBadRequest, gin.H{"message": "File does not belong to the collab"})
		return
	}

	// Delete the file from the Laravel backend
	deleteURL := config.GetFileServiceURL("collab_delete")
	response, errr := deleteCollabFileFromLaravel(file.FilePath, deleteURL)
	if errr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting file"})
		return
	}

	fmt.Println("Response from Laravel:", response)

	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error starting transaction"})
		return
	}

	// Delete the file from the database
	if err := tx.Where("file_id = ?", fileIDInt).Delete(&model.File{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting file details"})
		return
	}

	// Delete the file from the file folder
	if err := tx.Where("file_id = ?", fileIDInt).Delete(&model.FileFolder{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error deleting file details"})
		return
	}

	// Commit the transaction
	tx.Commit()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File " + file.FileName + " deleted successfully",
	})
}

// Handle Rename File
func HandleHandleRenameFile(c *gin.Context) {
	// Get the file id from post
	fileID := c.PostForm("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file id found in the request"})
		return
	}

	// Convert the file id to int
	fileIDInt, ok := strconv.Atoi(fileID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid file id"})
		return
	}

	// Get the new file name from post
	newFileName := c.PostForm("file_name")
	if newFileName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No new file name found in the request"})
		return
	}

	// Get the collab id from param
	collabID := c.Param("collab_id")
	if collabID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No collab id found in the request"})
		return
	}

	// Covert the collab id to int
	collabIDInt, ok := strconv.Atoi(collabID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid collab id"})
		return
	}

	// Get the folder id from the request
	folderID := c.PostForm("folder_id")
	if folderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No folder id found in the request"})
		return
	}

	// Convert the folder id to int
	folderIDInt, ok := strconv.Atoi(folderID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid folder id"})
		return
	}

	// Validate folder
	if !validateFolderExist(folderIDInt) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Folder does not exist"})
		return
	}

	// Start the transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error starting transaction"})
		return
	}

	// Get the file details from the database
	file := model.File{}
	if err := tx.Where("file_id = ?", fileIDInt).First(&file).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching file details"})
		return
	}

	// Check if the file belongs to the collab
	if file.CollabID != collabIDInt {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"message": "File does not belong to the collab"})
		return
	}

	oldFileName := file.FileName

	// Check if the new file name is same as the old file name
	if oldFileName == newFileName {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"message": "New file name is same as old file name"})
		return
	}

	// Check if file name is already exist in the current folder of the current collab
	var existingFile model.File
	err := tx.Joins("JOIN filefolder ON filefolder.file_id = file.file_id").
		Where("file.file_name = ? AND filefolder.folder_id = ? AND file.collab_id = ?", newFileName, folderIDInt, collabIDInt).
		First(&existingFile).Error
	if err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"message": "File name already exists in the folder"})
		return
	}

	// Update the file name
	newFileName = strings.ReplaceAll(newFileName, " ", "-")

	// Update the file name in the database and updated at
	if err := tx.Model(&file).Update("file_name", newFileName).Update("updated_at", time.Now()).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating file name"})
		return
	}

	// Commit the transaction
	tx.Commit()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File " + oldFileName + " renamed to " + newFileName + " successfully",
	})
}

// Handle Move File
func HandleMoveFile(c *gin.Context) {
	// Get the file id from post
	fileID := c.PostForm("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file id found in the request"})
		return
	}

	// Convert the file id to int
	fileIDInt, ok := strconv.Atoi(fileID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid file id"})
		return
	}

	// Get the collab id from param
	collabID := c.Param("collab_id")
	if collabID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No collab id found in the request"})
		return
	}

	// Covert the collab id to int
	collabIDInt, ok := strconv.Atoi(collabID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid collab id"})
		return
	}

	// Get target folder id from the request
	folderID := c.PostForm("folder_id")
	if folderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No folder id found in the request"})
		return
	}

	// Convert the folder id to int
	folderIDInt, ok := strconv.Atoi(folderID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid folder id"})
		return
	}

	// Validate folder
	if !validateFolderExist(folderIDInt) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Folder does not exist"})
		return
	}

	// Start the transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error starting transaction"})
		return
	}

	// Get the file details from the database
	file := model.File{}
	if err := tx.Where("file_id = ?", fileIDInt).First(&file).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching file details"})
		return
	}

	// Check if the file belongs to the collab
	if file.CollabID != collabIDInt {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"message": "File does not belong to the collab"})
		return
	}

	// Check if the file is already in the target folder
	var existingFileFolder model.FileFolder
	err := tx.Where("file_id = ? AND folder_id = ?", fileIDInt, folderIDInt).First(&existingFileFolder).Error
	if err == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"message": "File already exists in the target folder"})
		return
	}

	// Check if file name is already exist in the target folder of the current collab
	var existingFile model.File
	errr := tx.Joins("JOIN filefolder ON filefolder.file_id = file.file_id").
		Where("file.file_name = ? AND filefolder.folder_id = ? AND file.collab_id = ?", file.FileName, folderIDInt, collabIDInt).
		First(&existingFile).Error
	if errr == nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"message": "File name already exists in the target folder"})
		return
	}

	// Update the folder id
	if err := tx.Model(&file).Update("updated_at", time.Now()).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating file details"})
		return
	}

	// Update the file folder
	if err := tx.Model(&model.FileFolder{}).Where("file_id = ?", fileIDInt).Update("folder_id", folderIDInt).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error updating file details"})
		return
	}

	// Commit the transaction
	tx.Commit()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error committing transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "File " + file.FileName + " moved to folder " + folderID + " successfully",
	})
}

// Handle Download File
func HandleDownloadFile(c *gin.Context) {
	// Get the file path from the query parameter
	filePath := c.Query("file_path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file found in the request"})
		return
	}

	// Get the file from the Laravel backend
	downloadURL := config.GetFileServiceURL("collab_download") + filePath

	// Make an HTTP GET request to the Laravel server to download the file
	resp, err := http.Get(downloadURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error downloading file from backend"})
		return
	}
	defer resp.Body.Close()

	// Check response status code from Laravel
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Backend server error"})
		return
	}

	// Set the Content-Type header to the appropriate type (could also set it based on file extension)
	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

	// Optional: Set the Content-Disposition header to prompt a download with a filename
	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+filePath)

	// Stream the file content directly to the client
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error streaming file to client"})
		return
	}
}

// Handle Get File Details By Folder
func HandleGetFileDetailsByFolderOfCollab(c *gin.Context) {
	// Pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "15"))

	// Get the folder id from the query parameter
	folderID := c.Query("folder_id")
	if folderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No folder id found in the request"})
		return
	}

	// Convert the folder id to int
	folderIDInt, ok := strconv.Atoi(folderID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid folder id"})
		return
	}

	// Validate folder
	if !validateFolderExist(folderIDInt) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Folder does not exist"})
		return
	}

	// Get the collab id from param
	collabID := c.Param("collab_id")
	if collabID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No collab id found in the request"})
		return
	}

	// Covert the collab id to int
	collabIDInt, ok := strconv.Atoi(collabID)
	if ok != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid collab id"})
		return
	}

	// Get the file details from the database
	folder, err := getFolderAndFiles(folderIDInt, collabIDInt, page, count)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching file details"})
		return
	}

	// Get collab details
	var collab model.Collab
	if err := database.Db.Where("collab_id = ?", collabIDInt).First(&collab).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error fetching collab details"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":               "File details fetched successfully",
		"folder":                folder,
		"collab_id":             collab.CollabID,
		"collab_name":           collab.CollabName,
		"collab_archive_status": collab.CollabArchiveStatus,
	})
}

func getFolderAndFiles(folderID int, collabID int, page int, count int) (constant.FolderDetails, error) {
	var folderOr model.Folder
	offset := (page - 1) * count

	// Fetch the folder
	if err := database.Db.Table("folder").Where("folder_id = ?", folderID).First(&folderOr).Error; err != nil {
		return constant.FolderDetails{}, err
	}

	// Count Files in Folder
	var totalFiles int64
	if err := database.Db.Table("file").
		Joins("JOIN filefolder ON filefolder.file_id = file.file_id").
		Where("filefolder.folder_id = ? AND file.collab_id = ?", folderID, collabID).
		Count(&totalFiles).Error; err != nil {
		return constant.FolderDetails{}, err
	}

	// Fetch Files in Folder
	var files []model.File
	if err := database.Db.Joins("JOIN filefolder ON filefolder.file_id = file.file_id").
		Where("filefolder.folder_id = ? AND file.collab_id = ?", folderID, collabID).
		Offset(offset).Limit(count).Find(&files).Error; err != nil {
		return constant.FolderDetails{}, err
	}

	// Prepare the response
	folder := constant.FolderDetails{
		FolderID:   folderOr.FolderID,
		FolderName: folderOr.FolderName,
		TotalFiles: int(totalFiles),
	}

	folder.Files = make([]constant.FileDetails, len(files))
	for i, file := range files {
		var user model.ScholarizeUser
		if err := database.Db.Where("user_id = ?", file.UserID).First(&user).Error; err != nil {
			log.Printf("Error fetching user details: %v", err)
			return constant.FolderDetails{}, err
		}

		folder.Files[i] = constant.FileDetails{
			FileID:    file.FileID,
			FileName:  file.FileName,
			FileSize:  file.FileSize,
			UpdatedAt: file.UpdatedAt.Format("2006-01-02 15:04:05"),
			FilePath:  file.FilePath,
			UserDetails: constant.SimpleUser{
				UserID:         user.UserID,
				UserName:       user.UserName,
				UserEmail:      user.UserEmail,
				UserProfileImg: user.UserProfileImg,
			},
		}
	}

	return folder, nil
}

// Delete all files of a collab
func deleteAllCollabFiles(tx *gorm.DB, collabID int) error {
	// Get all files of the collab
	var files []model.File
	if err := tx.Where("collab_id = ?", collabID).Find(&files).Error; err != nil {
		return err
	}

	var fileFolder model.FileFolder
	for _, file := range files {
		// Delete the file from the file folder
		if err := tx.Where("file_id = ?", file.FileID).Delete(&fileFolder).Error; err != nil {
			return err
		}

		// Delete the file from the database
		if err := tx.Where("file_id = ?", file.FileID).Delete(&file).Error; err != nil {
			return err
		}

		// Delete the file from the Laravel backend
		deleteURL := config.GetFileServiceURL("collab_delete")
		_, err := deleteCollabFileFromLaravel(file.FilePath, deleteURL)
		if err != nil {
			return err
		}
	}

	return nil
}
