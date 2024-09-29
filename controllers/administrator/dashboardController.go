package administrator

import (
	"fmt"
	"net/http"
	"root/constant"
	"root/database"
	"root/model"
	"root/permission"

	"github.com/gin-gonic/gin"
)

const checkRoleId = "role_id = ?"
const checkArchiveStatus = "collab_archive_status = ?"

// HandleDashboardData returns the data for the dashboard
func HandleGetDashboardData(c *gin.Context) {
	dashboardData := getDashboardData()
	c.JSON(http.StatusOK, dashboardData)
}

// GetDashboardData returns the data for the dashboard
func getDashboardData() constant.DashboardData {
	return constant.DashboardData{
		TotalRegisteredUserCount:   getTotalRegisteredUserCount(),
		TotalAdvisorCount:          getTotalAdvisorCount(),
		TotalHodCount:              getTotalHodCount(),
		TotalUserCount:             getTotalUserCount(),
		TotalDepartmentCount:       getTotalDepartmentCount(),
		TotalActiveDepartmentCount: getTotalActiveDepartmentCount(),
		TotalPaperCount:            getTotalPaperCount(),
		TotalPublishedPaperCount:   getTotalPublishedPaperCount(),
		TotalGroupCount:            getTotalGroupCount(),
		TotalActiveGroupCount:      getTotalActiveGroupCount(),
		TotalArchivedGroupCount:    getTotalArchivedGroupCount(),
		PaperCountByDepartment:     getPublishedPaperCountByDepartment(),
	}
}

// User counts
func getTotalRegisteredUserCount() int64 {
	var total int64
	database.Db.Model(&model.ScholarizeUser{}).
		Where("user_id != ?", 0).
		Count(&total)
	return total
}

func getTotalAdvisorCount() int64 {
	var total int64
	advisorRoleId, _ := permission.GetRoleId(constant.AdvisorRole)
	database.Db.Model(&model.UserRole{}).Where(checkRoleId, advisorRoleId).
		Where("user_id != ?", 0).
		Count(&total)
	return total
}

func getTotalHodCount() int64 {
	var total int64
	hodRoleId, _ := permission.GetRoleId(constant.HODRole)
	database.Db.Model(&model.UserRole{}).Where(checkRoleId, hodRoleId).
		Where("user_id != ?", 0).
		Count(&total)
	return total
}

func getTotalUserCount() int64 {
	var total int64
	userRoleId, _ := permission.GetRoleId(constant.UserRole)
	database.Db.Model(&model.UserRole{}).Where(checkRoleId, userRoleId).
		Where("user_id != ?", 0).
		Count(&total)
	return total
}

// Department counts
func getTotalDepartmentCount() int64 {
	var total int64
	database.Db.Model(&model.Department{}).Count(&total)
	return total
}

func getTotalActiveDepartmentCount() int64 {
	var total int64
	database.Db.Model(&model.Department{}).Where("department_status", true).Count(&total)
	return total
}

// Paper counts
func getTotalPaperCount() int64 {
	var total int64
	database.Db.Model(&model.ResearchPaper{}).Count(&total)
	return total
}

func getTotalPublishedPaperCount() int64 {
	var total int64
	database.Db.Model(&model.ResearchPaper{}).Where("research_paper_status", "published").Count(&total)
	return total
}

func getPublishedPaperCountByDepartment() []constant.DepartmentPaperCount {
	var results []constant.DepartmentPaperCount

	err := database.Db.Model(&model.ResearchPaperDepartment{}).
		Select("department.department_id, department.department_name, department.department_tag, COUNT(researchpaperdepartment.research_paper_id) AS paper_count").
		Joins("JOIN department ON department.department_id = researchpaperdepartment.department_id").
		Joins("JOIN research_paper ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Where("research_paper.research_paper_status = ?", "published").
		Group("department.department_id, department.department_name, department.department_tag").
		Having("COUNT(researchpaperdepartment.research_paper_id) > 0").
		Find(&results).Error

	if err != nil {
		fmt.Println("Error fetching published paper count by department:", err)
		return nil
	}
	return results
}

// Group counts
func getTotalActiveGroupCount() int64 {
	var total int64
	database.Db.Model(&model.Collab{}).Where(checkArchiveStatus, false).Count(&total)
	return total
}

func getTotalArchivedGroupCount() int64 {
	var total int64
	database.Db.Model(&model.Collab{}).Where(checkArchiveStatus, true).Count(&total)
	return total
}

func getTotalGroupCount() int64 {
	var total int64
	database.Db.Model(&model.Collab{}).Count(&total)
	return total
}
