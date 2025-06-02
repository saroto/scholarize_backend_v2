package chat

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"root/database"
	"root/model"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Database query function - handles data retrieval only
func GetChatSession(sessionID uuid.UUID) (model.ChatSession, error) {
	var result model.ChatSession

	err := database.Db.Table("chat_sessions").Where("session_id = ?",
		sessionID ).First(&result).Error

	if err != nil {
		return model.ChatSession{}, err
	}

	return result, nil
}

// HTTP handler function - handles HTTP request/response
func HandleGetChatSession(c *gin.Context) {
	sessionIDStr := c.Param("sessionID")

	if sessionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}
	// Convert string to UUID
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid session ID format",
		})
		return
	}
	// Call the query function
	result, err := GetChatSession(sessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve chat session",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"message": "Chat session retrieved successfully!",
		"session": result,
	})
}



// Database function - creates chat session if it doesn't exist
func CreateChatSessionIfNotExists(userID int, PaperID int) (model.ChatSession, bool, error) {
	// Check if session already exists
	var existingSession model.ChatSession
	err := database.Db.Table("chat_sessions").
		Select("session_id, user_id, paper_id, created_at, updated_at"). 
		Where("paper_id = ? AND user_id = ?", PaperID, userID).
		First(&existingSession).Error
	
	if err == nil {
		// Session exists, return it with false (not created)
		return existingSession, false, nil
	}

	// Check if it's actually a "not found" error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.ChatSession{}, false, err
	}
	
	newSession := model.ChatSession{
		SessionID: 	   uuid.New(),
		UserID:        userID,
		PaperID:	   PaperID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	fmt.Println("very new",newSession);

	// Create with selected fields only to avoid foreign key issues
	err = database.Db.Table("chat_sessions").
		Select("session_id", "user_id", "paper_id", "created_at", "updated_at").
		Create(&newSession).Error
	
	if err != nil {
		return model.ChatSession{}, false, err
	}

	// Return new session with true (created)
	return newSession, true, nil
}


func UpdateChatSession( c *gin.Context)(){
	if c.PostForm("message") == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing message fields"})
		return
	}

	 message := c.PostForm("message")

	// Convert string to json.RawMessage
	 messageRaw := []byte(message)

	// You need to specify which session to update, e.g., by session_id
	sessionIDStr := c.Param("sessionID")
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}

	result := database.Db.Model(&model.ChatSession{}).
		Where("session_id = ?", sessionID).
		UpdateColumn("message", messageRaw)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update chat session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Chat session updated successfully"})
}


// Secure version - updates only if user owns the session
func UpdateChatSessionMessageByUser(sessionID uuid.UUID, userID int, message string) error {
	fmt.Println("=== DEBUG INFO ===")
	fmt.Println("sessionID:", sessionID)
	fmt.Println("userID:", userID)
	fmt.Println("message:", message)
	
	// First, let's check if the session exists at all
	var existingSession model.ChatSession
	checkResult := database.Db.Where("session_id = ?", sessionID).First(&existingSession)
	
	if checkResult.Error != nil {
		if errors.Is(checkResult.Error, gorm.ErrRecordNotFound) {
			fmt.Println("❌ Session not found with sessionID:", sessionID)
			return fmt.Errorf("session not found with ID: %s", sessionID)
		}
		fmt.Println("❌ Error checking session:", checkResult.Error)
		return checkResult.Error
	}
	// Check if the userID matches
	if existingSession.UserID != userID {
		fmt.Printf("❌ UserID mismatch. Expected: %d, Found: %d\n", userID, existingSession.UserID)
		return fmt.Errorf("session belongs to different user. Expected: %d, Found: %d", userID, existingSession.UserID)
	}
	
	// Now perform the update
	result := database.Db.Model(&model.ChatSession{}).
		Where("session_id = ? AND user_id = ?", sessionID, userID).
		Updates(map[string]interface{}{
			"message":    message,
			"updated_at": time.Now(),
		})

	fmt.Printf("Update result: Error=%v, RowsAffected=%d\n", result.Error, result.RowsAffected)

	if result.Error != nil {
		fmt.Println("❌ Update error:", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		fmt.Println("❌ No rows affected - this shouldn't happen after our checks")
		return gorm.ErrRecordNotFound
	}

	fmt.Println("✅ Chat session updated successfully")
	return nil
}

// Secure HTTP handler - checks user ownership
func UpdateChatSessionSecure(c *gin.Context) {
	// Validate message
	message := c.PostForm("message")
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing message field",
		})
		return
	}

	// Validate JSON format
	if !json.Valid([]byte(message)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON format in message",
		})
		return
	}

	// Get authenticated user
	userIdCont, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	userID, ok := userIdCont.(int)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid user ID",
		})
		return
	}

	// Parse session ID
	sessionIDStr := c.Param("sessionID")
	if sessionIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Session ID is required",
		})
		return
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid session ID format",
		})
		return
	}

	// Call the secure update function
	err = UpdateChatSessionMessageByUser(sessionID, userID, message)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Chat session not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update chat session",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Chat session updated successfully",
	})
}
