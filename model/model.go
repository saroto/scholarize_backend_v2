package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type ScholarizeUser struct {
	UserID         int       `gorm:"primaryKey;column:user_id"`
	UserName       string    `gorm:"column:user_name"`
	UserPassword   string    `gorm:"column:user_password"`
	UserEmail      string    `gorm:"column:user_email;unique"`
	UserProfileImg string    `gorm:"column:user_profile_img;type:text"`
	UserStatus     bool      `gorm:"column:user_status;default:true"`
	UserCreatedAt  time.Time `gorm:"column:user_created_at;type:timestamp"`
	UserUpdatedAt  time.Time `gorm:"column:user_updated_at;type:timestamp"`
}

func (ScholarizeUser) TableName() string {
	return "scholarize_user"
}

type AdminResetPassword struct {
	UserID           int       `gorm:"column:user_id"`
	IsBoarded        bool      `gorm:"column:is_boarded;default:false"`
	ResetToken       string    `gorm:"column:reset_token"`
	ResetTokenExpiry time.Time `gorm:"column:reset_token_expiry"`
}

// TableName sets the insert table name for this struct type
func (AdminResetPassword) TableName() string {
	return "admin_reset_password"
}

type Token struct {
	TokenID     int       `gorm:"primaryKey;column:token_id"`
	UserID      int       `gorm:"column:user_id"`
	ApiToken    string    `gorm:"column:api_token;type:text"`
	TokenExpire time.Time `gorm:"column:token_expire;type:timestamp"`
}

func (Token) TableName() string {
	return "token"
}

type Role struct {
	RoleID    int    `gorm:"primaryKey;column:role_id"`
	RoleName  string `gorm:"column:role_name"`
	RoleColor string `gorm:"column:role_color;default:#333333"`
}

func (Role) TableName() string {
	return "role"
}

type UserRole struct {
	UserRoleID int `gorm:"primaryKey;column:userrole_id"`
	UserID     int `gorm:"column:user_id"`
	RoleID     int `gorm:"column:role_id"`
}

func (UserRole) TableName() string {
	return "userrole"
}

type PermissionCategory struct {
	PermissionCategoryID   int    `gorm:"primaryKey;column:permission_category_id"`
	PermissionCategoryName string `gorm:"column:permission_category_name"`
}

func (PermissionCategory) TableName() string {
	return "permission_category"
}

type Permission struct {
	PermissionID         int    `gorm:"primaryKey;column:permission_id"`
	PermissionName       string `gorm:"column:permission_name"`
	PermissionCategoryID int    `gorm:"column:permission_category_id"`
}

func (Permission) TableName() string {
	return "permission"
}

type RolePermission struct {
	RolePermissionID int `gorm:"primaryKey;column:rolepermission_id"`
	RoleID           int `gorm:"column:role_id"`
	PermissionID     int `gorm:"column:permission_id"`
}

func (RolePermission) TableName() string {
	return "rolepermission"
}

type ResearchType struct {
	ResearchTypeID     int    `gorm:"primaryKey;column:research_type_id"`
	ResearchTypeName   string `gorm:"column:research_type_name"`
	ResearchTypeStatus bool   `gorm:"column:research_type_status;default:true"`
}

func (ResearchType) TableName() string {
	return "research_type"
}

type ResearchPaper struct {
	ResearchPaperID     int       `gorm:"primaryKey;column:research_paper_id"`
	PublicID            string    `gorm:"column:public_id"`
	ResearchTypeID      int       `gorm:"column:research_type_id"`
	ResearchTitle       string    `gorm:"column:research_title;type:text"`
	Abstract            string    `gorm:"column:abstract;type:text"`
	Tag                 string    `gorm:"column:tag;type:text"`
	Author              string    `gorm:"column:author;type:text"`
	Advisor             string    `gorm:"column:advisor;type:text"`
	PDFPath             string    `gorm:"column:pdf_path;type:text"`
	ResearchPaperStatus string    `gorm:"column:research_paper_status"`
	RejectedReason      string    `gorm:"column:rejected_reason" json:"rejected_reason,omitempty"`
	FulltextID          *int       `gorm:"column:fulltext_id"`
	CleantextID         *int       `gorm:"column:cleantext_id"`
	UserID              int       `gorm:"column:user_id"`
	SubmittedAt         time.Time `gorm:"column:submitted_at;type:timestamp"`
	PublishedAt         time.Time `gorm:"column:published_at;type:timestamp"`
}

func (ResearchPaper) TableName() string {
	return "research_paper"
}

type Fulltext struct {
	FulltextID      int    `gorm:"primaryKey;column:fulltext_id"`
	FulltextContent string `gorm:"column:fulltext_content;type:text"`
}

func (Fulltext) TableName() string {
	return "fulltext"
}

type Cleantext struct {
	CleantextID      int    `gorm:"primaryKey;column:cleantext_id"`
	CleantextContent string `gorm:"column:cleantext_content;type:text"`
}

