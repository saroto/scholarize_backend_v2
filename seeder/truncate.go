package seeder

import (
	"gorm.io/gorm"
)

// Truncate all tables
func TruncateAllTables(db *gorm.DB) error {
	tables := []string{
		"scholarize_user",
		"admin_reset_password",
		"token",
		"role",
		"userrole",
		"permission_category",
		"permission",
		"rolepermission",
		"department",
		"research_type",
		"research_paper",
		"researchpaperdepartment",
		"fulltext",
		"cleantext",
		"collab",
		"collab_member",
		"collabmemberpermission",
		"collab_permission_category",
		"collab_permission",
		"invite",
		"invitecollab",
		"task",
		"task_status",
		"statustask",
		"taskassignee",
		"comment",
		"subtask",
		"taskcomment",
		"subtaskcomment",
		"file",
		"folder",
		"filefolder",
		"schedule",
		"schedulecollab",
		"repeat_group",
		"schedulerepeat",
	}

	// Truncate each table
	for _, table := range tables {
		if err := db.Exec("TRUNCATE TABLE " + table + " RESTART IDENTITY CASCADE").Error; err != nil {
			return err
		}
	}

	return nil
}
