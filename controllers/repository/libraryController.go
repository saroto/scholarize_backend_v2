package repository

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"root/config"
	"root/constant"
	"root/database"
	"root/meilisearch"
	"root/model"
	"root/semantic_search"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var searchResult struct {
    Message        string          `json:"message"`
    Query          string          `json:"query"`
    PapersFound    int             `json:"papers_found"`
    PapersID       string          `json:"papers_id"`  // This is a string that needs conversion
    // ResearchPapers []ResearchPaper `json:"research_papers"`
}

func reverseArray(arr []int) []int {
	sortArr := arr
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		sortArr[i], sortArr[j] = sortArr[j], sortArr[i]
	}
	return sortArr
}

func OrderArray(arr, order []int) []int {
	// Create a map to store the order of each element
	orderMap := make(map[int]int)
	for index, value := range order {
		orderMap[value] = index
	}

	// Sort `arr` based on the order in `orderMap`
	sort.SliceStable(arr, func(i, j int) bool {
		return orderMap[arr[i]] < orderMap[arr[j]]
	})

	return arr
}

// Using Meili to Filter Published Papers
func MeiliSearchQueryFilter(searchTerm string, departmentFilter, researchTypeFilter, publishedYearFilter []string, sortOrder string) ([]int, error) {
	query := database.Db.Model(&model.ResearchPaper{}).
		Select("DISTINCT research_paper.research_paper_id, research_paper.published_at, research_paper.research_title").
		Joins("JOIN research_type ON research_paper.research_type_id = research_type.research_type_id").
		Joins("JOIN researchpaperdepartment ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Joins("JOIN department ON researchpaperdepartment.department_id = department.department_id").
		Where("research_paper.research_paper_status = 'published'")

	// Apply department id filter
	// if len(departmentFilter) > 0 {
	// 	query = query.Where("department.department_id IN (?)", departmentFilter)
	// }

	if len(departmentFilter) > 0 {
		departmentCount := len(departmentFilter)
		query = query.Where("department.department_id IN (?)", departmentFilter).
			Group("research_paper.research_paper_id").
			Having("COUNT(DISTINCT department.department_id) = ?", departmentCount)
	}

	// Apply research type id filter
	if len(researchTypeFilter) > 0 {
		query = query.Where("research_type.research_type_id IN (?)", researchTypeFilter)
	}

	baselineIDs := []int{}

	fmt.Println("Search Term: ", searchTerm)
	if searchTerm != "" {

		var uniqueIDs []int
		// Get the IDs of the research papers that match the search term full text
		reseahPaperIDs, err := meilisearch.MeiliFilterResearchPaperForPaperID(searchTerm)
		if err != nil {
			return nil, err
		}

		if len(reseahPaperIDs) == 0 {
			fmt.Println("No ID from research paper")
			// Get the IDs of the research papers that match the search term full text
			fulltextIDs, err := meilisearch.MeiliFilterFullTextForPaperID(searchTerm)
			if err != nil {
				return nil, err
			}

			if len(fulltextIDs) == 0 {
				fmt.Println("No ID from full text")
				// Get the IDs of the research papers that match the search term clean text
				cleantextIDs, err := meilisearch.MeiliFilterCleanTextForPaperID(searchTerm)
				if err != nil {
					return nil, err
				}

				if len(cleantextIDs) == 0 {
					fmt.Println("No ID from clean text")
				}

				uniqueIDs = cleantextIDs
			} else {
				uniqueIDs = fulltextIDs
			}
		} else {
			uniqueIDs = reseahPaperIDs
		}
		baselineIDs = uniqueIDs
		fmt.Println("ID From Meili: ", uniqueIDs)
		query = query.Where("research_paper.research_paper_id IN (?)", uniqueIDs)
	} else {
		if len(publishedYearFilter) == 0 {
			query = query.Order("research_paper.research_title " + sortOrder)
		}
	}

	// if searchTerm != "" {

	// 	var uniqueIDs []int
	// 	// Get the IDs of the research papers that match the search term full text
	// 	reseahPaperIDs, err := meilisearch.MeiliFilterResearchPaperForPaperID(searchTerm)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if len(reseahPaperIDs) == 0 {
	// 		fmt.Println("No ID from research paper")
	// 		// Get the IDs of the research papers that match the search term clean text
	// 		cleantextIDs, err := meilisearch.MeiliFilterCleanTextForPaperID(searchTerm)
	// 		if err != nil {
	// 			return nil, err
	// 		}

	// 		if len(cleantextIDs) == 0 {
	// 			fmt.Println("No ID from clean text")
	// 			// Get the IDs of the research papers that match the search term full text
	// 			fulltextIDs, err := meilisearch.MeiliFilterFullTextForPaperID(searchTerm)
	// 			if err != nil {
	// 				return nil, err
	// 			}

	// 			if len(fulltextIDs) == 0 {
	// 				fmt.Println("No ID from full text")
	// 			}
	// 			uniqueIDs = fulltextIDs

	// 		} else {
	// 			uniqueIDs = cleantextIDs
	// 		}
	// 	} else {
	// 		uniqueIDs = reseahPaperIDs
	// 	}
	// 	baselineIDs = uniqueIDs
	// 	fmt.Println("ID From Meili: ", uniqueIDs)
	// 	query = query.Where("research_paper.research_paper_id IN (?)", uniqueIDs)
	// } else {
	// 	if len(publishedYearFilter) == 0 {
	// 		query = query.Order("research_paper.research_title " + sortOrder)
	// 	}
	// }

	// Apply published year filter
	if len(publishedYearFilter) > 0 {
		query = query.Where("EXTRACT(YEAR FROM research_paper.published_at) IN (?)", publishedYearFilter).
			Order(fmt.Sprintf("research_paper.published_at %s", sortOrder))
	}

	var results []model.ResearchPaper
	// Select only the IDs of the research papers
	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Extract only the IDs from the results
	var paperIds []int
	for _, result := range results {
		paperIds = append(paperIds, result.ResearchPaperID)
	}

	// Sort paper by year if year filter is not empty
	if len(publishedYearFilter) > 0 {
		// Sort the paper by year
		return paperIds, nil
	}

	if len(baselineIDs) > 0 {
		paperIds = OrderArray(paperIds, baselineIDs)
	}

	return paperIds, nil
}


func SemanticSearchQueryFilter(searchTerm string, departmentFilter, researchTypeFilter, publishedYearFilter []string, sortOrder string) ([]int, error) {
	query := database.Db.Model(&model.ResearchPaper{}).
		Select("DISTINCT research_paper.research_paper_id, research_paper.published_at, research_paper.research_title").
		Joins("JOIN research_type ON research_paper.research_type_id = research_type.research_type_id").
		Joins("JOIN researchpaperdepartment ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Joins("JOIN department ON researchpaperdepartment.department_id = department.department_id").
		Where("research_paper.research_paper_status = 'published'")


	fmt.Println("Search Term: ", query)
	// Apply department id filter
	// if len(departmentFilter) > 0 {
	// 	query = query.Where("department.department_id IN (?)", departmentFilter)
	// }

	if len(departmentFilter) > 0 {
		departmentCount := len(departmentFilter)
		query = query.Where("department.department_id IN (?)", departmentFilter).
			Group("research_paper.research_paper_id").
			Having("COUNT(DISTINCT department.department_id) = ?", departmentCount)
	}

	// Apply research type id filter
	if len(researchTypeFilter) > 0 {
		query = query.Where("research_type.research_type_id IN (?)", researchTypeFilter)
	}

	baselineIDs := []int{}

	if searchTerm != "" {

		var uniqueIDs []int

		searchResult, err := semantic_search.SearchQuery(searchTerm)

		fmt.Println("Search Result: ", searchResult)
		if err != nil {
			return nil, err
		}

		var paperIDs []int
				// Check if papers_id was provided directly
		if papersIDInterface, ok := searchResult["papers_id"]; ok {
			// If papers_id is an array directly
			if papersIDArray, ok := papersIDInterface.([]interface{}); ok {
				for _, idInterface := range papersIDArray {
					if idFloat, ok := idInterface.(float64); ok {
						paperIDs = append(paperIDs, int(idFloat))
					}
				}
			} else if papersIDString, ok := papersIDInterface.(string); ok {
				// If papers_id is a string that needs parsing
				idStrings := strings.Split(papersIDString, ",")
				for _, idStr := range idStrings {
					idStr = strings.TrimSpace(idStr)
					id, err := strconv.Atoi(idStr)
					if err != nil {
						continue // Skip invalid IDs
					}
					paperIDs = append(paperIDs, id)
				}
			}
		}

		uniqueIDs = paperIDs
		
		fmt.Println("ID From semantic: ", uniqueIDs)

		baselineIDs = uniqueIDs
		query = query.Where("research_paper.research_paper_id IN (?)", uniqueIDs)
	} else {
		if len(publishedYearFilter) == 0 {
			query = query.Order("research_paper.research_title " + sortOrder)
		}
	}

	// Apply published year filter
	if len(publishedYearFilter) > 0 {
		query = query.Where("EXTRACT(YEAR FROM research_paper.published_at) IN (?)", publishedYearFilter).
			Order(fmt.Sprintf("research_paper.published_at %s", sortOrder))
	}

	var results []model.ResearchPaper
	// Select only the IDs of the research papers
	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Extract only the IDs from the results
	var paperIds []int
	for _, result := range results {
		paperIds = append(paperIds, result.ResearchPaperID)
	}

	// Sort paper by year if year filter is not empty
	if len(publishedYearFilter) > 0 {
		// Sort the paper by year
		return paperIds, nil
	}

	if len(baselineIDs) > 0 {
		paperIds = OrderArray(paperIds, baselineIDs)
	}

	return paperIds, nil
}

// Backup
// Query to search for research papers with based on filters for PUBLISHED papers only
func ResearchPaperSearchQuery(searchTerm string, departmentFilter, researchTypeFilter, publishedYearFilter []string, sortOrder string) ([]int, error) {
	var results []struct {
		ResearchPaperID int       `gorm:"column:research_paper_id"`
		PublishedAt     time.Time `gorm:"column:published_at"`
	}
	query := database.Db.Model(&model.ResearchPaper{}).
		Select("DISTINCT research_paper.research_paper_id, research_paper.published_at").
		Joins("JOIN research_type ON research_paper.research_type_id = research_type.research_type_id").
		Joins("JOIN researchpaperdepartment ON research_paper.research_paper_id = researchpaperdepartment.research_paper_id").
		Joins("JOIN department ON researchpaperdepartment.department_id = department.department_id").
		Joins("JOIN cleantext ON research_paper.cleantext_id = cleantext.cleantext_id").
		Where("research_paper.research_paper_status = 'published'").
		Group("research_paper.research_paper_id, research_paper.published_at")

	// Apply search term filter
	if searchTerm != "" {
		// Clean search term at laravel
		cleanUrl := config.GetFileServiceURL("cleantext")
		cleanedText, err := CleanTextAtLaravel(searchTerm, cleanUrl)
		if err == nil {
			searchTerm = strings.ToLower(cleanedText)
		} else {
			searchTerm = strings.ToLower(searchTerm)
		}

		searchTermPattern := "%" + searchTerm + "%"

		//query = query.Where("LOWER(research_paper.research_title) LIKE ? OR LOWER(research_paper.abstract) LIKE ? OR LOWER(research_paper.tag) LIKE ? OR LOWER(research_paper.author) LIKE ? OR LOWER(research_paper.advisor) LIKE ? OR LOWER(cleantext.cleantext_content) LIKE ?", searchTermPattern, searchTermPattern, searchTermPattern, searchTermPattern, searchTermPattern, searchTermPattern)
		query = query.Where(
			"LOWER(research_paper.research_title) LIKE ? OR "+
				"LOWER(research_paper.abstract) LIKE ? OR "+
				"LOWER(research_paper.tag) LIKE ? OR "+
				"LOWER(research_paper.author) LIKE ? OR "+
				"LOWER(research_paper.advisor) LIKE ? OR "+
				"LOWER(cleantext.cleantext_content) LIKE ?",
			searchTermPattern, searchTermPattern, searchTermPattern, searchTermPattern, searchTermPattern, searchTermPattern,
		)
	}

	// Apply department id filter
	if len(departmentFilter) > 0 {
		query = query.Where("department.department_id IN (?)", departmentFilter)
	}

	// Apply research type id filter
	if len(researchTypeFilter) > 0 {
		query = query.Where("research_type.research_type_id IN (?)", researchTypeFilter)
	}

	// Apply published year filter
	if len(publishedYearFilter) > 0 {
		query = query.Where("EXTRACT(YEAR FROM research_paper.published_at) IN (?)", publishedYearFilter)
	}

	query = query.Group("research_paper.research_paper_id, research_paper.published_at").
		Order("research_paper.published_at " + sortOrder)

	// Select only the IDs of the research papers
	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	// Extract only the IDs from the results
	var paperIds []int
	for _, result := range results {
		paperIds = append(paperIds, result.ResearchPaperID)
	}

	return paperIds, nil
}

// Get each block of research paper info when query for papers
func GetEachResearchPaperInfoOnQuery(paperId int) (constant.QueryResearchPaperInfo, error) {
	// Get the researh paper info
	var paper model.ResearchPaper
	result := database.Db.First(&paper, paperId)
	if result.Error != nil {
		return constant.QueryResearchPaperInfo{}, result.Error
	}
	if result.RowsAffected == 0 {
		return constant.QueryResearchPaperInfo{}, nil
	}

	// Research Title, only one title
	researchTitle := paper.ResearchTitle

	// Get Departments, can be multiple
	var departments []model.Department
	res := database.Db.Model(&departments).
		Joins("JOIN researchpaperdepartment ON researchpaperdepartment.department_id = department.department_id").
		Where("researchpaperdepartment.research_paper_id = ?", paperId).
		Find(&departments)
	if res.Error != nil {
		return constant.QueryResearchPaperInfo{}, res.Error
	}

	// Struct the departments info
	var departmentInfos []constant.DepartmentInfo
	for _, dept := range departments {
		departmentInfo := constant.DepartmentInfo{
			DepartmentID:     dept.DepartmentID,
			DepartmentName:   dept.DepartmentName,
			DepartmentTag:    dept.DepartmentTag,
			DepartmentColor:  dept.DepartmentColor,
			DepartmentStatus: dept.DepartmentStatus,
		}
		departmentInfos = append(departmentInfos, departmentInfo)
	}

	// Get the research type only a type
	var researchType model.ResearchType
	researchTypeInfo := database.Db.First(&researchType, paper.ResearchTypeID)
	if researchTypeInfo.Error != nil {
		return constant.QueryResearchPaperInfo{}, researchTypeInfo.Error
	}

	// Get the author
	author := paper.Author

	// Get the published date
	publishedAt := paper.PublishedAt

	// Get the abstract
	abstract := paper.Abstract

	// Return the research paper info
	return constant.QueryResearchPaperInfo{
		ResearchPaperID: paperId,
		PublicID:        paper.PublicID,
		Title:           researchTitle,
		Departments:     departmentInfos,
		PDFPath:         paper.PDFPath,
		Type:            researchType,
		Author:          author,
		PublishedAt:     publishedAt.Format("2006-01-02 15:04:05"),
		Abstract:        abstract,
		UserID:          paper.UserID,
	}, nil
}

// Handler for searching research papers with filters
func HandleResearchPaperSearch(c *gin.Context) {
	// Extract filters from the POST form
	searchTerm := c.Query("search")
	departmentFilters := c.QueryArray("department")
	researchTypeFilters := c.QueryArray("research_type")
	publishedYearFilters := c.QueryArray("published_year")
	sortOrder := c.DefaultQuery("sort", "ASC")

	// // Fetch the IDs of research papers that match the filters
	// paperIds, err := ResearchPaperSearchQuery(searchTerm, departmentFilters, researchTypeFilters, publishedYearFilters, sortOrder)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching paper IDs"})
	// 	return
	// }

	// Fetch the IDs of research papers that match the filters
	paperIds, err := MeiliSearchQueryFilter(searchTerm, departmentFilters, researchTypeFilters, publishedYearFilters, sortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching paper IDs"})
		return
	}

	// Fetch detailed information for each paper
	var papersInfo []constant.QueryResearchPaperInfo
	fmt.Println("Paper IDs: ", paperIds)
	for _, paperId := range paperIds {
		paperInfo, err := GetEachResearchPaperInfoOnQuery(paperId)
		if err != nil {
			log.Printf("Error fetching info for paper ID %d: %v", paperId, err)
			continue
		}
		papersInfo = append(papersInfo, paperInfo)
	}

	// Respond with the fetched paper information
	c.JSON(http.StatusOK, gin.H{
		"papers": papersInfo,
	})
}

func HandleResearchPaperSemanticSearch(c *gin.Context) {
	searchTerm := c.Query("search")
	departmentFilters := c.QueryArray("department")
	researchTypeFilters := c.QueryArray("research_type")
	publishedYearFilters := c.QueryArray("published_year")
	sortOrder := c.DefaultQuery("sort", "ASC")

	paperIds, err := SemanticSearchQueryFilter(searchTerm, departmentFilters, researchTypeFilters, publishedYearFilters, sortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching paper IDs"})
		return
	}

	var papersInfo []constant.QueryResearchPaperInfo
	for _, paperId := range paperIds {
		paperInfo, err := GetEachResearchPaperInfoOnQuery(paperId)
		if err != nil {
			log.Printf("Error fetching info for paper ID %d: %v", paperId, err)
			continue
		}
		papersInfo = append(papersInfo, paperInfo)
	}
	c.JSON(http.StatusOK, gin.H{
		"papers": papersInfo,
	})
}

// TODO: CHANGED TO PUBLIC ID
// Get individual research paper data
func HandleGetIndividualPaperPage(c *gin.Context) {
	// Get the paper ID from the URL
	idStr := c.Param("id")

	if idStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No ID found in the request",
		})
		return
	}

	// Create a variable to hold the research paper
	var paper model.ResearchPaper
	var fulltext model.Fulltext

	// Find the paper by public_id
	result := database.Db.Where("public_id = ?", idStr).First(&paper)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error fetching research paper",
		})
		return
	}

	// Check if the paper was found
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Research paper not found",
		})
		return
	}

	id := paper.ResearchPaperID

	// Check if the paper is published
	if paper.ResearchPaperStatus != "published" {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Forbidden: Paper is not published",
		})
		return
	}

	// Find the fulltext with the given ID
	result = database.Db.First(&fulltext, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error fetching fulltext",
		})
		return
	}

	// Check if the fulltext was found
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Fulltext not found",
		})
		return
	}

	// Return the full text content if everything is ok
	content := fulltext.FulltextContent

	// Get information about the departments that the paper belongs to
	paperInfo, _ := GetEachResearchPaperInfoOnQuery(paper.ResearchPaperID)

	// Return the paper if everything is ok
	c.JSON(http.StatusOK, gin.H{
		"message":          "Research paper retrieved successfully!",
		"paper_info":       paperInfo,
		"raw_paper":        paper,
		"fulltext_content": content,
	})
}

// Handle Download File
func HandleDownloadResearchPaper(c *gin.Context) {
	// Get the file path from the query parameter
	paperPath := c.Query("paper_path")
	if paperPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "No file found in the request"})
		return
	}

	// Get the file from the Laravel backend
	downloadURL := config.GetFileServiceURL("repo_download") + paperPath

	// Make an HTTP GET request to the Laravel server to download the file
	resp, err := http.Get(downloadURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error downloading file from backend"})
		return
	}
	defer resp.Body.Close()

	// Check response status code from Laravel
	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Backend server error"})
		return
	}

	// Set the Content-Type header to the appropriate type (could also set it based on file extension)
	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))

	// Optional: Set the Content-Disposition header to prompt a download
	c.Writer.Header().Set("Content-Disposition", "attachment; filename="+paperPath)

	// Stream the file content directly to the client
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Error streaming file to client"})
		return
	}

	// Return the response
	c.Status(http.StatusOK)
}
