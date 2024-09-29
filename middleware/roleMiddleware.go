package middleware

import (
	"fmt"
	"net/http"
	"root/constant"
	"root/database"
	"root/model"

	"github.com/gin-gonic/gin"
)

// userRole, userID is set in JwtMiddleware

// Admin or Super Admin role
func AdminOrSuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := getRole(c)

		if role != constant.SuperAdminRole {
			if role != constant.AdminRole {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Unauthorized, you are not an admin or super admin"})
				return
			}
		}

		// User is an admin or super admin
		c.Next()
	}
}

// HOD Role
func HodMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := getRole(c)

		// If user not HOD
		if role != constant.HODRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You are not a HOD"})
			return
		}
		// Get the department of the HOD
		department := getHODdepartment(c)

		// If department is empty
		if department == (model.Department{}) {
			fmt.Printf("User with ID %d is HOD but not assigned to any department! Returning nothing!\n", c.GetInt("userID"))
			c.AbortWithStatusJSON(http.StatusOK, gin.H{
				"message":               "HOD not assigned to any department!",
				"department":            nil,
				"papers":                nil,
				"total_research_papers": 0,
				"total_awaiting":        0,
				"total_rejected":        0,
			})
			return
		}

		// Set departmentID and departmentName in context
		c.Set("departmentID", int(department.DepartmentID))
		c.Set("departmentName", department.DepartmentName)

		fmt.Print("Department ID: ", department.DepartmentID, " Department Name: ", department.DepartmentName, "\n")

		// User is HOD proceed to the next middleware or handler
		c.Next()
	}
}

// Advisor Role
func AdvisorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := getRole(c)

		// If user not Advisor
		if role != constant.AdvisorRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You are not an advisor"})
			return
		}

		// User is Advisor proceed to the next middleware or handler
		c.Next()
	}
}

// SuperAdmin Role
func SuperAdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := getRole(c)

		// If user not SuperAdmin
		if role != constant.SuperAdminRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "You are not a super admin"})
			return
		}

		// User is SuperAdmin proceed to the next middleware or handler
		c.Next()
	}
}

// Get user role function
func getRole(c *gin.Context) string {
	userRole, exists := c.Get("userRole")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized user role not found in context"})
		return ""
	}

	role, ok := userRole.(string)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized user role not a string"})
		return ""
	}

	return role
}

// Get department of HOD
func getHODdepartment(c *gin.Context) model.Department {
	// Get the userID from context
	hodId, exists := c.Get("userID")
	if !exists {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return model.Department{}
	}

	// department head record
	var depHead model.DepartmentHead
	database.Db.Where("user_id = ?", hodId).First(&depHead)

	// department record
	var department model.Department
	database.Db.Where("department_id = ?", depHead.DepartmentID).First(&department)

	// return department record
	return department
}