func (Cleantext) TableName() string {
	return "cleantext"
}

type Department struct {
	DepartmentID     int    `gorm:"primaryKey;column:department_id"`
	DepartmentName   string `gorm:"column:department_name"`
	DepartmentTag    string `gorm:"column:department_tag;size:16"`
	DepartmentColor  string `gorm:"column:department_color;size:9"`
	DepartmentStatus bool   `gorm:"column:department_status;default:false"`
}

func (Department) TableName() string {
	return "department"
}

type DepartmentHead struct {
	DepartmentHeadID int `gorm:"primaryKey;column:departmenthead_id"`
	DepartmentID     int `gorm:"column:department_id"`
	UserID           int `gorm:"column:user_id"`
}

func (DepartmentHead) TableName() string {
	return "departmenthead"
}

type ResearchPaperDepartment struct {
	ResearchPaperDepartmentID int `gorm:"primaryKey;column:researchpaperdepartment_id"`
	DepartmentID              int `gorm:"column:department_id"`
	ResearchPaperID           int `gorm:"column:research_paper_id"`
}

func (ResearchPaperDepartment) TableName() string {
	return "researchpaperdepartment"
}

type Collab struct {
	CollabID            int    `gorm:"primaryKey;column:collab_id"`
	CollabName          string `gorm:"column:collab_name"`
	CollabArchiveStatus bool   `gorm:"column:collab_archive_status;default:false"`
	CollabColor         string `gorm:"column:collab_color"`
	OwnerID             int    `gorm:"column:owner_id"`
}

func (Collab) TableName() string {
	return "collab"
}

type Invite struct {
	InviteID    int    `gorm:"primaryKey;column:invite_id"`
	InviteToken string `gorm:"column:invite_token;type:text"`
	UserID      int    `gorm:"column:user_id"`
}

func (Invite) TableName() string {
	return "invite"
}

type InviteCollab struct {
	InviteCollabID int `gorm:"primaryKey;column:invitecollab_id"`
	InviteID       int `gorm:"column:invite_id"`
	CollabID       int `gorm:"column:collab_id"`
}

func (InviteCollab) TableName() string {
	return "invitecollab"
}

type CollabMember struct {
	CollabMemberID int  `gorm:"primaryKey;column:collab_member_id"`
	CollabID       int  `gorm:"column:collab_id"`
	UserID         int  `gorm:"column:user_id"`
	Joined         bool `gorm:"column:joined;default:false"`
}

func (CollabMember) TableName() string {
	return "collab_member"
}

type CollabPermissionCategory struct {
	CollabPermissionCategoryID   int    `gorm:"primaryKey;column:collab_permission_category_id"`
	CollabPermissionCategoryName string `gorm:"column:collab_permission_category_name;type:text"`
}

func (CollabPermissionCategory) TableName() string {
	return "collab_permission_category"
}

type CollabPermission struct {
	CollabPermissionID         int    `gorm:"primaryKey;column:collab_permission_id"`
	CollabPermissionName       string `gorm:"column:collab_permission_name"`
	CollabPermissionCategoryID int    `gorm:"column:collab_permission_category_id"`
}

func (CollabPermission) TableName() string {
	return "collab_permission"
}

type CollabMemberPermission struct {
	CollabMemberPermID int `gorm:"primaryKey;column:collabmemberperm_id"`
	CollabID           int `gorm:"column:collab_id"`
	CollabPermissionID int `gorm:"column:collab_permission_id"`
}

func (CollabMemberPermission) TableName() string {
	return "collabmemberpermission"
}

type Task struct {
	TaskID          int       `gorm:"primaryKey;column:task_id"`
	TaskTitle       string    `gorm:"column:task_title;type:text"`
	TaskPriority    bool      `gorm:"column:task_priority;default:false"`
	CollabID        int       `gorm:"column:collab_id"`
	StatusUpdatedAt time.Time `gorm:"column:status_updated_at"`
}

func (Task) TableName() string {
	return "task"
}

type TaskStatus struct {
	TaskStatusID    int    `gorm:"primaryKey;column:task_status_id"`
	TaskStatusName  string `gorm:"column:task_status_name"`
	TaskStatusColor string `gorm:"column:task_status_color"`
}

func (TaskStatus) TableName() string {
	return "task_status"
}

type StatusTask struct {
	StatusTaskID int `gorm:"primaryKey;column:statustask_id"`
	TaskID       int `gorm:"column:task_id"`
	TaskStatusID int `gorm:"column:task_status_id"`
}

func (StatusTask) TableName() string {
	return "statustask"
}

type TaskAssignee struct {
	TaskAssigneeID int `gorm:"primaryKey;column:taskassignee_id"`
	UserID         int `gorm:"column:user_id"`
	TaskID         int `gorm:"column:task_id"`
}

func (TaskAssignee) TableName() string {
	return "taskassignee"
}

