package constant

import (
	"root/model"
	"time"
)

// Auth structs

type GoogleTokenInfo struct {
	AccessToken string `json:"access_token"`
	Email       string `json:"email"`
	Name        string `json:"name"`
}

type OAuthResponse struct {
	UserCredentials UserCredentials `json:"user_credentials"`
}

type UserCredentials struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	ProfileURL string `json:"profile_url"`
}

type AdminCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type NewPassword struct {
	NewPassword string `json:"new_password"`
}

type UserRoleJson struct {
	Role string `json:"role"`
}

type UserPermissionList struct {
	PermissionCate  string   `json:"permission_category_name"`
	UserPermissions []string `json:"permissions"`
}

type UserPermission struct {
	PermissionCate string `json:"permission_category_name"`
	PermissionName string `json:"permission_name"`
}

// Admin page structs
type AdminDepartmentInfo struct {
	Department       DepartmentInfo `json:"department"`
	DepartmentHODs   []SimpleUser   `json:"department_hods"`
	EligibleHODUsers []SimpleUser   `json:"eligible_hod_users"`
}

type SimpleUser struct {
	UserID         int    `json:"user_id"`
	UserName       string `json:"user_name"`
	UserProfileImg string `json:"user_profile_img"`
	UserEmail      string `json:"user_email"`
}

type UserList struct {
	UserId         int        `json:"user_id"`
	UserName       string     `json:"user_name"`
	UserEmail      string     `json:"user_email"`
	UserProfileImg string     `json:"user_profile_img"`
	UserStatus     bool       `json:"user_status"`
	UserRole       model.Role `json:"user_role"`
}

type DepartmentPaperCount struct {
	DepartmentID   uint   `gorm:"column:department_id"`
	DepartmentName string `gorm:"column:department_name"`
	DepartmentTag  string `gorm:"column:department_tag"`
	PaperCount     int    `gorm:"column:paper_count"`
}

type DashboardData struct {
	TotalRegisteredUserCount   int64
	TotalAdvisorCount          int64
	TotalHodCount              int64
	TotalUserCount             int64
	TotalDepartmentCount       int64
	TotalActiveDepartmentCount int64
	TotalPaperCount            int64
	TotalPublishedPaperCount   int64
	TotalGroupCount            int64
	TotalActiveGroupCount      int64
	TotalArchivedGroupCount    int64
	PaperCountByDepartment     []DepartmentPaperCount
}

type RolePermissionAdminPanel struct {
	RoleID          int                `json:"role_id"`
	RoleName        string             `json:"role_name"`
	RoleColor       string             `json:"role_color"`
	RolePermissions []model.Permission `json:"role_permissions"`
}

// Repo structs

type LaravelUploadResponse struct {
	Message       string `json:"message"`
	FileName      string `json:"file_name"`
	OptimizedText string `json:"optimized_text"`
}

type AdminResearchPaperList struct {
	ResearchPaperID     int                      `json:"research_paper_id"`
	Title               string                   `json:"research_title"`
	Author              string                   `json:"author"`
	UserInfo            SimpleUser               `json:"user_info"`
	PublishedAt         string                   `json:"published_at"`
	ResearchPaperStatus string                   `json:"research_paper_status"`
	DepartmentInfo      []DepartmentInfoForPaper `json:"department_info"`
}

type DepartmentInfo struct {
	DepartmentID     int    `json:"department_id"`
	DepartmentName   string `json:"department_name"`
	DepartmentTag    string `json:"department_tag"`
	DepartmentColor  string `json:"department_color"`
	DepartmentStatus bool   `json:"department_status"`
}

type DepartmentInfoForPaper struct {
	DepartmentID    int    `json:"department_id"`
	DepartmentName  string `json:"department_name"`
	DepartmentTag   string `json:"department_tag"`
	DepartmentColor string `json:"department_color"`
}

type QueryResearchPaperInfo struct {
	ResearchPaperID int                `json:"research_paper_id"`
	PublicID        string             `json:"public_id"`
	Title           string             `json:"research_title"`
	Departments     []DepartmentInfo   `json:"departments"`
	PDFPath         string             `json:"pdf_path"`
	Type            model.ResearchType `json:"research_type"`
	Author          string             `json:"author"`
	PublishedAt     string             `json:"published_at"`
	Abstract        string             `json:"abstract"`
	UserID          int                `json:"user_id"`
}

// Collab Structs

type CollabLists struct {
	ActiveCollabs   []CollabDetails `json:"active"`
	ArchivedCollabs []CollabDetails `json:"archived"`
}

type CollabDetails struct {
	CollabID            int          `json:"collab_id"`
	CollabName          string       `json:"collab_name"`
	CollabArchiveStatus bool         `json:"collab_archive_status"`
	CollabColor         string       `json:"collab_color"`
	Owner               SimpleUser   `json:"owner_info"`
	Members             []SimpleUser `json:"members_info"`
	Actions             []string     `json:"actions"`
	TotalMembers        int          `json:"total_members"`
}

