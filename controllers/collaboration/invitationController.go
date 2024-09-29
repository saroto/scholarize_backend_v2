package collaboration

import (
	"fmt"
	"net/http"
	"root/controllers/notification"
	"root/database"
	"root/mail"
	"root/model"
	"root/permission"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

func GenerateInviteToken(userID, collabID int) (string, error) {
	// Generate a new token
	token := jwt.New(jwt.SigningMethodHS256)
	InviteTokenSecret := viper.GetString("secret_invite_token.key")

	// Set claims
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = userID
	claims["collab_id"] = collabID
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()

	// Sign the token
	tokenString, err := token.SignedString([]byte(InviteTokenSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseInviteToken(tokenString string) (*jwt.Token, error) {
	// Parse the token
	InviteTokenSecret := viper.GetString("secret_invite_token.key")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(InviteTokenSecret), nil
	})
	if err != nil {
		return nil, err
	}

	return token, nil
}

func ValidateInviteToken(tokenString string) (jwt.MapClaims, error) {
	// Parse the token
	token, err := ParseInviteToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Validate the token
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, err
	}

	return claims, nil
}

func ExtractInviteTokenData(tokenString string) (int, int, error) {
	// Validate the token
	claims, err := ValidateInviteToken(tokenString)
	if err != nil {
		return 0, 0, err
	}

	// Extract the data
	userID := int(claims["user_id"].(float64))
	collabID := int(claims["collab_id"].(float64))

	return userID, collabID, nil
}

// SMTP
// Added Notification
// For creating group only
func CreateInviteTokenForMembers(collabID int, memberIDs []string) {
	// Generate the invite token and add to database
	for _, memberID := range memberIDs {
		memberID, _ := strconv.Atoi(memberID)
		token, err := GenerateInviteToken(memberID, collabID)
		if err != nil {
			return
		}

		// Create the invite data
		inviteData := model.Invite{
			InviteToken: token,
			UserID:      memberID,
		}
		database.Db.Create(&inviteData)

		// Create Invite Collab Data
		inviteCollabData := model.InviteCollab{
			InviteID: inviteData.InviteID,
			CollabID: collabID,
		}
		database.Db.Create(&inviteCollabData)

		// Add user to the collab
		addCollabMember(collabID, memberID)

		// Get collab detail
		var collab model.Collab
		database.Db.Where("collab_id = ?", collabID).First(&collab)

		var owner model.ScholarizeUser
		database.Db.Where("user_id = ?", collab.OwnerID).First(&owner)

		// Get user detail
		user := permission.GetUserById(memberID)

		fmt.Printf("Invite token for member %d: %s\n", memberID, token)

		// SMTP
		inviteCollabSMTP := viper.GetBool("mailsmtp.toggle.userpanel.collab_invite")
		fmt.Printf("Invite Collab SMTP: %v\n", inviteCollabSMTP)
		if inviteCollabSMTP {
			// Send email to the new super user
			emailBody := mail.EmailTemplateData{
				PreviewHeader: "Collaboration Invitation - " + collab.CollabName,
				EmailPurpose:  "You have been invited to join collaboration group " + collab.CollabName + " By " + owner.UserName + " !",
				ActionURL:     viper.GetString("client.userpanel") + "/joincollab/" + token,
				Action:        "Join collaboration group",
				EmailEnding:   "If you believe this is a mistake, please ignore this message.",
			}

			// Customize the email template
			emailBodyData, err := mail.CustomizeHTML(emailBody)
			if err != nil {
				return
			}

			// Send email to user
			errSending := mail.SendEmail(user.UserEmail, "Scholarize - You have been invited to join collaboration", emailBodyData)
			if errSending != nil {
				return
			}
		}

		// Insert notification
		notiErr := notification.InsertCollabInviteNotification(collabID, memberID, token, collab.CollabName, owner.UserName)
		if notiErr != nil {
			return
		}
	}
}

// For accepting the invitation
func ValidateInviteLink(tokenString string) (int, int, error) {
	// Validate the token
	userID, collabID, err := ExtractInviteTokenData(tokenString)
	if err != nil {
		return 0, 0, err
	}

	// Join the table invite and invite collab
	var inviteData model.Invite
	database.Db.Select("invite.invite_id, invite.user_id, invite.invite_token").
		Joins("JOIN invitecollab ON invitecollab.invite_id = invite.invite_id").
		Where("invite.user_id = ? AND collab_id = ?", userID, collabID).
		First(&inviteData)

	// Check if the invite exists
	if inviteData.InviteID == 0 {
		return 0, 0, err
	}

	return userID, collabID, nil
}

// Handle join collab
func HandleJoinCollab(c *gin.Context) {
	// Get the token from Param
	inviteToken := c.Param("token")
	if inviteToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invite token is required"})
		return
	}

	// Validate the token
	userID, collabID, err := ValidateInviteLink(inviteToken)
	if userID == 0 || collabID == 0 || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invite token"})
		return
	}

	// Check is Collab is archived
	var collab model.Collab
	database.Db.Where("collab_id = ?", collabID).First(&collab)
	if collab.CollabID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab does not exist"})
		return
	}
	if collab.CollabArchiveStatus {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Collab is archived"})
		return
	}

	// Check if user exists
	var user model.ScholarizeUser
	database.Db.Where("user_id = ?", userID).First(&user)
	if user.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User does not exist"})
		return
	}

	// Check if the user is not the owner
	database.Db.Where("collab_id = ?", collabID).First(&collab)
	if collab.OwnerID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are the owner of the group"})
		return
	}

	// Check if the user already joined the group or not with joined status true
	var collabMember model.CollabMember
	database.Db.Where("collab_id = ? AND user_id = ?", collabID, userID).First(&collabMember)
	if collabMember.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You are not invited to the group"})
		return
	}

	if collabMember.Joined {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You have already joined the group"})
		return
	}

	// Update the joined status
	if err := updateJoinedStatus(collabID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update the joined status"})
		return
	}

	// Delete the invitation and invite collab detail of the user
	if err := deleteInviteAndRelatedData(userID, inviteToken, collabID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete the invite"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "You have successfully joined " + collab.CollabName + "!",
	})
}