type Comment struct {
	CommentID   int       `gorm:"primaryKey;column:comment_id"`
	CommentText string    `gorm:"column:comment_text;type:text"`
	UserID      int       `gorm:"column:user_id"`
	CommentAt   time.Time `gorm:"column:commented_at"`
}

func (Comment) TableName() string {
	return "comment"
}

type Subtask struct {
	SubtaskID    int    `gorm:"primaryKey;column:subtask_id"`
	SubtaskTitle string `gorm:"column:subtask_title;type:text"`
	TaskID       int    `gorm:"column:task_id"`
}

func (Subtask) TableName() string {
	return "subtask"
}

type TaskComment struct {
	TaskCommentID int `gorm:"primaryKey;column:taskcomment_id"`
	CommentID     int `gorm:"column:comment_id"`
	TaskID        int `gorm:"column:task_id"`
}

func (TaskComment) TableName() string {
	return "taskcomment"
}

type SubtaskComment struct {
	SubtaskCommentID int `gorm:"primaryKey;column:subtaskcomment_id"`
	CommentID        int `gorm:"column:comment_id"`
	SubtaskID        int `gorm:"column:subtask_id"`
}

func (SubtaskComment) TableName() string {
	return "subtaskcomment"
}

type File struct {
	FileID    int       `gorm:"primaryKey;column:file_id"`
	FileName  string    `gorm:"column:file_name;type:text"`
	FileSize  string    `gorm:"column:file_size;type:text"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	UserID    int       `gorm:"column:user_id"`
	CollabID  int       `gorm:"column:collab_id"`
	FilePath  string    `gorm:"column:file_path;type:text"`
}

func (File) TableName() string {
	return "file"
}

type Folder struct {
	FolderID   int    `gorm:"primaryKey;column:folder_id"`
	FolderName string `gorm:"column:folder_name;type:text"`
}

func (Folder) TableName() string {
	return "folder"
}

type FileFolder struct {
	FileFolderID int `gorm:"primaryKey;column:filefolder_id"`
	FolderID     int `gorm:"column:folder_id"`
	FileID       int `gorm:"column:file_id"`
}

func (FileFolder) TableName() string {
	return "filefolder"
}

type Schedule struct {
	ScheduleID        int       `gorm:"primaryKey;column:schedule_id"`
	ScheduleTitle     string    `gorm:"column:schedule_title;type:text"`
	ScheduleTimeStart time.Time `gorm:"column:schedule_time_start"`
	ScheduleTimeEnd   time.Time `gorm:"column:schedule_time_end"`
	UserID            int       `gorm:"column:user_id"`
	RepeatInterval    int       `gorm:"column:repeat_interval"`
	RepeatGroup       string    `gorm:"column:repeat_group;type:text"`
	CreatedAt         time.Time `gorm:"column:created_at"`
}

func (Schedule) TableName() string {
	return "schedule"
}

type ScheduleCollab struct {
	ScheduleCollabID int `gorm:"primaryKey;column:schedulecollab_id"`
	ScheduleID       int `gorm:"column:schedule_id"`
	CollabID         int `gorm:"column:collab_id"`
}

func (ScheduleCollab) TableName() string {
	return "schedulecollab"
}

type Notification struct {
	NotificationID  int           `gorm:"primaryKey;column:notification_id"`
	NotificationAt  time.Time     `gorm:"column:notification_at"`
	NotificationMsg string        `gorm:"column:notification_msg;type:text"`
	UserIDs         pq.Int64Array `gorm:"column:user_ids;type:integer[]"`
	UserReads       pq.Int64Array `gorm:"column:user_reads;type:integer[]"`
	Link            string        `gorm:"column:link;type:text;default:null"`
	IsCollabInvite  bool          `gorm:"column:is_collab_invite;default:false"`
	InviteToken     string        `gorm:"column:invite_token;type:text;default:null"`
}

func (Notification) TableName() string {
	return "notification"
}


type ChatSession struct {
    SessionID     uuid.UUID       `gorm:"primaryKey;column:session_id"`
    UserID        int             `gorm:"column:user_id"`
    PaperID       int             `gorm:"column:paper_id"` 
    Message       json.RawMessage `gorm:"column:message;type:json"`
    CreatedAt     time.Time       `gorm:"column:created_at"`
    UpdatedAt     time.Time       `gorm:"column:updated_at"`
    // User          ScholarizeUser  `gorm:"foreignKey:UserID;references:UserID"`
    // ResearchPaper ResearchPaper   `gorm:"foreignKey:PaperID;references:ReesearchPaperID"` 
}

// type ChatHistory struct {
// 	HistoryID uuid.UUID     `gorm:"primaryKey;column:history_id"`
// 	SessionID uuid.UUID     `gorm:"column:session_id"`
// 	Message json.RawMessage `gorm:"column:message;type:json"`
// 	CreatedAt time.Time 	`gorm:"column:created_at"`
// 	UpdatedAt time.Time 	`gorm:"column:updated_at"`

// 	Session ChatSession 	`gorm:"references:SessionID"`
// }
