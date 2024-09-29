package seeder

import (
	"fmt"
	"log"
	"root/constant"
	"root/database"
	"root/generator"
	"root/model"
	"root/permission"

	"github.com/spf13/viper"
)

// Seed Status
var seedStatuses = []string{}

// Check Seed Status
func isSeeded() bool {
	var count int64
	database.Db.Model(&model.ScholarizeUser{}).Count(&count)
	return count > 0
}

func SeedAllData() {
	status := isSeeded()
	if status {
		fmt.Println("Data already seeded")
		return
	}
	SeedRoles()
	SeedPermissionCategories()
	SeedPermissions()
	SeedRolePermission()

	SeedResearchTypes()
	SeedDepartments()
	SeedFolders()

	SeedCollabPermissionCategories()
	SeedCollabPermissions()

	SeedTaskStatus()

	SeedSuperadmin(viper.GetString("superadmin.email"))

	fmt.Println("Data seeded successfully")
}

// Seed all Roles
func SeedRoles() {
	roles := []model.Role{
		{RoleName: constant.UserRole, RoleColor: "#FFC28A"},
		{RoleName: constant.AdvisorRole, RoleColor: "#FFE500"},
		{RoleName: constant.HODRole, RoleColor: "#FFADAD"},
		{RoleName: constant.AdminRole, RoleColor: "#7200BB"},
		{RoleName: constant.SuperAdminRole, RoleColor: "#7200BB"},
	}
	database.Db.CreateInBatches(roles, len(roles))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Roles: %v", roles))
}

// Seed user table, currently only Super admin
func SeedSuperadmin(email string) {
	plainPassword, err := generator.GeneratePlainPassword(8)
	if err != nil {
		log.Fatalf("Failed to generate user-friendly password: %v", err)
	}

	// Hash the password
	hashedPassword, err := generator.HashPassword(plainPassword)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}

	// Create the admin user with the hashed password
	adminUser := model.ScholarizeUser{
		UserName:       "Super Admin",
		UserPassword:   hashedPassword,
		UserEmail:      email,
		UserProfileImg: viper.GetString("default_profile_img.admin"),
	}

	// Create or get the existing user
	database.Db.FirstOrCreate(&adminUser, model.ScholarizeUser{UserEmail: adminUser.UserEmail})

	// Assign Superadmin Role
	superAdminRoleId, err := permission.GetRoleId(constant.SuperAdminRole)
	if err != nil {
		log.Fatalf("Failed to get Super Admin role ID: %v", err)
	}
	userRoleId, err := permission.GetRoleId(constant.UserRole)
	if err != nil {
		log.Fatalf("Failed to get User role ID: %v", err)
	}

	permission.AssignUserRole(adminUser.UserID, superAdminRoleId)
	permission.AssignUserRole(adminUser.UserID, userRoleId)

	// Insert the admin user into reset password table
	adminResetPassword := model.AdminResetPassword{
		UserID: adminUser.UserID,
	}
	database.Db.FirstOrCreate(&adminResetPassword, model.AdminResetPassword{UserID: adminUser.UserID})

	// SMTP for boarding Super Admin if needed
	// newSuperAdminSMTP := viper.GetBool("mailsmtp.toggle.adminpanel.new_superadmin")

	// Output the admin user's email and plain-text password
	fmt.Println("Super Admin Details:")
	fmt.Printf("Email: %s\n", adminUser.UserEmail)
	fmt.Printf("Password: %s\n", plainPassword) // For one-time use only
	fmt.Printf("Role: %d\n", superAdminRoleId)

	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Super Admin: %s", adminUser.UserEmail))
}

// Seed all Permission Categories
func SeedPermissionCategories() {
	categories := []model.PermissionCategory{
		{PermissionCategoryName: "Repository"},
		{PermissionCategoryName: "Collaboration"},
	}
	database.Db.CreateInBatches(categories, len(categories))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Permission Categories: %v", categories))
}

// seed permission with their respective categories
const permCate = "permission_category_name = ?"

func SeedPermissions() {
	// Check id of the permission categories
	var repositoryCat, collaborationCat model.PermissionCategory
	database.Db.Where(permCate, "Repository").First(&repositoryCat)
	database.Db.Where(permCate, "Collaboration").First(&collaborationCat)

	// Define the adding permissions list
	permissions := []model.Permission{
		// Repository permissions
		{PermissionName: "Publish research paper", PermissionCategoryID: repositoryCat.PermissionCategoryID},
		{PermissionName: "Research submission approval", PermissionCategoryID: repositoryCat.PermissionCategoryID},

		// Collaboration permissions
		{PermissionName: "Create group", PermissionCategoryID: collaborationCat.PermissionCategoryID},
	}
	database.Db.CreateInBatches(permissions, len(permissions))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Permissions: %v", permissions))
}

// Seed role permissions
func SeedRolePermission() {
	rolePermissions := map[string][]string{
		"User": {
			"Publish research paper",
		},
		"Advisor": {
			"Publish research paper",
			"Create group",
		},
		"HOD": {
			"Publish research paper",
			"Research submission approval",
		},
	}

	for roleName, permissions := range rolePermissions {
		roleId, _ := permission.GetRoleId(roleName)

		// Loop through the permissions for this role
		for _, permissionName := range permissions {
			// Get the permission ID from the database
			permissionId := permission.GetPermissionId(permissionName) // replace with your function to get the permission ID
			permission.AssignRolePermission(roleId, permissionId)
		}
	}
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Role Permissions: %v", rolePermissions))
}