type IndividualCollabDetails struct {
	CollabID            int          `json:"collab_id"`
	CollabName          string       `json:"collab_name"`
	CollabArchiveStatus bool         `json:"collab_archive_status"`
	Owner               SimpleUser   `json:"owner_info"`
	Members             []SimpleUser `json:"members_info"`
	PendingInvites      []SimpleUser `json:"pending_invites"`
}

// Task struct
type TaskDetails struct {
	TaskID          int          `json:"task_id"`
	TaskTitle       string       `json:"task_title"`
	TaskPriority    bool         `json:"task_priority"`
	StatusUpdatedAt time.Time    `json:"status_updated_at"`
	TaskAssignee    []SimpleUser `json:"task_assignee"`
}

type TaskList struct {
	TaskStatusID    int           `json:"task_status_id"`
	TaskStatusName  string        `json:"task_status_name"`
	TaskStatusColor string        `json:"task_status_color"`
	TaskDetails     []TaskDetails `json:"task_details"`
}

type TaskCommentDetails struct {
	CommentID   int        `json:"comment_id"`
	CommentText string     `json:"comment_text"`
	CommentedBy SimpleUser `json:"commented_by"`
	CommentedAt string     `json:"commented_at"`
}

// File Struct
type FileDetails struct {
	FileID      int        `json:"file_id"`
	FileName    string     `json:"file_name"`
	FileSize    string     `json:"file_size"`
	UpdatedAt   string     `json:"updated_at"`
	FilePath    string     `json:"file_path"`
	UserDetails SimpleUser `json:"user_details"`
}

type FolderDetails struct {
	FolderID   int           `json:"folder_id"`
	FolderName string        `json:"folder_name"`
	TotalFiles int           `json:"total_files"`
	Files      []FileDetails `json:"files"`
}

// Schedule in Collab struct
type NewSchedule struct {
	ScheduleTitle     string    `gorm:"column:schedule_title;type:text"`
	ScheduleTimeStart time.Time `gorm:"column:schedule_time_start"`
	ScheduleTimeEnd   time.Time `gorm:"column:schedule_time_end"`
	RepeatInterval    int       `gorm:"column:repeat_interval"`
	RepeatGroup       string    `gorm:"column:repeat_group;type:text"`
}

type UpdateSchedule struct {
	ScheduleTitle     string    `gorm:"column:schedule_title;type:text"`
	ScheduleTimeStart time.Time `gorm:"column:schedule_time_start"`
	ScheduleTimeEnd   time.Time `gorm:"column:schedule_time_end"`
}

type UpdateRepeatSchedule struct {
	ScheduleTitle     string    `gorm:"column:schedule_title;type:text"`
	ScheduleTimeStart time.Time `gorm:"column:schedule_time_start"`
	ScheduleTimeEnd   time.Time `gorm:"column:schedule_time_end"`
	RepeatGroup       string    `gorm:"column:repeat_group;type:text"`
}

// Schedule in User struct
type ScheduleDetail struct {
	ScheduleID        int       `gorm:"column:schedule_id"`
	ScheduleTitle     string    `gorm:"column:schedule_title;type:text"`
	ScheduleTimeStart time.Time `gorm:"column:schedule_time_start"`
	ScheduleTimeEnd   time.Time `gorm:"column:schedule_time_end"`
	RepeatInterval    int       `gorm:"column:repeat_interval"`
	RepeatGroup       string    `gorm:"column:repeat_group;type:text"`
	UserID            int       `gorm:"column:user_id"`
	UserName          string    `gorm:"column:user_name"`
}

type CollabSchedule struct {
	CollabID            int              `json:"collab_id"`
	CollabName          string           `json:"collab_name"`
	CollabArchiveStatus bool             `json:"collab_archive_status"`
	CollabColor         string           `json:"collab_color"`
	ScheduleDetails     []ScheduleDetail `json:"schedules"`
}

type NewScheduleForCollabs struct {
	CollabIDs         []int     `json:"collab_ids"`
	ScheduleTitle     string    `gorm:"column:schedule_title;type:text"`
	ScheduleTimeStart time.Time `gorm:"column:schedule_time_start"`
	ScheduleTimeEnd   time.Time `gorm:"column:schedule_time_end"`
	RepeatInterval    int       `gorm:"column:repeat_interval"`
	RepeatGroup       string    `gorm:"column:repeat_group;type:text"`
}

type PythonResponse struct {
	Message          string      `json:"message"`
	Data             interface{} `json:"data"`
	PDFProcessStatus string      `json:"pdf_process_status"`
	ErrorType        string      `json:"error_type,omitempty"`
	StatusCode       int         `json:"status_code,omitempty"`
}