// Delete the invite and related data
func deleteInviteAndRelatedData(userID int, inviteToken string, collabID int) error {
	// Start a transaction
	tx := database.Db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// Find and delete the invite
	var inviteData model.Invite
	if err := tx.Where("user_id = ? AND invite_token = ?", userID, inviteToken).First(&inviteData).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&inviteData).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Find and delete the invite collab data
	var inviteCollabData model.InviteCollab
	if err := tx.Where("invite_id = ? AND collab_id = ?", inviteData.InviteID, collabID).First(&inviteCollabData).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&inviteCollabData).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return err
	}

	return nil
}

func addCollabMember(collabID int, userID int) error {
	// Add the user as a member
	newCollabMember := model.CollabMember{
		CollabID: collabID,
		UserID:   userID,
		Joined:   false,
	}
	err := database.Db.Create(&newCollabMember).Error
	if err != nil {
		return err
	}
	return nil
}

func updateJoinedStatus(collabID int, userID int) error {
	// Update the joined status
	err := database.Db.Model(&model.CollabMember{}).
		Where("collab_id = ? AND user_id = ?", collabID, userID).
		Update("joined", true).Error
	if err != nil {
		return err
	}
	return nil
}

// SMTP
// Added Notification
func CreateOrUpdateInvitesForMembers(tx *gorm.DB, collabID int, memberIDs []int) error {
	// Get collab detail
	var collab model.Collab
	if err := tx.Where("collab_id = ?", collabID).First(&collab).Error; err != nil {
		return fmt.Errorf("Error getting collab detail: %v", err)
	}

	var owner model.ScholarizeUser
	database.Db.Where("user_id = ?", collab.OwnerID).First(&owner)

	for _, memberID := range memberIDs {
		token, err := GenerateInviteToken(memberID, collabID)
		if err != nil {
			return fmt.Errorf("Error generating token for member %d: %v", memberID, err)
		}

		// Get user email
		user := permission.GetUserById(memberID)
		if user == nil {
			return fmt.Errorf("Error getting user detail for member %d", memberID)
		}

		// Check if there is an existing invite
		var invite model.Invite
		result := tx.Where("user_id = ? AND invite_id IN (SELECT invite_id FROM invitecollab WHERE collab_id = ?)", memberID, collabID).First(&invite)

		if result.Error == nil {
			// Update the token if the invite already exists
			invite.InviteToken = token
			if err := tx.Save(&invite).Error; err != nil {
				return fmt.Errorf("Error updating invite token for member %d: %v", memberID, err)
			}
		} else {
			// Create a new invite and link it to the collaboration
			inviteData := model.Invite{
				InviteToken: token,
				UserID:      memberID,
			}
			if err := tx.Create(&inviteData).Error; err != nil {
				return fmt.Errorf("Error creating invite data for member %d: %v", memberID, err)
			}

			inviteCollabData := model.InviteCollab{
				InviteID: inviteData.InviteID,
				CollabID: collabID,
			}
			if err := tx.Create(&inviteCollabData).Error; err != nil {
				return fmt.Errorf("Error creating invite collab data for member %d: %v", memberID, err)
			}
		}

		fmt.Printf("Invite token for member %d: %s\n", memberID, token)

		// SMTP
		inviteCollabSMTP := viper.GetBool("mailsmtp.toggle.userpanel.collab_invite")
		fmt.Printf("Invite Collab SMTP: %v\n", inviteCollabSMTP)
		// Send email to the new collab member
		if inviteCollabSMTP {
			emailBody := mail.EmailTemplateData{
				PreviewHeader: "Collaboration Invitation - " + collab.CollabName,
				EmailPurpose:  "You have been invited to join collaboration group " + collab.CollabName + " By " + owner.UserName + " !",
				ActionURL:     viper.GetString("client.userpanel") + "/joincollab/" + token,
				Action:        "Join collaboration group",
				EmailEnding:   "If you believe this is a mistake, please ignore this message.",
			}

			// Customize the email template
			emailBodyData, err := mail.CustomizeHTML(emailBody)
			if err != nil {
				return fmt.Errorf("Error customizing email: %v", err)
			}

			// Send email to user
			errSending := mail.SendEmail(user.UserEmail, "Scholarize - You have been invited to join collaboration", emailBodyData)
			if errSending != nil {
				return fmt.Errorf("Error sending email to user %s: %v", user.UserEmail, errSending)
			}
		}

		// Insert notification
		notiErr := notification.InsertCollabInviteNotification(collabID, memberID, token, collab.CollabName, owner.UserName)
		if notiErr != nil {
			return fmt.Errorf("Error inserting notification: %v", notiErr)
		}
	}
	return nil
}