// Seed Research Types
func SeedResearchTypes() {
	types := []model.ResearchType{
		{ResearchTypeName: "Capstone"},
		{ResearchTypeName: "Thesis"},
		{ResearchTypeName: "Dissertation"},
		{ResearchTypeName: "Proposal"},
		{ResearchTypeName: "Journal"},
		{ResearchTypeName: "Conference"},
		{ResearchTypeName: "Report"},
		{ResearchTypeName: "Book"},
		{ResearchTypeName: "Article"},
		{ResearchTypeName: "Other"},
	}

	database.Db.CreateInBatches(types, len(types))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Research Types: %v", types))
}

// Seed Departments
func SeedDepartments() {

	departments := []model.Department{
		{DepartmentName: "Computer Science", DepartmentTag: "CS", DepartmentColor: "#0030AA", DepartmentStatus: true},
		{DepartmentName: "Management of Information Systems", DepartmentTag: "MIS", DepartmentColor: "#0077AA", DepartmentStatus: true},
		{DepartmentName: "Center for Professional Education", DepartmentTag: "CPE", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Architectural Engineering", DepartmentTag: "AE", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Architecture", DepartmentTag: "ARC", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Banking and Finance", DepartmentTag: "BAF", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Business Administration", DepartmentTag: "BUS", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Civil Engineering", DepartmentTag: "CE", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Construction Management", DepartmentTag: "CM", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Digital Arts and Design", DepartmentTag: "DAD", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Economics", DepartmentTag: "ECON", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "English Language Teaching", DepartmentTag: "ELT", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Industrial Engineering", DepartmentTag: "IE", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "International Relations", DepartmentTag: "IR", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "International Trade and Logistics", DepartmentTag: "ITL", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Languages", DepartmentTag: "DL", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "Mathematics", DepartmentTag: "DM", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "English Language Preparatory Program", DepartmentTag: "ELP", DepartmentColor: generator.GenerateDarkColor()},
		{DepartmentName: "International Foundation Diploma", DepartmentTag: "IFD", DepartmentColor: generator.GenerateDarkColor()},
	}
	database.Db.CreateInBatches(departments, len(departments))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Departments: %v", departments))
}

// Seed Folders
func SeedFolders() {
	folders := []model.Folder{
		{FolderName: "Final Product"},
		{FolderName: "Draft Document"},
		{FolderName: "Reference"},
		{FolderName: "Data Collection"},
		{FolderName: "Miscellaneous"},
	}
	database.Db.CreateInBatches(folders, len(folders))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Folders: %v", folders))
}

// Seed Collab Permission Category
func SeedCollabPermissionCategories() {
	categories := []model.CollabPermissionCategory{
		{CollabPermissionCategoryName: "Task"},
		{CollabPermissionCategoryName: "File"},
		{CollabPermissionCategoryName: "Schedule"},
	}
	database.Db.CreateInBatches(categories, len(categories))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Collab Permission Categories: %v", categories))
}

// Seed collab permissions
const collabPermCat = "collab_permission_category_name = ?"

func SeedCollabPermissions() {
	var taskCat, fileCat, scheduleCat model.CollabPermissionCategory
	database.Db.Where(collabPermCat, "Task").First(&taskCat)
	database.Db.Where(collabPermCat, "File").First(&fileCat)
	database.Db.Where(collabPermCat, "Schedule").First(&scheduleCat)

	permissions := []model.CollabPermission{
		{CollabPermissionName: "Create task", CollabPermissionCategoryID: taskCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Edit task", CollabPermissionCategoryID: taskCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Delete task", CollabPermissionCategoryID: taskCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Upload file", CollabPermissionCategoryID: fileCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Edit file", CollabPermissionCategoryID: fileCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Delete file", CollabPermissionCategoryID: fileCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Add schedule event", CollabPermissionCategoryID: scheduleCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Edit schedule event", CollabPermissionCategoryID: scheduleCat.CollabPermissionCategoryID},
		{CollabPermissionName: "Delete schedule event", CollabPermissionCategoryID: scheduleCat.CollabPermissionCategoryID},
	}

	// Seed the permissions
	for _, perm := range permissions {
		database.Db.FirstOrCreate(&perm, model.CollabPermission{CollabPermissionName: perm.CollabPermissionName})
	}
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Collab Permissions: %v", permissions))
}

// Seed Task Status
func SeedTaskStatus() {
	statuses := []model.TaskStatus{
		{TaskStatusName: "TO DO", TaskStatusColor: "#B2AFAF"},
		{TaskStatusName: "IN PROGRESS", TaskStatusColor: "#0030AA"},
		{TaskStatusName: "IN REVIEW", TaskStatusColor: "#FFB800"},
		{TaskStatusName: "COMPLETED", TaskStatusColor: "#00A000"},
		{TaskStatusName: "ON HOLD", TaskStatusColor: "#BB0000"},
	}

	// Seed the status
	database.Db.CreateInBatches(statuses, len(statuses))
	seedStatuses = append(seedStatuses, fmt.Sprintf("Seeded Task Status: %v", statuses))
}
