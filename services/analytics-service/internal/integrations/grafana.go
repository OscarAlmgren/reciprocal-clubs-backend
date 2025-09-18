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

// GrafanaClient manages Grafana integration for dashboard creation and management
type GrafanaClient struct {
	config     *GrafanaConfig
	httpClient *http.Client
	logger     logging.Logger
}

// NewGrafanaClient creates a new Grafana client
func NewGrafanaClient(config *GrafanaConfig, logger logging.Logger) *GrafanaClient {
	return &GrafanaClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ValidateConfig validates Grafana configuration
func (g *GrafanaClient) ValidateConfig() error {
	if g.config.URL == "" {
		return fmt.Errorf("Grafana URL is required")
	}
	if g.config.APIKey == "" {
		return fmt.Errorf("Grafana API key is required")
	}
	return nil
}

// TestConnection tests connectivity to Grafana API
func (g *GrafanaClient) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", g.config.URL+"/api/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to Grafana: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Grafana health check failed: status %d", resp.StatusCode)
	}

	g.logger.Info("Grafana connection test successful", map[string]interface{}{
		"url": g.config.URL,
	})

	return nil
}

// GrafanaDashboard represents a Grafana dashboard
type GrafanaDashboard struct {
	Dashboard interface{} `json:"dashboard"`
	Overwrite bool        `json:"overwrite"`
	Message   string      `json:"message,omitempty"`
}

// CreateDashboard creates or updates a dashboard in Grafana
func (g *GrafanaClient) CreateDashboard(ctx context.Context, dashboardConfig map[string]interface{}) error {
	dashboard := GrafanaDashboard{
		Dashboard: dashboardConfig,
		Overwrite: true,
		Message:   "Created/Updated by analytics-service",
	}

	jsonData, err := json.Marshal(dashboard)
	if err != nil {
		return fmt.Errorf("failed to marshal dashboard: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.config.URL+"/api/dashboards/db", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create dashboard request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create dashboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Grafana dashboard creation failed: status %d", resp.StatusCode)
	}

	g.logger.Info("Dashboard created/updated in Grafana", map[string]interface{}{
		"status": resp.StatusCode,
	})

	return nil
}

// GrafanaDataSource represents a Grafana data source
type GrafanaDataSource struct {
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	URL       string            `json:"url"`
	Access    string            `json:"access"`
	Database  string            `json:"database,omitempty"`
	BasicAuth bool              `json:"basicAuth"`
	JsonData  map[string]interface{} `json:"jsonData,omitempty"`
}

// CreateDataSource creates a data source in Grafana
func (g *GrafanaClient) CreateDataSource(ctx context.Context, dataSource *GrafanaDataSource) error {
	jsonData, err := json.Marshal(dataSource)
	if err != nil {
		return fmt.Errorf("failed to marshal data source: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.config.URL+"/api/datasources", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create data source request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create data source: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Grafana data source creation failed: status %d", resp.StatusCode)
	}

	g.logger.Info("Data source created in Grafana", map[string]interface{}{
		"name": dataSource.Name,
		"type": dataSource.Type,
		"status": resp.StatusCode,
	})

	return nil
}

// GetDashboards retrieves all dashboards from Grafana
func (g *GrafanaClient) GetDashboards(ctx context.Context) ([]map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", g.config.URL+"/api/search?type=dash-db", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboards: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Grafana dashboard retrieval failed: status %d", resp.StatusCode)
	}

	var dashboards []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&dashboards); err != nil {
		return nil, fmt.Errorf("failed to decode dashboards: %w", err)
	}

	g.logger.Info("Retrieved dashboards from Grafana", map[string]interface{}{
		"count": len(dashboards),
	})

	return dashboards, nil
}

// DeleteDashboard deletes a dashboard from Grafana
func (g *GrafanaClient) DeleteDashboard(ctx context.Context, uid string) error {
	req, err := http.NewRequestWithContext(ctx, "DELETE", g.config.URL+"/api/dashboards/uid/"+uid, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete dashboard: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Grafana dashboard deletion failed: status %d", resp.StatusCode)
	}

	g.logger.Info("Dashboard deleted from Grafana", map[string]interface{}{
		"uid": uid,
		"status": resp.StatusCode,
	})

	return nil
}

// CreateAnalyticsDashboard creates a standard analytics dashboard
func (g *GrafanaClient) CreateAnalyticsDashboard(ctx context.Context, clubID string) error {
	dashboard := map[string]interface{}{
		"id":    nil,
		"uid":   fmt.Sprintf("analytics-club-%s", clubID),
		"title": fmt.Sprintf("Analytics Dashboard - Club %s", clubID),
		"tags":  []string{"analytics", "club", clubID},
		"timezone": "browser",
		"panels": []map[string]interface{}{
			{
				"id":    1,
				"title": "Event Count",
				"type":  "stat",
				"targets": []map[string]interface{}{
					{
						"expr":   fmt.Sprintf(`analytics_events_total{club_id="%s"}`, clubID),
						"legendFormat": "Events",
					},
				},
				"gridPos": map[string]interface{}{
					"h": 8,
					"w": 12,
					"x": 0,
					"y": 0,
				},
			},
			{
				"id":    2,
				"title": "Metrics Over Time",
				"type":  "graph",
				"targets": []map[string]interface{}{
					{
						"expr":   fmt.Sprintf(`rate(analytics_events_total{club_id="%s"}[5m])`, clubID),
						"legendFormat": "Event Rate",
					},
				},
				"gridPos": map[string]interface{}{
					"h": 8,
					"w": 12,
					"x": 12,
					"y": 0,
				},
			},
		},
		"time": map[string]interface{}{
			"from": "now-6h",
			"to":   "now",
		},
		"refresh": "30s",
	}

	return g.CreateDashboard(ctx, dashboard)
}

// GetHealth returns Grafana integration health status
func (g *GrafanaClient) GetHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"service": "grafana",
		"status":  "unknown",
		"url":     g.config.URL,
		"timestamp": time.Now(),
	}

	// Test connection
	if err := g.TestConnection(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
	} else {
		health["status"] = "healthy"
	}

	return health
}

// CreateFolder creates a folder in Grafana for organizing dashboards
func (g *GrafanaClient) CreateFolder(ctx context.Context, title, uid string) error {
	folder := map[string]interface{}{
		"uid":   uid,
		"title": title,
	}

	jsonData, err := json.Marshal(folder)
	if err != nil {
		return fmt.Errorf("failed to marshal folder: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.config.URL+"/api/folders", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create folder request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.config.APIKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 && resp.StatusCode != 409 { // 409 means folder already exists
		return fmt.Errorf("Grafana folder creation failed: status %d", resp.StatusCode)
	}

	g.logger.Info("Folder created/verified in Grafana", map[string]interface{}{
		"title": title,
		"uid": uid,
		"status": resp.StatusCode,
	})

	return nil
}