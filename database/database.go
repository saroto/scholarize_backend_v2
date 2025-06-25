package database

import (
	"fmt"
	"log"
	"root/model"

	"gorm.io/driver/postgres"

	_ "github.com/lib/pq"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

// var db *sql.DB
var Db *gorm.DB

func ConnectDB() *gorm.DB {
	var err error
	dbURL := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Bangkok",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.user"),
		viper.GetString("database.password"),
		viper.GetString("database.dbname"))
	// Open connection to the database
	Db, err = gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	return Db
}

func CloseDBConnection() {
	//var err error
	sqlDB, err := Db.DB()
	if err != nil {
		log.Fatalln(err)
	}
	sqlDB.Close()
}

// AutoMigrateDB will create the tables based on the models of Scholarize
func AutoMigrateDB() error {
	migration := viper.GetBool("database.auto_migration")
	if !migration {
		fmt.Println("Auto migration is disabled")
		return nil
	}

	models := []interface{}{
		&model.ScholarizeUser{}, &model.AdminResetPassword{}, &model.Token{},
		&model.Role{}, &model.UserRole{}, &model.PermissionCategory{},
		&model.Permission{}, &model.RolePermission{}, &model.ResearchType{},
		&model.ResearchPaper{}, &model.Fulltext{}, &model.Cleantext{},
		&model.Department{}, &model.DepartmentHead{}, &model.ResearchPaperDepartment{},
		&model.Collab{}, &model.Invite{}, &model.InviteCollab{},
		&model.CollabMember{}, &model.CollabPermissionCategory{},
		&model.CollabPermission{}, &model.CollabMemberPermission{},
		&model.Task{}, &model.TaskStatus{}, &model.StatusTask{},
		&model.TaskAssignee{}, &model.Comment{}, &model.Subtask{},
		&model.TaskComment{}, &model.SubtaskComment{}, &model.File{},
		&model.Folder{}, &model.FileFolder{}, &model.Schedule{},
		&model.ScheduleCollab{}, &model.Notification{},
		&model.ChatSession{}, &model.JobQueue{},
	}
	for _, model := range models {
		if err := Db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}
	// Drop if extist
	//Db.Exec("DROP INDEX IF EXISTS idx_research_fulltext")
	//Db.Exec("DROP INDEX IF EXISTS idx_cleantext_fulltext")

	// Add GIN indexes for full-text search
	//Db.Exec("CREATE INDEX idx_research_fulltext ON research_paper USING GIN (to_tsvector('english', research_title || ' ' || abstract || ' ' || tag || ' ' || author || ' ' || advisor))")
	//Db.Exec("CREATE INDEX idx_cleantext_fulltext ON cleantext USING GIN (to_tsvector('english', cleantext_content))")

	fmt.Printf("Successfully migrated all models\n")
	return nil
}
