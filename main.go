package main

import (
	"fmt"
	"os"
	"root/auth"
	"root/config"
	"root/constant"
	"root/controllers/administrator"
	"root/controllers/chat"
	"root/controllers/collaboration"
	"root/controllers/notification"
	"root/controllers/repository"
	"root/controllers/scheduling"
	"root/database"
	"root/meilisearch"
	"root/middleware"
	"root/seeder"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Set the TZ environment variable to Asia/Bangkok (UTC+7)

	os.Setenv("TZ", "Asia/Bangkok")

	// Ensure the time package uses the correct timezone
	loc, err := time.LoadLocation("Asia/Bangkok")
	if err != nil {
		panic(err)
	}
	time.Local = loc

	config.InitConfig()
	database.ConnectDB()
	defer database.CloseDBConnection()
	meilisearch.InitMeiliSearch()
	// Run database migrations
	err = database.AutoMigrateDB()
	if err != nil {
		fmt.Printf("Failed to migrate the database: %v", err)
	}

	r := gin.Default()

	// Cors
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"*"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowCredentials = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Seed database
	seeder.SeedAllData()

	// Load the HTML templates
	r.LoadHTMLGlob("templates/*.html")

	// API Routes
	api := r.Group("/api")
	{
		// api.POST("/triponzoid", func(c *gin.Context) {
		// 	key := c.PostForm("Th8s4s4mP1eK3y")
		// 	if key == "Th8s4s4mP1eK3y" {
		// 		seeder.TruncateAllTables(database.Db)
		// 		c.JSON(http.StatusOK, gin.H{"message": "All tables truncated!"})
		// 	} else {
		// 		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized access!"})
		// 	}
		// })

		// api.POST("/extracttoken", func(c *gin.Context) {
		// 	a := auth.ExtractToken(c)
		// 	claims, _ := auth.ParseApiToken(a)
		// 	c.JSON(http.StatusOK, gin.H{"token": a, "claims": claims})
		// })

		// api.POST("/extracttokenpass", func(c *gin.Context) {
		// 	a := auth.ExtractToken(c)
		// 	claims, _ := auth.ParseResetPasswordToken(a)
		// 	c.JSON(http.StatusOK, gin.H{"reset_token": a, "claims": claims})
		// })

		api.POST("/login", auth.HandleFrontPanelLogin)
		api.POST("/adminlogin", auth.HandleAdminPanelLogin)
		api.POST("/resetpasswordreq", auth.HandleSendResetPasswordLink)
		api.GET("/resetpassword/:token", auth.HandleAccessResetPasswordPage)
		api.POST("/resetpassword", auth.HandleUpdateAdminPasswordOnReset)

		api.GET("/joincollab/:token", collaboration.HandleJoinCollab)

		api.GET("/browse", repository.HandleHybridSearch)
		api.GET("/browse/:id", repository.HandleGetIndividualPapeprForPublicUser)
		// 		api.GET("/browse", repository.HandleResearchPaperSemanticSearch)
		api.POST("/repository/mypublications/updateStatus", repository.UpdatePaperStatus)
		api.POST("/repository/mypublications/notifyFailPaper", repository.NotifyUserForFailPaper)

	}

	// Group the routes requires Jwt Token
	api.Use(middleware.JwtMiddleware())
	{
		api.GET("/notifications", notification.HandleGetAllNotifications)
		api.POST("/notifications/:notificationID", notification.HandleMarkNotificationAsRead)
		api.POST("/logout", auth.HandleLogout)
	}

	// Repository Routes
	repo := api.Group("/repository")
	{
		repo.GET("/downloadpaper", repository.HandleDownloadResearchPaper)

		// repo.GET("/researchlibrary", repository.HandleResearchPaperSearch)
		// repo.GET("/researchlibrary", repository.HandleResearchPaperSemanticSearch)

		repo.GET("/researchlibrary", repository.HandleHybridSearch)
		repo.GET("/researchlibrary/:id", repository.HandleGetIndividualPaperPage)
		repo.GET("/researchlibrary/inprogress/:id", repository.GetInProgressResearchPapers)

		repo.GET("/uploadform", repository.GetResearchPaperUploadFormData)
		repo.POST("/uploadform", repository.HandleResearchPaperUpload)

		repo.GET("/mypublications", repository.HandleDisplayMyPublishedResearchPapers)
		repo.GET("/mypublications/awaiting/:id", repository.HandlePreviewAwaitingPaper)
		repo.GET("/mypublications/rejected/:id", repository.HandlePreviewRejectedPaper)
		repo.POST("/mypublications/rejected/:id", repository.HandleResubmitRejectedPaper)
		// repo.POST("/mypublications/updateStatus", repository.UpdatePaperStatus)
		// repo.POST("/researchpaper/status", repository.GetPaperStatusForPdfProcessing)
	}

	//Chat Routes
	chatSession := api.Group("/chat")
	{
		chatSession.GET("/session/:sessionID", chat.HandleGetChatSession)
		chatSession.POST("/update-session/:sessionID", chat.UpdateChatSessionSecure)
	}

	// Start using HOD middleware for the REPOSITORY routes
	hod := repo.Use(middleware.HodMiddleware())
	{
		hod.GET("/submissions",
			middleware.RoleHasPermissionMiddleware(constant.ResearchSubmissionApproval),
			repository.QueryResearchPaperByDepartment)
		hod.GET("/submissions/:id",
			middleware.RoleHasPermissionMiddleware(constant.ResearchSubmissionApproval),
			repository.HandlePreviewSubmission)

		hod.GET("/submissions/inprogress",
			middleware.RoleHasPermissionMiddleware(constant.ResearchSubmissionApproval),
			repository.HandleGetJobStatus)
		hod.POST("/submissions/:id",
			middleware.RoleHasPermissionMiddleware(constant.ResearchSubmissionApproval),
			repository.HandleApproveRejectSubmission)
	}

	// Collaborations Routes
	collab := api.Group("/collab")
	{
		// All collab users
		collab.GET("/", collaboration.HandleListCollabs)
		collab.POST("/leavecollab", collaboration.HandleLeaveCollab)

		collab.GET("/edit", collaboration.HandleGetUpdateFormCollab)
		collab.GET("/editquery", collaboration.HandleGetAvailableMembersForCollab)

		// Has permission to create collab
		collabCreate := collab.Group("")
		collabCreate.Use(middleware.RoleHasPermissionMiddleware(constant.CreateGroup))
		{
			collabCreate.GET("/createquery", collaboration.HandleGetAvailableMembersForNewCollab)
			collabCreate.POST("/create", collaboration.HandleCreateCollab)
		}

		collab.POST("/updatepermissions", collaboration.HandleUpdatePermissionForCollabMembers)

		collabOwner := collab.Group("")
		collabOwner.GET("/permissions", collaboration.HandleGetPermissionForCollabMembers)
		collabOwner.Use(middleware.IsPOSTCollabOwnerMiddleware())
		{
			collabOwner.POST("/updatecollab", collaboration.HandleUpdateCollab)
			collabOwner.POST("/deletecollab", collaboration.HandleDeleteCollab)
			collabOwner.POST("/archivecollab", collaboration.HandleArchiveCollab)
			collabOwner.POST("/removependingmember", collaboration.HandleRemovePendingMember)
		}
	}

	// Start using middleware for the COLLAB routes
	collab.Use(middleware.IsCollabOwnerOrMemberMiddleware())
	{
		// Get the collab details
		collab.GET("/:collab_id/get", collaboration.HandleGetCollabDetails)
		collab.GET("/:collab_id/getmembers", collaboration.HandleGetAllCollabMembers)

		// Collab Task
		task := collab.Group("/:collab_id/task")
		{
			// Normal get task routes
			task.GET("/getalltasks", collaboration.HandleGetAllTasks)
			task.GET("/getassignee", collaboration.HandleGetTaskAssignees)
			task.GET("/getsubtasks", collaboration.HandleGetAllSubtasks)
			task.GET("/gettaskcomments", collaboration.HandleGetTaskComments)
			task.GET("/getsubtaskcomments", collaboration.HandleGetSubtaskComments)

			// Start using middleware for the TASK routes
			taskPost := task.Use(middleware.CollabArchiveStatusMiddleware())
			{
				taskPost.POST("/createtask",
					middleware.CollabMemberHasPermissionMiddleware(constant.CreateTask),
					collaboration.HandleAddNewTask)
				taskPost.POST("/updatepriority",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditTask),
					collaboration.HandleUpdateTaskPriority)
				taskPost.POST("/deletetask",
					middleware.CollabMemberHasPermissionMiddleware(constant.DeleteTask),
					collaboration.HandleDeleteTask)
				taskPost.POST("/updatetaskname",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditTask),
					collaboration.HandleUpdateTaskName)
				taskPost.POST("/updatetaskstatus",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditTask),
					collaboration.HandleUpdateTaskStatus)
				taskPost.POST("/updateassignee",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditTask),
					collaboration.HandleAssignAssigneeToTask)

				taskPost.POST("/createsubtask",
					middleware.CollabMemberHasPermissionMiddleware(constant.CreateTask),
					collaboration.HandleCreateSubtask)
				taskPost.POST("/updatesubtaskname",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditTask),
					collaboration.HandleUpdateSubtaskName)
				taskPost.POST("/deletesubtask",
					middleware.CollabMemberHasPermissionMiddleware(constant.DeleteTask),
					collaboration.HandleDeleteSubtask)

				taskPost.POST("/addtaskcomment",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditTask),
					collaboration.HandleAddCommentToTask)
				taskPost.POST("/addsubtaskcomment",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditTask),
					collaboration.HandleAddCommentToSubtask)
			}
		}

		// File upload routes
		file := collab.Group("/:collab_id/file")
		{
			file.GET("/getfolders", collaboration.HandleGetAllFolders)
			file.GET("/downloadfile", collaboration.HandleDownloadFile)
			file.GET("/getfiles", collaboration.HandleGetFileDetailsByFolderOfCollab)

			// Start using middleware for the FILE routes
			filePost := file.Use(middleware.CollabArchiveStatusMiddleware())
			{
				filePost.POST("/uploadfiles",
					middleware.CollabMemberHasPermissionMiddleware(constant.UploadFile),
					collaboration.HandleUploadFilesToCollab)
				filePost.POST("/deletefile",
					middleware.CollabMemberHasPermissionMiddleware(constant.DeleteFile),
					collaboration.HandleDeleteCollabFile)
				filePost.POST("/renamefile",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditFile),
					collaboration.HandleHandleRenameFile)
				filePost.POST("/movefile",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditFile),
					collaboration.HandleMoveFile)
			}
		}

		// Collab Schedule
		collabSchedule := collab.Group("/:collab_id/schedule")
		{
			collabSchedule.GET("/get", scheduling.HandleGetAllSchedulesInCollab)

			// Start using middleware for the SCHEDULE routes
			collabSchedulePOST := collabSchedule.Use(middleware.CollabArchiveStatusMiddleware())
			{
				collabSchedulePOST.POST("/create",
					middleware.CollabMemberHasPermissionMiddleware(constant.AddScheduleEvent),
					scheduling.HandleCreateSchedule)

				collabSchedulePOST.PUT("/update",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditScheduleEvent),
					scheduling.HandleUpdateSchedule)
				collabSchedulePOST.PUT("/update-repeat",
					middleware.CollabMemberHasPermissionMiddleware(constant.EditScheduleEvent),
					scheduling.HandleUpdateRepeatedSchedules)

				collabSchedulePOST.DELETE("/delete",
					middleware.CollabMemberHasPermissionMiddleware(constant.DeleteScheduleEvent),
					scheduling.HandleDeleteSchedule)
				collabSchedulePOST.DELETE("/delete-repeat",
					middleware.CollabMemberHasPermissionMiddleware(constant.DeleteScheduleEvent),
					scheduling.HandleDeleteRepeatedSchedules)
			}
		}
	}

	// Schedule route
	schedule := api.Group("/schedule")
	{
		schedule.GET("/get", scheduling.HandleGetUserSchedules)
		schedule.GET("/getwithfilter", scheduling.HandleGetUserSchedulesFilter)
		schedule.POST("/create", scheduling.HandleCreateScheduleForSelectedCollabs)
	}

	// Routes that require admin role
	admin := api.Group("/admin").Use(middleware.AdminOrSuperAdminMiddleware())
	{
		admin.GET("/dashboard", administrator.HandleGetDashboardData)

		admin.GET("/researchpapers", administrator.HandleGetResearchPaperList)
		admin.POST("/updateresearchpaper", administrator.HandleUpdateResearchPaper)
		admin.POST("/updateresearchpapertitle", administrator.HandleUpdateResearchPaperTitle)
		admin.POST("/updateresearchpaperdate", administrator.HandleUpdateResearchPaperDate)

		admin.GET("/researchtypes", administrator.HandleGetResearchTypeList)
		admin.POST("/addresearchtype", administrator.HandleAddResearchType)
		admin.POST("/updateresearchtype", administrator.HandleUpdateResearchType)

		admin.GET("/departments", administrator.GetDepartmentsList)
		admin.POST("/adddepartment", administrator.HandleAddDepartment)
		admin.POST("/updatedepartment", administrator.HandleUpdateDepartment)

		admin.GET("/users", administrator.HandleGetFrontUserList)
		admin.POST("/updateuser", administrator.HandleUpdateFrontUserInfo)

		admin.GET("/admins", administrator.HandleGetAdminList)
		admin.POST("/addadmin", administrator.HandleAddAdmin)
		admin.POST("/removeadmin", administrator.HandleRemoveAdmin)

		admin.POST("/adminboarding", auth.HandleUpdateAdminPasswordOnBoarding)

		admin.GET("/rolepermission", administrator.HandleGetRolePermissionList)
		admin.POST("/updaterolepermission", administrator.HandleUpdateRolePermissions)
	}

	// Routes that require super admin role
	superAdmin := api.Group("/superadmin").Use(middleware.SuperAdminMiddleware())
	{
		superAdmin.POST("/transfersa", administrator.HandleTransferSuperAdmin)
	}

	// go func() {
	// 	log.Print("Start RabbitMQ Consumer")
	// 	// Call the Consumer function from the queue package
	// 	queue.Consumer()
	// }()

	// Start the server on port 2812 on production
	r.Run(":2812")
}
