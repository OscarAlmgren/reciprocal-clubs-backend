package repository

import (
	"context"

	"reciprocal-clubs-backend/pkg/shared/database"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/governance-service/internal/models"

	"gorm.io/gorm"
)

// Repository handles database operations for governance
type Repository struct {
	*database.BaseRepository
	db     *gorm.DB
	logger logging.Logger
}

// NewRepository creates a new governance repository
func NewRepository(db *gorm.DB, logger logging.Logger) *Repository {
	// Convert gorm.DB to database.Database if needed
	// For now, we'll work with the gorm.DB directly
	return &Repository{
		db:     db,
		logger: logger,
	}
}

// Example operations - replace with actual governance operations

// CreateExample creates a new example
func (r *Repository) CreateExample(ctx context.Context, example *models.Example) error {
	if err := r.db.WithContext(ctx).Create(example).Error; err != nil {
		r.logger.Error("Failed to create example", map[string]interface{}{
			"error": err.Error(),
			"name":  example.Name,
		})
		return err
	}

	r.logger.Info("Example created successfully", map[string]interface{}{
		"example_id": example.ID,
		"name":       example.Name,
	})

	return nil
}

// GetExample retrieves an example by ID
func (r *Repository) GetExample(ctx context.Context, id uint) (*models.Example, error) {
	var example models.Example
	if err := r.db.WithContext(ctx).First(&example, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, err
		}
		r.logger.Error("Failed to get example", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return nil, err
	}

	return &example, nil
}

// UpdateExample updates an existing example
func (r *Repository) UpdateExample(ctx context.Context, example *models.Example) error {
	if err := r.db.WithContext(ctx).Save(example).Error; err != nil {
		r.logger.Error("Failed to update example", map[string]interface{}{
			"error":      err.Error(),
			"example_id": example.ID,
		})
		return err
	}

	r.logger.Info("Example updated successfully", map[string]interface{}{
		"example_id": example.ID,
	})

	return nil
}

// DeleteExample deletes an example
func (r *Repository) DeleteExample(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Example{}, id).Error; err != nil {
		r.logger.Error("Failed to delete example", map[string]interface{}{
			"error": err.Error(),
			"id":    id,
		})
		return err
	}

	r.logger.Info("Example deleted successfully", map[string]interface{}{
		"example_id": id,
	})

	return nil
}

// ListExamples retrieves all examples
func (r *Repository) ListExamples(ctx context.Context) ([]models.Example, error) {
	var examples []models.Example
	if err := r.db.WithContext(ctx).Find(&examples).Error; err != nil {
		r.logger.Error("Failed to list examples", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	return examples, nil
}