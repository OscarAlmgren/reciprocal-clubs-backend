package integrations

import (
	"context"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// IntegrationsConfig holds configuration for all external integrations
type IntegrationsConfig struct {
	ElasticSearch *ElasticSearchConfig `json:"elasticsearch"`
	DataDog       *DataDogConfig       `json:"datadog"`
	Grafana       *GrafanaConfig       `json:"grafana"`
	BigQuery      *BigQueryConfig      `json:"bigquery"`
	S3            *S3Config            `json:"s3"`
}

// ElasticSearchConfig holds Elasticsearch configuration
type ElasticSearchConfig struct {
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Index    string `json:"index"`
}

// DataDogConfig holds DataDog configuration
type DataDogConfig struct {
	APIKey    string `json:"api_key"`
	AppKey    string `json:"app_key"`
	Site      string `json:"site"`
	Namespace string `json:"namespace"`
}

// GrafanaConfig holds Grafana configuration
type GrafanaConfig struct {
	URL      string `json:"url"`
	APIKey   string `json:"api_key"`
	OrgID    int    `json:"org_id"`
}

// BigQueryConfig holds Google BigQuery configuration
type BigQueryConfig struct {
	ProjectID      string `json:"project_id"`
	DatasetID      string `json:"dataset_id"`
	CredentialsPath string `json:"credentials_path"`
}

// S3Config holds AWS S3 configuration for data export
type S3Config struct {
	Region     string `json:"region"`
	Bucket     string `json:"bucket"`
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
	PathPrefix string `json:"path_prefix"`
}

// AnalyticsIntegrations manages all external analytics integrations
type AnalyticsIntegrations struct {
	ElasticSearch *ElasticSearchClient
	DataDog       *DataDogClient
	Grafana       *GrafanaClient
	BigQuery      *BigQueryClient
	S3            *S3Client
	logger        logging.Logger
}

// NewAnalyticsIntegrations creates a new analytics integrations manager
func NewAnalyticsIntegrations(config *IntegrationsConfig, logger logging.Logger) *AnalyticsIntegrations {
	integrations := &AnalyticsIntegrations{
		logger: logger,
	}

	// Initialize ElasticSearch client
	if config.ElasticSearch != nil {
		integrations.ElasticSearch = NewElasticSearchClient(config.ElasticSearch, logger)
	}

	// Initialize DataDog client
	if config.DataDog != nil {
		integrations.DataDog = NewDataDogClient(config.DataDog, logger)
	}

	// Initialize Grafana client
	if config.Grafana != nil {
		integrations.Grafana = NewGrafanaClient(config.Grafana, logger)
	}

	// Initialize BigQuery client
	if config.BigQuery != nil {
		integrations.BigQuery = NewBigQueryClient(config.BigQuery, logger)
	}

	// Initialize S3 client
	if config.S3 != nil {
		integrations.S3 = NewS3Client(config.S3, logger)
	}

	return integrations
}

// ValidateConfig validates all integration configurations
func (ai *AnalyticsIntegrations) ValidateConfig() error {
	if ai.ElasticSearch != nil {
		if err := ai.ElasticSearch.ValidateConfig(); err != nil {
			return err
		}
	}

	if ai.DataDog != nil {
		if err := ai.DataDog.ValidateConfig(); err != nil {
			return err
		}
	}

	if ai.Grafana != nil {
		if err := ai.Grafana.ValidateConfig(); err != nil {
			return err
		}
	}

	if ai.BigQuery != nil {
		if err := ai.BigQuery.ValidateConfig(); err != nil {
			return err
		}
	}

	if ai.S3 != nil {
		if err := ai.S3.ValidateConfig(); err != nil {
			return err
		}
	}

	return nil
}

// TestConnections tests connectivity to all configured integrations
func (ai *AnalyticsIntegrations) TestConnections(ctx context.Context) error {
	if ai.ElasticSearch != nil {
		if err := ai.ElasticSearch.TestConnection(ctx); err != nil {
			ai.logger.Warn("ElasticSearch connection failed", map[string]interface{}{"error": err.Error()})
		}
	}

	if ai.DataDog != nil {
		if err := ai.DataDog.TestConnection(ctx); err != nil {
			ai.logger.Warn("DataDog connection failed", map[string]interface{}{"error": err.Error()})
		}
	}

	if ai.Grafana != nil {
		if err := ai.Grafana.TestConnection(ctx); err != nil {
			ai.logger.Warn("Grafana connection failed", map[string]interface{}{"error": err.Error()})
		}
	}

	if ai.BigQuery != nil {
		if err := ai.BigQuery.TestConnection(ctx); err != nil {
			ai.logger.Warn("BigQuery connection failed", map[string]interface{}{"error": err.Error()})
		}
	}

	if ai.S3 != nil {
		if err := ai.S3.TestConnection(ctx); err != nil {
			ai.logger.Warn("S3 connection failed", map[string]interface{}{"error": err.Error()})
		}
	}

	return nil
}

// ExportData exports analytics data to configured external systems
func (ai *AnalyticsIntegrations) ExportData(ctx context.Context, data interface{}, exportType string) error {
	switch exportType {
	case "elasticsearch":
		if ai.ElasticSearch != nil {
			return ai.ElasticSearch.IndexData(ctx, data)
		}
	case "bigquery":
		if ai.BigQuery != nil {
			return ai.BigQuery.InsertData(ctx, data)
		}
	case "s3":
		if ai.S3 != nil {
			return ai.S3.UploadData(ctx, data)
		}
	}

	ai.logger.Warn("Export type not configured or unsupported", map[string]interface{}{"export_type": exportType})
	return nil
}

// SendMetrics sends metrics to configured monitoring systems
func (ai *AnalyticsIntegrations) SendMetrics(ctx context.Context, metrics map[string]interface{}) error {
	if ai.DataDog != nil {
		if err := ai.DataDog.SendMetrics(ctx, metrics); err != nil {
			ai.logger.Error("Failed to send metrics to DataDog", map[string]interface{}{"error": err.Error()})
		}
	}

	return nil
}

// CreateDashboard creates or updates dashboards in configured systems
func (ai *AnalyticsIntegrations) CreateDashboard(ctx context.Context, dashboardConfig map[string]interface{}) error {
	if ai.Grafana != nil {
		if err := ai.Grafana.CreateDashboard(ctx, dashboardConfig); err != nil {
			ai.logger.Error("Failed to create Grafana dashboard", map[string]interface{}{"error": err.Error()})
			return err
		}
	}

	return nil
}

// GetHealth returns health status of all integrations
func (ai *AnalyticsIntegrations) GetHealth(ctx context.Context) map[string]interface{} {
	health := map[string]interface{}{
		"timestamp": time.Now(),
		"integrations": map[string]interface{}{},
	}

	integrations := health["integrations"].(map[string]interface{})

	if ai.ElasticSearch != nil {
		integrations["elasticsearch"] = ai.ElasticSearch.GetHealth(ctx)
	}

	if ai.DataDog != nil {
		integrations["datadog"] = ai.DataDog.GetHealth(ctx)
	}

	if ai.Grafana != nil {
		integrations["grafana"] = ai.Grafana.GetHealth(ctx)
	}

	if ai.BigQuery != nil {
		integrations["bigquery"] = ai.BigQuery.GetHealth(ctx)
	}

	if ai.S3 != nil {
		integrations["s3"] = ai.S3.GetHealth(ctx)
	}

	return health
}