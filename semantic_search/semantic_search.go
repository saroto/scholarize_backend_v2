package semantic_search

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func Search(query string) ([]string, error) {
    host := viper.GetString("semantic_search.host")

	searchRequest := map[string]interface{}{
		"query": query,
	}

	jsonBody, err := json.Marshal(searchRequest)
	
	if err != nil {
	return nil, fmt.Errorf("error marshalling search query: %w", err)
	}

	url := fmt.Sprintf("%s/api/semantic_search", host)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))

	if err != nil {
		return nil, fmt.Errorf("error making search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)

	  if err != nil {
        return nil, fmt.Errorf("error executing search request: %w", err)
    }
    defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		 body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search request failed with status: %s", body)
	}

	var searchResponse struct {
		Results []string `json:"results"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("error decoding search response: %w", err)
	}

	return searchResponse.Results, nil

}

// Update the Search function to return a structured response
func SearchQuery(query string) (map[string]interface{}, error) {
    host := viper.GetString("semantic_search.host")
    urlStr := fmt.Sprintf("%s/api/semantic_search/", host) 
    
    // Create form data
    formData := neturl.Values{}
    formData.Set("query", query)
    
    // Create a new request with form data
    req, err := http.NewRequest("POST", urlStr, strings.NewReader(formData.Encode()))
    if err != nil {
        return nil, fmt.Errorf("error making search request: %w", err)
    }
    
    // Set the correct headers
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.Header.Set("Accept", "application/json")
    
    client := &http.Client{Timeout: 20 * time.Second}
    res, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("error executing search request: %w", err)
    }
    defer res.Body.Close()
    
    // Read the full response body
    body, err := io.ReadAll(res.Body)
    if err != nil {
        return nil, fmt.Errorf("error reading response body: %w", err)
    }
    
    if res.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("search request failed with status %d: %s", res.StatusCode, string(body))
    }
    
    // Decode the JSON response
    var searchResponse map[string]interface{}
    if err := json.Unmarshal(body, &searchResponse); err != nil {
        return nil, fmt.Errorf("error decoding search response: %w", err)
    }
    
    return searchResponse, nil
}