package meilisearch

import (
	"fmt"

	"github.com/meilisearch/meilisearch-go"
	"github.com/spf13/viper"
)

var client *meilisearch.Client
var researchPaperIndexes, fulltextIndexes, cleanedTextIndexes map[string]*meilisearch.Index

func configureIndex(index *meilisearch.Index, searchableAttributes []string, rankingRules []string) {
	_, err := index.UpdateSearchableAttributes(&searchableAttributes)
	if err != nil {
		panic(err)
	}

	_, err = index.UpdateRankingRules(&rankingRules)
	if err != nil {
		panic(err)
	}
}

func InitMeiliSearch() {
	host := viper.GetString("meilisearch.host")
	apiKey := viper.GetString("meilisearch.api_key")
	researchPaperIndex := viper.GetString("meilisearch.index.research_paper")
	fulltextIndex := viper.GetString("meilisearch.index.fulltext")
	cleanedTextIndex := viper.GetString("meilisearch.index.cleantext")
	fmt.Println("Research Paper Index:", researchPaperIndex)
	fmt.Println("Fulltext Index:", fulltextIndex)
	fmt.Println("Cleaned Text Index:", cleanedTextIndex)

	client = meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   host,
		APIKey: apiKey,
	})

	researchPaperIndexes = map[string]*meilisearch.Index{
		researchPaperIndex: client.Index(researchPaperIndex),
	}
	fulltextIndexes = map[string]*meilisearch.Index{
		fulltextIndex: client.Index(fulltextIndex),
	}
	cleanedTextIndexes = map[string]*meilisearch.Index{
		cleanedTextIndex: client.Index(cleanedTextIndex),
	}

	configureIndex(researchPaperIndexes[researchPaperIndex], []string{
		"research_title",
		"advisor",
		"author",
		"tag",
		"abstract",
	}, []string{
		"exactness",
		"attribute",
		"typo",
		"proximity",
		"words",
	})

	configureIndex(fulltextIndexes[fulltextIndex], []string{
		"fulltext_content",
	}, []string{
		"words",
		"typo",
		"exactness",
		"proximity",
	})

	configureIndex(cleanedTextIndexes[cleanedTextIndex], []string{
		"cleantext_content",
	}, []string{
		"words",
		"typo",
		"proximity",
		"exactness",
		"sort",
	})
}

type SearchResult struct {
	ResearchPaperID int
	FullTextID      int
}

func MeiliFilterResearchPaperForPaperID(query string) ([]int, error) {
	var ids []int
	seen := make(map[int]struct{})

	for _, index := range researchPaperIndexes {
		searchRes, err := index.Search(query, &meilisearch.SearchRequest{
			Query: query,
		})
		if err != nil {
			return nil, err
		}

		for _, hit := range searchRes.Hits {
			researchPaperID, paperIDOk := hit.(map[string]interface{})["research_paper_id"].(float64)
			if paperIDOk {
				id := int(researchPaperID)
				if _, found := seen[id]; !found {
					ids = append(ids, id)
					seen[id] = struct{}{}
				}
			}
		}
	}

	return ids, nil
}

func MeiliFilterFullTextForPaperID(query string) ([]int, error) {
	var ids []int
	seen := make(map[int]struct{})

	for _, index := range fulltextIndexes {
		searchRes, err := index.Search(query, &meilisearch.SearchRequest{
			Query: query,
		})
		if err != nil {
			return nil, err
		}

		for _, hit := range searchRes.Hits {
			fullTextID, paperIDOk := hit.(map[string]interface{})["fulltext_id"].(float64)
			if paperIDOk {
				id := int(fullTextID)
				if _, found := seen[id]; !found {
					ids = append(ids, id)
					seen[id] = struct{}{}
				}
			}
		}
	}

	return ids, nil
}

func MeiliFilterCleanTextForPaperID(query string) ([]int, error) {
	var ids []int
	seen := make(map[int]struct{})

	for _, index := range cleanedTextIndexes {
		searchRes, err := index.Search(query, &meilisearch.SearchRequest{
			Query: query,
		})
		if err != nil {
			return nil, err
		}

		for _, hit := range searchRes.Hits {
			cleanTextID, paperIDOk := hit.(map[string]interface{})["cleantext_id"].(float64)
			if paperIDOk {
				id := int(cleanTextID)
				if _, found := seen[id]; !found {
					ids = append(ids, id)
					seen[id] = struct{}{}
				}
			}
		}
	}

	return ids, nil
}
