package util

import (
	"encoding/json"
	"os"
	"sync"
)

// CachedResponse matches the frontend's CachedResponse interface
type CachedResponse struct {
	PromptText string `json:"promptText"`
	HasImage   bool   `json:"hasImage"`
	XML        string `json:"xml"`
}

var (
	cachedResponses     []CachedResponse
	cachedResponsesOnce sync.Once
)

// LoadCachedResponses loads cached responses from a JSON file
func LoadCachedResponses(path string) {
	cachedResponsesOnce.Do(func() {
		data, err := os.ReadFile(path)
		if err != nil {
			cachedResponses = []CachedResponse{}
			return
		}
		if err := json.Unmarshal(data, &cachedResponses); err != nil {
			cachedResponses = []CachedResponse{}
		}
	})
}

// FindCachedResponse finds a matching cached response
func FindCachedResponse(promptText string, hasImage bool) *CachedResponse {
	for i := range cachedResponses {
		if cachedResponses[i].PromptText == promptText &&
			cachedResponses[i].HasImage == hasImage &&
			cachedResponses[i].XML != "" {
			return &cachedResponses[i]
		}
	}
	return nil
}
