package database

import (
	"context"
	"fmt"
	"time"

	"reciprocal-clubs-backend/pkg/shared/config"
	"reciprocal-clubs-backend/pkg/shared/logging"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database represents a database connection with GORM
type Database struct {
	*gorm.DB
	config *config.DatabaseConfig
	logger logging.Logger
}

// BaseModel is the base model for all database models
type BaseModel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ClubID    uint      `json:"club_id" gorm:"index;not null"` // Multi-tenant partitioning
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SoftDeleteModel extends BaseModel with soft delete capability
type SoftDeleteModel struct {
	BaseModel
	DeletedAt *time.Time `json:"deleted_at,omitempty" gorm:"index"`
}

// NewConnection creates a new database connection with GORM
func NewConnection(cfg *config.DatabaseConfig, logger logging.Logger) (*Database, error) {
	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{
		Logger: newGormLogger(logger),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	database := &Database{
		DB:     db,
		config: cfg,
		logger: logger,
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established", map[string]interface{}{
		"host":     cfg.Host,
		"port":     cfg.Port,
		"database": cfg.Database,
	})

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// HealthCheck performs a health check on the database
func (d *Database) HealthCheck(ctx context.Context) error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// Transaction executes a function within a database transaction
func (d *Database) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return d.WithContext(ctx).Transaction(fn)
}

// WithTenant returns a GORM DB instance scoped to a specific tenant (club)
func (d *Database) WithTenant(clubID uint) *gorm.DB {
	return d.Where("club_id = ?", clubID)
}

// WithContext returns a GORM DB instance with context
func (d *Database) WithContext(ctx context.Context) *gorm.DB {
	return d.DB.WithContext(ctx)
}

// Migrate runs database migrations for given models
func (d *Database) Migrate(models ...interface{}) error {
	d.logger.Info("Running database migrations", map[string]interface{}{
		"models_count": len(models),
	})

	for _, model := range models {
		if err := d.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}

	d.logger.Info("Database migrations completed successfully", nil)
	return nil
}

// Repository is the base repository interface
type Repository interface {
	Create(ctx context.Context, clubID uint, entity interface{}) error
	GetByID(ctx context.Context, clubID uint, id uint, entity interface{}) error
	Update(ctx context.Context, clubID uint, entity interface{}) error
	Delete(ctx context.Context, clubID uint, id uint, entity interface{}) error
	List(ctx context.Context, clubID uint, entities interface{}, filters map[string]interface{}) error
}

// BaseRepository provides common repository operations
type BaseRepository struct {
	db     *Database
	logger logging.Logger
}

// NewBaseRepository creates a new base repository
func NewBaseRepository(db *Database, logger logging.Logger) *BaseRepository {
	return &BaseRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new entity
func (r *BaseRepository) Create(ctx context.Context, clubID uint, entity interface{}) error {
	// Set club_id for multi-tenancy
	if setter, ok := entity.(interface{ SetClubID(uint) }); ok {
		setter.SetClubID(clubID)
	}

	if err := r.db.WithContext(ctx).Create(entity).Error; err != nil {
		r.logger.Error("Failed to create entity", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"entity":  fmt.Sprintf("%T", entity),
		})
		return fmt.Errorf("failed to create entity: %w", err)
	}

	r.logger.Debug("Entity created successfully", map[string]interface{}{
		"club_id": clubID,
		"entity":  fmt.Sprintf("%T", entity),
	})

	return nil
}

// GetByID retrieves an entity by ID
func (r *BaseRepository) GetByID(ctx context.Context, clubID uint, id uint, entity interface{}) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).First(entity, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("entity not found: id=%d, club_id=%d", id, clubID)
		}
		r.logger.Error("Failed to get entity by ID", map[string]interface{}{
			"error":   err.Error(),
			"id":      id,
			"club_id": clubID,
			"entity":  fmt.Sprintf("%T", entity),
		})
		return fmt.Errorf("failed to get entity: %w", err)
	}

	return nil
}

// Update updates an existing entity
func (r *BaseRepository) Update(ctx context.Context, clubID uint, entity interface{}) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).Save(entity).Error; err != nil {
		r.logger.Error("Failed to update entity", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"entity":  fmt.Sprintf("%T", entity),
		})
		return fmt.Errorf("failed to update entity: %w", err)
	}

	r.logger.Debug("Entity updated successfully", map[string]interface{}{
		"club_id": clubID,
		"entity":  fmt.Sprintf("%T", entity),
	})

	return nil
}

// Delete soft deletes an entity
func (r *BaseRepository) Delete(ctx context.Context, clubID uint, id uint, entity interface{}) error {
	if err := r.db.WithTenant(clubID).WithContext(ctx).Delete(entity, id).Error; err != nil {
		r.logger.Error("Failed to delete entity", map[string]interface{}{
			"error":   err.Error(),
			"id":      id,
			"club_id": clubID,
			"entity":  fmt.Sprintf("%T", entity),
		})
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	r.logger.Debug("Entity deleted successfully", map[string]interface{}{
		"id":      id,
		"club_id": clubID,
		"entity":  fmt.Sprintf("%T", entity),
	})

	return nil
}

// List retrieves entities with optional filters
func (r *BaseRepository) List(ctx context.Context, clubID uint, entities interface{}, filters map[string]interface{}) error {
	query := r.db.WithTenant(clubID).WithContext(ctx)

	// Apply filters
	for key, value := range filters {
		query = query.Where(fmt.Sprintf("%s = ?", key), value)
	}

	if err := query.Find(entities).Error; err != nil {
		r.logger.Error("Failed to list entities", map[string]interface{}{
			"error":   err.Error(),
			"club_id": clubID,
			"filters": filters,
		})
		return fmt.Errorf("failed to list entities: %w", err)
	}

	return nil
}

// gormLogger wraps our logger for GORM
type gormLogger struct {
	logger logging.Logger
}

func newGormLogger(logger logging.Logger) logger.Interface {
	return &gormLogger{logger: logger}
}

func (l *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Info(msg, map[string]interface{}{"data": data})
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Warn(msg, map[string]interface{}{"data": data})
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Error(msg, map[string]interface{}{"data": data})
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := map[string]interface{}{
		"elapsed": elapsed,
		"sql":     sql,
		"rows":    rows,
	}

	if err != nil {
		fields["error"] = err.Error()
		l.logger.Error("SQL query failed", fields)
	} else {
		l.logger.Debug("SQL query executed", fields)
	}
}