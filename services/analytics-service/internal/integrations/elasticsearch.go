package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// ElasticSearchClient manages ElasticSearch integration
type ElasticSearchClient struct {
	config     *ElasticSearchConfig
	httpClient *http.Client
	logger     logging.Logger
}

// NewElasticSearchClient creates a new ElasticSearch client
func NewElasticSearchClient(config *ElasticSearchConfig, logger logging.Logger) *ElasticSearchClient {
	return &ElasticSearchClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ValidateConfig validates ElasticSearch configuration
func (es *ElasticSearchClient) ValidateConfig() error {
	if es.config.URL == "" {
		return fmt.Errorf("ElasticSearch URL is required")
	}
	if es.config.Index == "" {
		return fmt.Errorf("ElasticSearch index is required")
	}
	return nil
}

// TestConnection tests connectivity to ElasticSearch
func (es *ElasticSearchClient) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", es.config.URL+"/_cluster/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if es.config.Username != "" && es.config.Password != "" {
		req.SetBasicAuth(es.config.Username, es.config.Password)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to ElasticSearch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ElasticSearch health check failed: status %d", resp.StatusCode)
	}

	es.logger.Info("ElasticSearch connection test successful", map[string]interface{}{
		"url": es.config.URL,
		"index": es.config.Index,
	})

	return nil
}

// IndexData indexes analytics data in ElasticSearch
func (es *ElasticSearchClient) IndexData(ctx context.Context, data interface{}) error {
	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create index request
	indexURL := fmt.Sprintf("%s/%s/_doc", es.config.URL, es.config.Index)
	req, err := http.NewRequestWithContext(ctx, "POST", indexURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create index request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if es.config.Username != "" && es.config.Password != "" {
		req.SetBasicAuth(es.config.Username, es.config.Password)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to index data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ElasticSearch indexing failed: status %d", resp.StatusCode)
	}

	es.logger.Info("Data indexed in ElasticSearch", map[string]interface{}{
		"index": es.config.Index,
		"status": resp.StatusCode,
	})

	return nil
}

// SearchData searches for data in ElasticSearch
func (es *ElasticSearchClient) SearchData(ctx context.Context, query map[string]interface{}) (map[string]interface{}, error) {
	// Convert query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query: %w", err)
	}

	// Create search request
	searchURL := fmt.Sprintf("%s/%s/_search", es.config.URL, es.config.Index)
	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(queryJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if es.config.Username != "" && es.config.Password != "" {
		req.SetBasicAuth(es.config.Username, es.config.Password)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ElasticSearch search failed: status %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search results: %w", err)
	}

	es.logger.Info("ElasticSearch search completed", map[string]interface{}{
		"index": es.config.Index,
		"hits": result["hits"],
	})

	return result, nil
}

// CreateIndex creates an index with mapping in ElasticSearch
func (es *ElasticSearchClient) CreateIndex(ctx context.Context, mapping map[string]interface{}) error {
	// Convert mapping to JSON
	mappingJSON, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// Create index request
	indexURL := fmt.Sprintf("%s/%s", es.config.URL, es.config.Index)
	req, err := http.NewRequestWithContext(ctx, "PUT", indexURL, bytes.NewBuffer(mappingJSON))
	if err != nil {
		return fmt.Errorf("failed to create index request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if es.config.Username != "" && es.config.Password != "" {
		req.SetBasicAuth(es.config.Username, es.config.Password)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 400 { // 400 might mean index already exists
		return fmt.Errorf("ElasticSearch index creation failed: status %d", resp.StatusCode)
	}

	es.logger.Info("ElasticSearch index created/verified", map[string]interface{}{
		"index": es.config.Index,
		"status": resp.StatusCode,
	})

	return nil
}

// GetHealth returns ElasticSearch health status
func (es *ElasticSearchClient) GetHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"service": "elasticsearch",
		"status":  "unknown",
		"url":     es.config.URL,
		"index":   es.config.Index,
		"timestamp": time.Now(),
	}

	// Test connection
	if err := es.TestConnection(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
	} else {
		health["status"] = "healthy"
	}

	return health
}

// BulkIndex performs bulk indexing of multiple documents
func (es *ElasticSearchClient) BulkIndex(ctx context.Context, documents []interface{}) error {
	if len(documents) == 0 {
		return nil
	}

	var bulkBody bytes.Buffer

	for _, doc := range documents {
		// Add index action
		action := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": es.config.Index,
			},
		}
		actionJSON, _ := json.Marshal(action)
		bulkBody.Write(actionJSON)
		bulkBody.WriteByte('\n')

		// Add document
		docJSON, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %w", err)
		}
		bulkBody.Write(docJSON)
		bulkBody.WriteByte('\n')
	}

	// Create bulk request
	bulkURL := fmt.Sprintf("%s/_bulk", es.config.URL)
	req, err := http.NewRequestWithContext(ctx, "POST", bulkURL, &bulkBody)
	if err != nil {
		return fmt.Errorf("failed to create bulk request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")
	if es.config.Username != "" && es.config.Password != "" {
		req.SetBasicAuth(es.config.Username, es.config.Password)
	}

	resp, err := es.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute bulk request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("ElasticSearch bulk indexing failed: status %d", resp.StatusCode)
	}

	es.logger.Info("Bulk indexing completed", map[string]interface{}{
		"index": es.config.Index,
		"document_count": len(documents),
		"status": resp.StatusCode,
	})

	return nil
}