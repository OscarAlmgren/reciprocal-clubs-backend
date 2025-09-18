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

// DataDogClient manages DataDog integration for metrics and monitoring
type DataDogClient struct {
	config     *DataDogConfig
	httpClient *http.Client
	logger     logging.Logger
	baseURL    string
}

// NewDataDogClient creates a new DataDog client
func NewDataDogClient(config *DataDogConfig, logger logging.Logger) *DataDogClient {
	baseURL := "https://api.datadoghq.com"
	if config.Site != "" {
		baseURL = fmt.Sprintf("https://api.%s", config.Site)
	}

	return &DataDogClient{
		config:  config,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ValidateConfig validates DataDog configuration
func (dd *DataDogClient) ValidateConfig() error {
	if dd.config.APIKey == "" {
		return fmt.Errorf("DataDog API key is required")
	}
	if dd.config.AppKey == "" {
		return fmt.Errorf("DataDog App key is required")
	}
	return nil
}

// TestConnection tests connectivity to DataDog API
func (dd *DataDogClient) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", dd.baseURL+"/api/v1/validate", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("DD-API-KEY", dd.config.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", dd.config.AppKey)

	resp, err := dd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to DataDog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("DataDog validation failed: status %d", resp.StatusCode)
	}

	dd.logger.Info("DataDog connection test successful", map[string]interface{}{
		"base_url": dd.baseURL,
	})

	return nil
}

// DataDogMetric represents a DataDog metric
type DataDogMetric struct {
	Metric string      `json:"metric"`
	Points [][]float64 `json:"points"`
	Type   string      `json:"type,omitempty"`
	Host   string      `json:"host,omitempty"`
	Tags   []string    `json:"tags,omitempty"`
}

// DataDogMetricsPayload represents the payload for sending metrics
type DataDogMetricsPayload struct {
	Series []DataDogMetric `json:"series"`
}

// SendMetrics sends metrics to DataDog
func (dd *DataDogClient) SendMetrics(ctx context.Context, metrics map[string]interface{}) error {
	var series []DataDogMetric

	timestamp := float64(time.Now().Unix())

	// Convert generic metrics to DataDog format
	for name, value := range metrics {
		if floatVal, ok := value.(float64); ok {
			metric := DataDogMetric{
				Metric: dd.addNamespace(name),
				Points: [][]float64{{timestamp, floatVal}},
				Type:   "gauge",
				Tags:   []string{"source:analytics-service"},
			}
			series = append(series, metric)
		}
	}

	payload := DataDogMetricsPayload{Series: series}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dd.baseURL+"/api/v1/series", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create metrics request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", dd.config.APIKey)

	resp, err := dd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("DataDog metrics submission failed: status %d", resp.StatusCode)
	}

	dd.logger.Info("Metrics sent to DataDog", map[string]interface{}{
		"metric_count": len(series),
		"status": resp.StatusCode,
	})

	return nil
}

// DataDogEvent represents a DataDog event
type DataDogEvent struct {
	Title      string   `json:"title"`
	Text       string   `json:"text"`
	DateHappened int64  `json:"date_happened,omitempty"`
	Priority   string   `json:"priority,omitempty"`
	Host       string   `json:"host,omitempty"`
	Tags       []string `json:"tags,omitempty"`
	AlertType  string   `json:"alert_type,omitempty"`
	SourceTypeName string `json:"source_type_name,omitempty"`
}

// SendEvent sends an event to DataDog
func (dd *DataDogClient) SendEvent(ctx context.Context, event *DataDogEvent) error {
	if event.DateHappened == 0 {
		event.DateHappened = time.Now().Unix()
	}

	if event.SourceTypeName == "" {
		event.SourceTypeName = "analytics-service"
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dd.baseURL+"/api/v1/events", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create event request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", dd.config.APIKey)

	resp, err := dd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("DataDog event submission failed: status %d", resp.StatusCode)
	}

	dd.logger.Info("Event sent to DataDog", map[string]interface{}{
		"title": event.Title,
		"status": resp.StatusCode,
	})

	return nil
}

// SendCustomMetric sends a single custom metric to DataDog
func (dd *DataDogClient) SendCustomMetric(ctx context.Context, name string, value float64, tags []string) error {
	timestamp := float64(time.Now().Unix())

	metric := DataDogMetric{
		Metric: dd.addNamespace(name),
		Points: [][]float64{{timestamp, value}},
		Type:   "gauge",
		Tags:   append(tags, "source:analytics-service"),
	}

	payload := DataDogMetricsPayload{Series: []DataDogMetric{metric}}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dd.baseURL+"/api/v1/series", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create metric request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", dd.config.APIKey)

	resp, err := dd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metric: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("DataDog metric submission failed: status %d", resp.StatusCode)
	}

	dd.logger.Info("Custom metric sent to DataDog", map[string]interface{}{
		"metric": name,
		"value": value,
		"status": resp.StatusCode,
	})

	return nil
}

// GetHealth returns DataDog integration health status
func (dd *DataDogClient) GetHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"service": "datadog",
		"status":  "unknown",
		"base_url": dd.baseURL,
		"timestamp": time.Now(),
	}

	// Test connection
	if err := dd.TestConnection(ctx); err != nil {
		health["status"] = "unhealthy"
		health["error"] = err.Error()
	} else {
		health["status"] = "healthy"
	}

	return health
}

// addNamespace adds namespace prefix to metric names
func (dd *DataDogClient) addNamespace(metricName string) string {
	if dd.config.Namespace != "" {
		return fmt.Sprintf("%s.%s", dd.config.Namespace, metricName)
	}
	return fmt.Sprintf("analytics.%s", metricName)
}

// CreateAlert creates a monitor/alert in DataDog
func (dd *DataDogClient) CreateAlert(ctx context.Context, alertConfig map[string]interface{}) error {
	jsonData, err := json.Marshal(alertConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal alert config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", dd.baseURL+"/api/v1/monitor", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create alert request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", dd.config.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", dd.config.AppKey)

	resp, err := dd.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("DataDog alert creation failed: status %d", resp.StatusCode)
	}

	dd.logger.Info("Alert created in DataDog", map[string]interface{}{
		"status": resp.StatusCode,
	})

	return nil
}